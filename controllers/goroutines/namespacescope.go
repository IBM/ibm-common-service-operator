//
// Copyright 2022 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package goroutines

import (
	"context"
	"fmt"
	"strings"
	"time"

	gset "github.com/deckarep/golang-set"
	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nssv1 "github.com/IBM/ibm-namespace-scope-operator/api/v1"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	constant "github.com/IBM/ibm-common-service-operator/controllers/constant"
)

var ctx = context.Background()

func SyncUpNSSConfigMap(bs *bootstrap.Bootstrap) {
	for {
		//get ConfigMap of namespace-scope
		namespaceScopeKey := types.NamespacedName{Name: "namespace-scope", Namespace: bs.CSData.CPFSNs}
		nssConfigMap, err := getNsScopeConfigmap(bs, namespaceScopeKey)
		if err != nil || nssConfigMap == nil {
			continue
		}

		// get targetNamespace from OperatorGroup
		existOG := &olmv1.OperatorGroupList{}
		if err := bs.Reader.List(context.TODO(), existOG, &client.ListOptions{Namespace: bs.CSData.CPFSNs}); err != nil {
			klog.Errorf("Failed to get OperatorGroup in %s namespace: %v, retry in 10 seconds", bs.CSData.CPFSNs, err)
			time.Sleep(10 * time.Second)
			continue
		}
		if len(existOG.Items) != 1 {
			klog.Errorf("The number of OperatorGroup in %s namespace is incorrect, Only one OperatorGroup is allowed in one namespace", bs.CSData.CPFSNs)
			time.Sleep(10 * time.Second)
			continue
		}

		originalOG := &existOG.Items[0]
		originalOGNs := originalOG.Status.Namespaces

		// get NSS CR common-service
		nssKey := types.NamespacedName{Name: "common-service", Namespace: bs.CSData.CPFSNs}
		nssCR, err := getNSSCR(bs, nssKey)
		// Single or All Namespaces Mode
		if err == nil && nssCR == nil {
			// get NamespaceScope from ConfigMap
			originalNSSCMNs := strings.Split(nssConfigMap.Data["namespaces"], ",")

			OGNsSet := gset.NewSet()
			NSSCMNsSet := gset.NewSet()
			mergeNsSet := gset.NewSet()
			for _, ns := range originalOGNs {
				mergeNsSet.Add(ns)
				OGNsSet.Add(ns)
			}
			for _, ns := range originalNSSCMNs {
				mergeNsSet.Add(ns)
				NSSCMNsSet.Add(ns)
			}

			// if the existing version is empty or less than 4.0.0, NSS ConfigMap value will be copied to OperatorGroup.
			// Otherwise, NSS ConfigMap's value will not impact OperatorGroup
			v1IsLarger, convertErr := util.CompareVersion("4.0.0", nssConfigMap.GetAnnotations()["version"])
			if convertErr != nil {
				klog.Errorf("Failed to compare the version in ConfigMap %s: %v, retry again in 10 seconds", namespaceScopeKey.String(), err)
				time.Sleep(10 * time.Second)
				continue
			}
			// only happened during upgrade from Cloud Pak 2.0 to Cloud Pak 3.0
			if !mergeNsSet.Equal(OGNsSet) && v1IsLarger {
				mergeNsMems := mergeNsSet.ToSlice()
				var targetNsMems []string
				for _, ns := range mergeNsMems {
					targetNsMems = append(targetNsMems, ns.(string))
				}
				originalOG.Spec.TargetNamespaces = targetNsMems
				if err := bs.Client.Update(ctx, originalOG); err != nil {
					klog.Errorf("Failed to update OperatorGroup %s/%s: %v, retry again in 10 seconds", originalOG.GetNamespace(), originalOG.GetName(), err)
					time.Sleep(10 * time.Second)
					continue
				}

				// tag version in ConfigMap whenever the OperatorGroup is updated due to the synchronization in upgrade.
				// It avoids OperatorGroup's targetNamespace manipulation via NSS ConfigMap after upgrade
				if nssConfigMap.GetAnnotations() == nil {
					nssConfigMap.SetAnnotations(make(map[string]string))
				}
				nssConfigMap.Annotations["version"] = bs.CSData.Version
				if err := bs.Client.Update(ctx, nssConfigMap); err != nil {
					klog.Errorf("Failed to update ConfigMap %s: %v, retry again in 10 seconds", namespaceScopeKey.String(), err)
					time.Sleep(10 * time.Second)
					continue
				}
			} else if !OGNsSet.Equal(NSSCMNsSet) { // This helps to seed targetNamespaces from OperatorGroup into NSS ConfigMap, it will not happen during upgrade
				OGNsMems := OGNsSet.ToSlice()
				var targetNsMems []string
				for _, ns := range OGNsMems {
					targetNsMems = append(targetNsMems, ns.(string))
				}
				nssConfigMap.Data["namespaces"] = strings.Join(targetNsMems[:], ",")
				if nssConfigMap.GetAnnotations() == nil {
					nssConfigMap.SetAnnotations(make(map[string]string))
				}
				nssConfigMap.Annotations["version"] = bs.CSData.Version
				if err := bs.Client.Update(ctx, nssConfigMap); err != nil {
					klog.Errorf("Failed to update ConfigMap %s: %v, retry again in 10 seconds", namespaceScopeKey.String(), err)
					time.Sleep(10 * time.Second)
					continue
				}
			}
		} else if nssCR != nil {
			// MultiNamespaces Mode

			// get NamespaceScope from ConfigMap
			originalNSSCRNs := nssCR.Spec.NamespaceMembers

			OGNsSet := gset.NewSet()
			NSSCrNsSet := gset.NewSet()
			mergeNsSet := gset.NewSet()
			for _, ns := range originalOGNs {
				mergeNsSet.Add(ns)
				OGNsSet.Add(ns)
			}
			for _, ns := range originalNSSCRNs {
				mergeNsSet.Add(ns)
				NSSCrNsSet.Add(ns)
			}

			// if the existing version is empty or less than 4.0.0, NSS NS members will be copied to OperatorGroup.
			// Otherwise, NSS CR NS member's value will not impact OperatorGroup
			v1IsLarger, convertErr := util.CompareVersion("4.0.0", nssCR.GetAnnotations()["version"])
			if convertErr != nil {
				klog.Errorf("Failed to compare the version in ConfigMap %s: %v, retry again in 10 seconds", nssKey.String(), err)
				time.Sleep(10 * time.Second)
				continue
			}
			// only happened during upgrade from Cloud Pak 2.0 to Cloud Pak 3.0
			if !mergeNsSet.Equal(OGNsSet) && v1IsLarger {
				mergeNsMems := mergeNsSet.ToSlice()
				var targetNsMems []string
				for _, ns := range mergeNsMems {
					targetNsMems = append(targetNsMems, ns.(string))
				}
				originalOG.Spec.TargetNamespaces = targetNsMems
				if err := bs.Client.Update(ctx, originalOG); err != nil {
					klog.Errorf("Failed to update OperatorGroup %s/%s: %v, retry again in 10 seconds", originalOG.GetNamespace(), originalOG.GetName(), err)
					time.Sleep(10 * time.Second)
					continue
				}

				// tag version in NSS CR whenever the OperatorGroup is updated due to the synchronization in upgrade.
				// It avoids OperatorGroup's targetNamespace manipulation via NSS ConfigMap after upgrade
				if nssCR.GetAnnotations() == nil {
					nssCR.SetAnnotations(make(map[string]string))
				}
				nssCR.Annotations["version"] = bs.CSData.Version
				if err := bs.Client.Update(ctx, nssCR); err != nil {
					klog.Errorf("Failed to update ConfigMap %s: %v, retry again in 10 seconds", nssKey.String(), err)
					time.Sleep(10 * time.Second)
					continue
				}
			} else if !OGNsSet.Equal(NSSCrNsSet) { // This helps to seed targetNamespaces from OperatorGroup into NSS CR, it will not happen during upgrade
				OGNsMems := OGNsSet.ToSlice()
				var targetNsMems []string
				for _, ns := range OGNsMems {
					targetNsMems = append(targetNsMems, ns.(string))
				}
				// https://freshman.tech/snippets/go/copy-slices/
				nssCR.Spec.NamespaceMembers = append(targetNsMems[:0:0], targetNsMems...)
				if nssCR.GetAnnotations() == nil {
					nssCR.SetAnnotations(make(map[string]string))
				}
				nssCR.Annotations["version"] = bs.CSData.Version
				if err := bs.Client.Update(ctx, nssCR); err != nil {
					klog.Errorf("Failed to update ConfigMap %s: %v, retry again in 10 seconds", nssKey.String(), err)
					time.Sleep(10 * time.Second)
					continue
				}
			}
		}
	}
}

