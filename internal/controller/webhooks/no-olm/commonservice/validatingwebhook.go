//
// Copyright 2025 IBM Corporation
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

package commonservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	// certmanagerv1alpha1 "github.com/ibm/ibm-cert-manager-operator/apis/certmanager/v1alpha1"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/v4/internal/controller/common"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-operator-ibm-com-v1alpha1-operandrequest,mutating=true,failurePolicy=fail,sideEffects=None,groups=operator.ibm.com,resources=operandrequests,verbs=create;update,versions=v1alpha1,name=moperandrequest.kb.io,admissionReviewVersions=v1

// OperandRequestDefaulter points to correct RegistryNamespace
type Defaulter struct {
	*bootstrap.Bootstrap
	decoder *admission.Decoder
}

// podAnnotator adds an annotation to every incoming pods.
func (r *Defaulter) Handle(ctx context.Context, req admission.Request) admission.Response {
	klog.Infof("Webhook is invoked by Commonservice %s/%s", req.AdmissionRequest.Namespace, req.AdmissionRequest.Name)

	// Initialize the context for the tenant topology
	serviceNs := util.GetServicesNamespace(r.Reader)
	operatorNs, operatorNsErr := util.GetOperatorNamespace()
	if operatorNsErr != nil {
		return admission.Errored(http.StatusBadRequest, operatorNsErr)
	}

	catalogSourceName, catalogSourceNs := util.GetCatalogSource(constant.IBMCSPackage, operatorNs, r.Reader)
	if catalogSourceName == "" || catalogSourceNs == "" {
		err := fmt.Errorf("failed to get catalogsource")
		return admission.Errored(http.StatusBadRequest, err)
	}

	// handle the request from CommonService
	cs := &apiv3.CommonService{}
	csUnstrcuted := &unstructured.Unstructured{}

	err := r.decoder.Decode(req, cs)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Convert the request to unstructured
	err = r.decoder.Decode(req, csUnstrcuted)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// if it is master CommonService CR, check operator and services namespaces
	if req.AdmissionRequest.Name == constant.MasterCR && req.AdmissionRequest.Namespace == operatorNs {
		klog.Infof("Start to validate master CommonService CR")
		// check OperatorNamespace
		opNs := cs.Spec.OperatorNamespace
		deniedOpNs, err := r.CheckNamespace(string(opNs))
		if err != nil {
			return admission.Denied(fmt.Sprintf("Can't check operatorNamespace. Found error: %v", err))
		}
		if deniedOpNs {
			return admission.Denied(fmt.Sprintf("Operator Namespace: %v should be one of WATCH_NAMESPACE", opNs))
		}

		// check ServicesNamespace
		serviceNs := cs.Spec.ServicesNamespace
		deniedServiceNs, err := r.CheckNamespace(string(serviceNs))
		if err != nil {
			return admission.Denied(fmt.Sprintf("Can't check serviceNamespace. Found error: %v", err))
		}
		if deniedServiceNs {
			return admission.Denied(fmt.Sprintf("Services Namespace: %v should be one of WATCH_NAMESPACE", serviceNs))
		}

	} else {
		klog.Infof("Start to validate non-master CommonService CR")
		// check OperatorNamespace
		operatorNamespace := cs.Spec.OperatorNamespace
		deniedOperatorNs := r.CheckConfig(string(operatorNamespace), operatorNs)
		if deniedOperatorNs {
			return admission.Denied(fmt.Sprintf("Operator Namespace: %v is not allowed to be configured in namespace %v", operatorNamespace, req.AdmissionRequest.Namespace))
		}

		// check ServicesNamespace
		servicesNamespace := cs.Spec.ServicesNamespace
		deniedServicesNs := r.CheckConfig(string(servicesNamespace), serviceNs)
		if deniedServicesNs {
			return admission.Denied(fmt.Sprintf("Services Namespace: %v is not allowed to be configured in namespace %v", servicesNamespace, req.AdmissionRequest.Namespace))
		}

		// check CatalogName
		catalogName := cs.Spec.CatalogName
		deniedCatalog := r.CheckConfig(string(catalogName), catalogSourceName)
		if deniedCatalog {
			return admission.Denied(fmt.Sprintf("CatalogSource Name: %v is not allowed to be configured in namespace %v", catalogName, req.AdmissionRequest.Namespace))
		}

		// check CatalogNamespace
		catalogNamespace := cs.Spec.CatalogNamespace
		deniedCatalogNs := r.CheckConfig(string(catalogNamespace), catalogSourceNs)
		if deniedCatalogNs {
			return admission.Denied(fmt.Sprintf("CatalogSource Namespace: %v is not allowed to be configured in namespace %v", catalogNamespace, req.AdmissionRequest.Namespace))
		}
	}

	// check HugePageSetting
	deniedHugePage, err := r.HugePageSettingDenied(csUnstrcuted)
	if err != nil || deniedHugePage {
		return admission.Denied(fmt.Sprintf("HugePageSetting is invalid: %v", err))
	}

	// admission.PatchResponse generates a Response containing patches.
	return admission.Allowed("")
}

