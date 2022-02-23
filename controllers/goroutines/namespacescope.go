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
	"reflect"
	"time"

	gset "github.com/deckarep/golang-set"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"

	nssv1 "github.com/IBM/ibm-namespace-scope-operator/api/v1"
	ssv1 "github.com/IBM/ibm-secretshare-operator/api/v1"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
)

var ctx = context.Background()

// SyncUpNSSCR syncs up the namespace members in source CR and target CR
func SyncUpNSSCR(bs *bootstrap.Bootstrap) {
	for {
		// wait for nss CRD
		for _, kind := range NSSKinds {
			if err := bs.WaitResourceReady(OperatorAPIGroupVersion, kind); err != nil {
				klog.Errorf("Failed to wait for resource ready with kind %s, apiGroupVersion: %s", kind, OperatorAPIGroupVersion)
				continue
			}
		}

		// wait for source and target CR
		for _, cr := range NSSCRList {
			for {
				ready := waitCRReady(bs, cr, bs.CSData.MasterNs)
				if ready {
					break
				}
				time.Sleep(10 * time.Second)
			}
		}

		// fetch the source and target NSS CR
		sourceNsScope := &nssv1.NamespaceScope{}
		sourceNsScopeKey := types.NamespacedName{Name: NSSSourceCR, Namespace: bs.CSData.MasterNs}
		if err := bs.Reader.Get(ctx, sourceNsScopeKey, sourceNsScope); err != nil {
			klog.Errorf("Failed to get NSS CR %s: %v, retry again", sourceNsScopeKey.String(), err)
			continue
		}

		targetNsScope := &nssv1.NamespaceScope{}
		targetNsScopeKey := types.NamespacedName{Name: NSSTargetCR, Namespace: bs.CSData.MasterNs}
		if err := bs.Reader.Get(ctx, targetNsScopeKey, targetNsScope); err != nil {
			klog.Errorf("Failed to get NSS CR %s: %v, retry again", targetNsScopeKey.String(), err)
			continue
		}

		mergeNsSet := gset.NewSet()
		targetNsSet := gset.NewSet()
		// we can't convert []T to []interface{} directly in Go, have to add it to set by loop
		for _, ns := range sourceNsScope.Spec.NamespaceMembers {
			mergeNsSet.Add(ns)
		}
		for _, ns := range targetNsScope.Spec.NamespaceMembers {
			mergeNsSet.Add(ns)
			targetNsSet.Add(ns)
		}

		// sync up when namepsace in source CR is different from target CR
		if !mergeNsSet.Equal(targetNsSet) {
			mergeNsMems := mergeNsSet.ToSlice()
			var targetNsMems []string
			for _, ns := range mergeNsMems {
				targetNsMems = append(targetNsMems, ns.(string))
			}
			targetNsScope.Spec.NamespaceMembers = targetNsMems
			if err := bs.Client.Update(ctx, targetNsScope); err != nil {
				klog.Errorf("Failed to update NSS resource %s: %v, retry again", targetNsScopeKey.String(), err)
				continue
			}
		}

		// wait for secretshare CRD
		if err := bs.WaitResourceReady(SecretShareAPIGroupVersion, SecretShareKind); err != nil {
			klog.Errorf("Failed to wait for resource ready with kind %s, apiGroupVersion: %s", SecretShareKind, SecretShareAPIGroupVersion)
			continue
		}

		secretshare := &ssv1.SecretShare{}
		secretshareKey := types.NamespacedName{Name: SecretShareCppName, Namespace: bs.CSData.MasterNs}
		nsList := []ssv1.TargetNamespace{}
		scopeCR := &nssv1.NamespaceScope{}
		odlmNsScopeKey := types.NamespacedName{Name: "odlm-scope-managedby-odlm", Namespace: bs.CSData.MasterNs}
		if err := bs.Reader.Get(ctx, odlmNsScopeKey, scopeCR); err != nil {
			if errors.IsNotFound(err) {
				klog.Warningf("Not found NSS resource %s: %v, retry in 1 minute", targetNsScopeKey.String(), err)
				time.Sleep(1 * time.Minute)
				continue
			}
			klog.Errorf("Failed to get NSS resource %s: %v, retry again", targetNsScopeKey.String(), err)
			continue
		}
		for _, ns := range scopeCR.Status.ValidatedMembers {
			nsList = append(nsList, ssv1.TargetNamespace{Namespace: ns})
		}
		if err := bs.Reader.Get(ctx, secretshareKey, secretshare); err != nil && !errors.IsNotFound(err) {
			klog.Errorf("Failed to get SecretShare CR %s: %v, retry again", secretshareKey.String(), err)
			continue
		} else if errors.IsNotFound(err) {
			secretshare.ObjectMeta.Name = SecretShareCppName
			secretshare.ObjectMeta.Namespace = bs.CSData.MasterNs
			secretshare.Spec = ssv1.SecretShareSpec{
				Configmapshares: []ssv1.Configmapshare{
					{
						Configmapname: "ibm-cpp-config",
						Sharewith:     nsList,
					},
				},
			}
			if err := bs.Client.Create(ctx, secretshare); err != nil {
				klog.Errorf("Failed to create SecretShare CR %s: %v, retry again", secretshareKey.String(), err)
				continue
			}
		} else {
			originalss := secretshare.DeepCopy()
			secretshare.Spec = ssv1.SecretShareSpec{
				Configmapshares: []ssv1.Configmapshare{
					{
						Configmapname: "ibm-cpp-config",
						Sharewith:     nsList,
					},
				},
			}
			if !reflect.DeepEqual(originalss, secretshare) {
				if err := bs.Client.Update(ctx, secretshare); err != nil {
					klog.Errorf("Failed to update SecretShare CR %s: %v, retry again", secretshareKey.String(), err)
					continue
				}
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

func waitCRReady(bs *bootstrap.Bootstrap, nssKey, namespace string) bool {
	if err := utilwait.PollImmediateInfinite(time.Second*10, func() (done bool, err error) {
		nsScope := &nssv1.NamespaceScope{}
		nsScopeKey := types.NamespacedName{Name: nssKey, Namespace: namespace}
		err = bs.Reader.Get(ctx, nsScopeKey, nsScope)
		if err != nil {
			return false, err
		}
		return true, nil
	}); err != nil {
		klog.Errorf("waiting for NSS CR: %v, retry in 10 seconds", err)
		return false
	}
	return true
}
