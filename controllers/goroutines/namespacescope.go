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
	"strings"
	"time"

	gset "github.com/deckarep/golang-set"
	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
)

var ctx = context.Background()

func SyncUpNSSConfigMap(bs *bootstrap.Bootstrap) {
	for {
		//get ConfigMap of namespace-scope
		nssConfigMap := &corev1.ConfigMap{}
		namespaceScopeKey := types.NamespacedName{Name: "namespace-scope", Namespace: bs.CSData.MasterNs}
		if err := bs.Reader.Get(ctx, namespaceScopeKey, nssConfigMap); err != nil {
			if errors.IsNotFound(err) {
				// Backward compatible upgrade from version 3.4.x and fresh installation in CP 3.0
				if err := bs.CreateNsScopeConfigmap(); err != nil {
					klog.Errorf("Failed to create Namespace Scope ConfigMap: %v, retry in 5 seconds", err)
					time.Sleep(5 * time.Second)
					continue
				}
			} else {
				klog.Errorf("Failed to get configmap %s: %v, retry in 10 seconds", namespaceScopeKey.String(), err)
				time.Sleep(10 * time.Second)
				continue
			}
		} else {
			// get targetNamespace from OperatorGroup
			existOG := &olmv1.OperatorGroupList{}
			if err := bs.Reader.List(context.TODO(), existOG, &client.ListOptions{Namespace: bs.CSData.MasterNs}); err != nil {
				klog.Errorf("Failed to get OperatorGroup in %s namespace: %v, retry in 10 seconds", bs.CSData.MasterNs, err)
				time.Sleep(10 * time.Second)
				continue
			}
			if len(existOG.Items) != 1 {
				klog.Errorf("The number of OperatorGroup in %s namespace is incorrect, Only one OperatorGroup is allowed in one namespace", bs.CSData.MasterNs)
				time.Sleep(10 * time.Second)
				continue
			}

			originalOG := &existOG.Items[0]
			originalOGNs := originalOG.Status.Namespaces

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
				// Consider restart the pod if it is necessary
			}
		}

	}

}