func getNsScopeConfigmap(bs *bootstrap.Bootstrap, namespaceScopeKey types.NamespacedName) (*corev1.ConfigMap, error) {
	nssConfigMap := &corev1.ConfigMap{}
	if err := bs.Reader.Get(ctx, namespaceScopeKey, nssConfigMap); err != nil {
		if errors.IsNotFound(err) {
			// Backward compatible upgrade from version 3.4.x and fresh installation in CP 3.0
			if err := bs.CreateNsScopeConfigmap(); err != nil {
				klog.Errorf("Failed to create Namespace Scope ConfigMap: %v, retry in 5 seconds", err)
				time.Sleep(5 * time.Second)
				return nil, err
			}
			return nil, nil
		}
		klog.Errorf("Failed to get configmap %s: %v, retry in 10 seconds", namespaceScopeKey.String(), err)
		time.Sleep(10 * time.Second)
		return nil, err
	}
	return nssConfigMap, nil
}

func getNSSCR(bs *bootstrap.Bootstrap, nssKey types.NamespacedName) (*nssv1.NamespaceScope, error) {
	// check if NSS CR crd exist
	dc := discovery.NewDiscoveryClientForConfigOrDie(bs.Config)
	exist, err := bs.ResourceExists(dc, "operator.ibm.com/v1", "NamespaceScope")
	if err != nil {
		klog.Errorf("Failed to check resource with kind: %s, apiGroupVersion: %s, retry in 10 seconds", "NamespaceScope", "operator.ibm.com/v1")
		time.Sleep(10 * time.Second)
		return nil, err
	}
	if exist && len(strings.Split(bs.CSData.WatchNamespaces, ",")) > 1 {
		// get NSS CR common-service
		nssCR := &nssv1.NamespaceScope{}
		if err := bs.Reader.Get(ctx, nssKey, nssCR); err != nil {
			if errors.IsNotFound(err) {
				// Backward compatible upgrade from version 3.4.x and fresh installation in CP 3.0
				if err := bs.RenderTemplate(constant.NamespaceScopeCR, bs.CSData); err != nil {
					klog.Errorf("Failed to create Namespace Scope CR %s: %v, retry in 5 seconds", nssKey.String(), err)
					time.Sleep(5 * time.Second)
					return nil, err
				}
				return nil, fmt.Errorf("wait for the next iteration")
			}
			klog.Errorf("Failed to get Namespace Scope CR %s: %v, retry in 10 seconds", nssKey.String(), err)
			time.Sleep(10 * time.Second)
			return nil, err
		}
		return nssCR, nil
	}
	return nil, nil
}
