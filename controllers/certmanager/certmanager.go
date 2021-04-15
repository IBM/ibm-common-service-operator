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

package certmanager

import (
	"time"

	utilwait "k8s.io/apimachinery/pkg/util/wait"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/klog"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
)

var (
	DeployCRs       = []string{CSSSIssuer, CSCACert, CSCAIssuer}
	apiGroupVersion = "certmanager.k8s.io/v1alpha1"
	Kinds           = []string{"Issuer", "Certificate"}
	MasterNamespace string
	placeholder     = "placeholder"
)

// DeployCR deploys CR certificate and issuer when their CRDs are ready
func DeployCR(bs *bootstrap.Bootstrap) {
	for _, kind := range Kinds {
		if err := waitResourceReady(bs, apiGroupVersion, kind); err != nil {
			klog.Errorf("Failed to wait for resource ready with kind %s, apiGroupVersion: %s", kind, apiGroupVersion)
		}
	}

	for _, cr := range DeployCRs {
		for {
			done := deployResource(bs, cr)
			if done {
				break
			}
			time.Sleep(10 * time.Second)
		}

	}
}

func waitResourceReady(bs *bootstrap.Bootstrap, apiGroupVersion string, kind string) error {
	dc := discovery.NewDiscoveryClientForConfigOrDie(bs.Config)
	if err := utilwait.PollImmediateInfinite(time.Second*10, func() (done bool, err error) {
		exist, err := bs.ResourceExists(dc, apiGroupVersion, kind)
		if err != nil {
			return exist, err
		}
		if !exist {
			klog.Infof("waiting for resource ready with kind: %s, apiGroupVersion: %s", kind, apiGroupVersion)
		}
		return exist, nil
	}); err != nil {
		return err
	}
	return nil
}

func deployResource(bs *bootstrap.Bootstrap, cr string) bool {
	if err := utilwait.PollImmediateInfinite(time.Second*10, func() (done bool, err error) {
		err = bs.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cr, placeholder, bs.MasterNamespace)))
		if err != nil {
			return false, err
		}
		return true, nil
	}); err != nil {
		klog.Errorf("Failed to create Certmanager resource: %v, retry in 30 seconds", err)
		return false
	}
	return true
}
