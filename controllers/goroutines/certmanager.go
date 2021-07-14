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

package goroutines

import (
	"context"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
)

// DeployCertManagerCR deploys CR certificate and issuer when their CRDs are ready
func DeployCertManagerCR(bs *bootstrap.Bootstrap) {
	deployedNs := bs.CSData.MasterNs
	if bs.MultiInstancesEnable {
		deployedNs = bs.CSData.ControlNs
	}
	for {
		if !getCertSubscription(bs.Reader, deployedNs) {
			time.Sleep(2 * time.Minute)
			continue
		}
		break
	}

	for _, kind := range CertManagerKinds {
		if err := bs.WaitResourceReady(CertManagerApiGroupVersion, kind); err != nil {
			klog.Errorf("Failed to wait for resource ready with kind %s, apiGroupVersion: %s", kind, CertManagerApiGroupVersion)
		}
	}

	for _, cr := range CertManagerCRs {
		for {
			done := bs.DeployResource(cr, placeholder)
			if done {
				break
			}
			time.Sleep(10 * time.Second)
		}

	}
}

// getCertSubscription return true if Cert Manager subscription found, otherwise return false
func getCertSubscription(r client.Reader, MasterNs string) bool {
	subName := "ibm-cert-manager-operator"
	subNs := MasterNs
	sub := &olmv1alpha1.Subscription{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: subName, Namespace: subNs}, sub)
	return err == nil
}
