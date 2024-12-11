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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	"github.com/IBM/ibm-common-service-operator/v4/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/v4/controllers/common"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/constant"
)

const (
	mongodbPreloadCm   = "mongodb-preload-endpoint"
	mongodbStatefulSet = "icp-mongodb"
)

// Cleanup_Keycloak_Cert will delete Keycloak Certificate when OperandConfig is updated to new version
func CleanupKeycloakCert(bs *bootstrap.Bootstrap) {
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
				if !exist {
					klog.Infof("cert-manager CRD does not exist, skip deleting Keycloak Certificate %s", constant.KeycloakCert)
				} else if exist {
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

// Cleanup_MongoDB_Preload_CM will delete mongodb-preload-endpoint ConfigMap when icp-mongodb StatefulSet has owner reference
func CleanupMongodbPreloadCm(bs *bootstrap.Bootstrap) {
	for {
		// check if icp-mongodb StatefulSet exists
		statefulSet := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "StatefulSet",
				"metadata": map[string]interface{}{
					"name":      mongodbStatefulSet,
					"namespace": bs.CSData.ServicesNs,
				},
			},
		}
		if err := bs.Reader.Get(ctx, types.NamespacedName{Name: mongodbStatefulSet, Namespace: bs.CSData.ServicesNs}, statefulSet); err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("StatefulSet %s does not exist in %s, skip deleting %s ConfigMap", mongodbStatefulSet, bs.CSData.ServicesNs, mongodbPreloadCm)
				break
			} else {
				klog.Errorf("Failed to get StatefulSet %s: %v, retrying...", mongodbStatefulSet, err)
				time.Sleep(5 * time.Second)
				continue
			}
		}

		// check if icp-mongodb StatefulSet has owner reference, delete mongodb-preload-endpoint ConfigMap
		if (statefulSet.GetOwnerReferences() != nil) && (len(statefulSet.GetOwnerReferences()) > 0) {
			preloadCm := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":      mongodbPreloadCm,
						"namespace": bs.CSData.ServicesNs,
					},
				},
			}
			if err := bs.Client.Delete(ctx, preloadCm); err != nil {
				if errors.IsNotFound(err) {
					klog.Infof("ConfigMap %s does not exist in %s, skip deleting", mongodbPreloadCm, bs.CSData.ServicesNs)
					break
				}
				klog.Errorf("Failed to delete ConfigMap %s: %v, retrying...", mongodbPreloadCm, err)
				time.Sleep(5 * time.Second)
				continue
			}
			klog.Infof("ConfigMap %s in %s is deleted for the preparation of MongoDB migration.", mongodbPreloadCm, bs.CSData.ServicesNs)
			break
		}
		klog.Infof("StatefulSet %s does not have owner reference, skip deleting %s ConfigMap", mongodbStatefulSet, mongodbPreloadCm)
		break
	}
}

func CleanupResources(bs *bootstrap.Bootstrap) {
	go CleanupKeycloakCert(bs)
	go CleanupMongodbPreloadCm(bs)
}
