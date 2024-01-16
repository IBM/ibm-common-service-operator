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
	"time"

	"k8s.io/klog"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
)

// Cleanup_Keycloak_Cert will delete Keycloak Certificate when OperandConfig is updated to new version
func CleanupResources(bs *bootstrap.Bootstrap) {
	for {
		// wait ODLM OperandConfig CR resources
		if err := bs.WaitResourceReady("operator.ibm.com/v1alpha1", "OperandConfig"); err != nil {
			klog.Error("Failed to wait for resource ready with kind: OperandConfig, apiGroupVersion: operator.ibm.com/v1alpha1")
			time.Sleep(5 * time.Second)
			continue
		}

		opcon, err := bs.GetOperandConfig(ctx, "common-service", bs.CSData.ServicesNs)
		if err != nil || opcon == nil {
			time.Sleep(5 * time.Second)
			continue
		}

		// check if OperandConfig's annotation version is updated
		if opcon.Annotations != nil {
			v1IsLarger, convertErr := util.CompareVersion(bs.CSData.Version, opcon.Annotations["version"])
			if convertErr != nil {
				klog.Errorf("Failed to convert version for OperandConfig: %v", convertErr)
				time.Sleep(5 * time.Second)
				continue
			}
			// if OperandConfig's version is updated to the same as CS version or larger, delete Keycloak Certificate
			if !v1IsLarger {
				// check if cert-manager CRD does not exist, then skip cert-manager related controllers initialization
				exist, err := bs.CheckCRD(constant.CertManagerAPIGroupVersionV1, "Certificate")
				if err != nil {
					klog.Errorf("Failed to check if cert-manager CRD exists: %v", err)
					time.Sleep(5 * time.Second)
					continue
				}
				if !exist && err == nil {
					klog.Infof("cert-manager CRD does not exist, skip deleting Keycloak Certificate %s", constant.KeycloakCert)
				} else if exist && err == nil {
					if err := bs.DeleteFromYaml(constant.KeycloakCertTemplate, bs.CSData); err != nil {
						klog.Errorf("Failed to delete Keycloak Certificate %s: %v", constant.KeycloakCert, err)
						time.Sleep(5 * time.Second)
						continue
					}
				}
			}
		}

		time.Sleep(2 * time.Minute)
	}
}