func (r *Defaulter) CheckNamespace(name string) (bool, error) {
	watchNamespaces := util.GetWatchNamespace()
	denied := false
	if name == "" {
		return false, nil
	}
	if len(watchNamespaces) != 0 && !util.Contains(strings.Split(watchNamespaces, ","), name) {
		denied = true
	}
	return denied, nil
}

func (r *Defaulter) CheckConfig(config, parameter string) bool {
	if config == "" {
		return false
	}
	return config != parameter
}

func (r *Defaulter) HugePageSettingDenied(cs *unstructured.Unstructured) (bool, error) {
	if hugespages := cs.Object["spec"].(map[string]interface{})["hugepages"]; hugespages != nil {
		if enable := hugespages.(map[string]interface{})["enable"]; enable != nil && enable.(bool) {
			hugePagesStruct, err := r.UnmarshalHugePages(hugespages)
			if err != nil {
				return true, fmt.Errorf("failed to unmarshal hugepages: %v", err)
			}

			for size, allocation := range hugePagesStruct.HugePagesSizes {
				// check if size is in the format of `hugepages-<size>`
				sizeSplit := strings.Split(size, "-")
				if len(sizeSplit) != 2 || sizeSplit[0] != "hugepages" {
					return true, fmt.Errorf("invalid hugepages size on prefix: %s, please specify in the format of `hugepages-<size>`", size)
				} else if _, err := resource.ParseQuantity(sizeSplit[1]); err != nil {
					return true, fmt.Errorf("invalid hugepages size on Quantity: %s, please specify in the format of `hugepages-<size>`", size)
				}
				if _, err := resource.ParseQuantity(allocation); err != nil && allocation != "" {
					return true, fmt.Errorf("invalid hugepages allocation: %s, please specify in the format of `hugepages-<size>: <allocation>`", allocation)
				}
			}
		}
	}

	return false, nil
}

// UnmarshalHugePages unmarshals the hugepages map to HugePages struct
func (r *Defaulter) UnmarshalHugePages(hugespages interface{}) (*apiv3.HugePages, error) {
	hugespagesBytes, err := json.Marshal(hugespages)
	if err != nil {
		return nil, err
	}

	hugePagesStruct := &apiv3.HugePages{}
	if err := json.Unmarshal(hugespagesBytes, hugePagesStruct); err != nil {
		return nil, err
	}

	hugespagesBytesSanitized, err := json.Marshal(util.SanitizeData(hugespages, "string", true))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(hugespagesBytesSanitized, &hugePagesStruct.HugePagesSizes); err != nil {
		return nil, err
	}

	return hugePagesStruct, nil
}

func (r *Defaulter) InjectDecoder(decoder *admission.Decoder) error {
	r.decoder = decoder
	return nil
}

func (r *Defaulter) SetupWebhookWithManager(mgr ctrl.Manager) error {

	mgr.GetWebhookServer().
		Register("/validate-operator-ibm-com-v3-commonservice",
			&webhook.Admission{Handler: r})

	return nil
}
