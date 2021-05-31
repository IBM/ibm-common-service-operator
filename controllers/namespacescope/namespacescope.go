//
// Copyright 2021 IBM Corporation
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

package nss

import (
	"context"
	"time"

	gset "github.com/deckarep/golang-set"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"

	nssv1 "github.com/IBM/ibm-namespace-scope-operator/api/v1"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
)

var (
	apiGroupVersion = "operator.ibm.com/v1"
	Kinds           = []string{"NamespaceScope"}
	CRList          = []string{"common-service", "nss-odlm-scope"}
	sourceCR        = "common-service"
	targetCR        = "nss-odlm-scope"
)

var ctx = context.Background()

// SyncUpCR syncs up the namespace members in source CR and target CR
func SyncUpCR(bs *bootstrap.Bootstrap) {
	for {
		// wait for nss CRD
		for _, kind := range Kinds {
			if err := bs.WaitResourceReady(apiGroupVersion, kind); err != nil {
				klog.Errorf("Failed to wait for resource ready with kind %s, apiGroupVersion: %s", kind, apiGroupVersion)
				continue
			}
		}

		// wait for source and target CR
		for _, cr := range CRList {
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
		sourceNsScopeKey := types.NamespacedName{Name: sourceCR, Namespace: bs.CSData.MasterNs}
		if err := bs.Reader.Get(ctx, sourceNsScopeKey, sourceNsScope); err != nil {
			klog.Errorf("Failed to get NSS CR %s: %v, retry again", sourceNsScopeKey.String(), err)
			continue
		}

		targetNsScope := &nssv1.NamespaceScope{}
		targetNsScopeKey := types.NamespacedName{Name: targetCR, Namespace: bs.CSData.MasterNs}
		if err := bs.Reader.Get(ctx, targetNsScopeKey, targetNsScope); err != nil {
			klog.Errorf("Failed to get NSS CR %s: %v, retry again", targetNsScopeKey.String(), err)
			continue
		}

		sourceNsSet := gset.NewSet()
		targetNsSet := gset.NewSet()
		// we can't convert []T to []interface{} directly in Go, have to add it to set by loop
		for _, ns := range sourceNsScope.Spec.NamespaceMembers {
			sourceNsSet.Add(ns)
		}
		for _, ns := range targetNsScope.Spec.NamespaceMembers {
			sourceNsSet.Add(ns)
			targetNsSet.Add(ns)
		}

		// sync up when namepsace in source CR is different from target CR
		if !sourceNsSet.Equal(targetNsSet) {
			sourcenNsMems := sourceNsSet.ToSlice()
			var targetNsMems []string
			for _, ns := range sourcenNsMems {
				targetNsMems = append(targetNsMems, ns.(string))
			}
			targetNsScope.Spec.NamespaceMembers = targetNsMems
			if err := bs.Client.Update(ctx, targetNsScope); err != nil {
				klog.Errorf("Failed to update NSS resource %s: %v, retry again", targetNsScopeKey.String(), err)
				continue
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
