//
// Copyright 2023 IBM Corporation
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
	"os"

	"k8s.io/klog"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
)

// CreateUpdateConfig deploys config builder for global cpp configmap
func CleanUpWebhook(finish chan<- struct{}, bs *bootstrap.Bootstrap) {
	validatingWebhookConfiguration := bootstrap.Resource{
		Name:    "ibm-common-service-validating-webhook-" + bs.CSData.OperatorNs,
		Version: "v1",
		Group:   "admissionregistration.k8s.io",
		Kind:    "ValidatingWebhookConfiguration",
		Scope:   "clusterScope",
	}

	mutatingWebhookConfiguration := bootstrap.Resource{
		Name:    "ibm-operandrequest-webhook-configuration-" + bs.CSData.OperatorNs,
		Version: "v1",
		Group:   "admissionregistration.k8s.io",
		Kind:    "MutatingWebhookConfiguration",
		Scope:   "clusterScope",
	}

	webhookService := bootstrap.Resource{
		Name:    "webhook-service",
		Version: "v1",
		Group:   "",
		Kind:    "Service",
		Scope:   "namespaceScope",
	}

	if err := bs.Cleanup(bs.CSData.OperatorNs, &validatingWebhookConfiguration); err != nil {
		klog.Errorf("Failed to cleanup validatingWebhookConfig: %v", err)
		os.Exit(1)
	}

	if err := bs.Cleanup(bs.CSData.OperatorNs, &mutatingWebhookConfiguration); err != nil {
		klog.Errorf("Failed to cleanup mutatingWebhookConfiguration: %v", err)
		os.Exit(1)
	}

	if err := bs.Cleanup(bs.CSData.OperatorNs, &webhookService); err != nil {
		klog.Errorf("Failed to cleanup webhookService: %v", err)
		os.Exit(1)
	}

	close(finish)

}
