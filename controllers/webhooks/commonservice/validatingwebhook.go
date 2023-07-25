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

package operandrequest

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	// certmanagerv1alpha1 "github.com/ibm/ibm-cert-manager-operator/apis/certmanager/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	operatorv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
)

// +kubebuilder:webhook:path=/validate-operator-ibm-com-v3-commonservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=operator.ibm.com,resources=commonservice,verbs=create;update,versions=v3,name=vcommonservice.kb.io,admissionReviewVersions=v1

// CommonServiceDefaulter points to correct ServiceNamespace
type Defaulter struct {
	Reader  client.Reader
	Client  client.Client
	decoder *admission.Decoder
}

// podAnnotator adds an annotation to every incoming pods.
func (r *Defaulter) Handle(ctx context.Context, req admission.Request) admission.Response {
	klog.Infof("Webhook is invoked by Commonservice %s/%s", req.AdmissionRequest.Namespace, req.AdmissionRequest.Name)

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

	cs := &operatorv3.CommonService{}

	err := r.decoder.Decode(req, cs)
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

	// admission.PatchResponse generates a Response containing patches.
	return admission.Allowed("")
}

func (r *Defaulter) CheckNamespace(name string) (bool, error) {
	watchNamespaces := util.GetWatchNamespace()
	denied := false
	if name == "" {
		return false, nil
	}
	// in cluster scope
	if len(watchNamespaces) == 0 {
		ctx := context.Background()
		ns := &corev1.Namespace{}
		nsKey := types.NamespacedName{
			Name: name,
		}
		// check if this namespace exist
		if err := r.Client.Get(ctx, nsKey, ns); err != nil {
			klog.Errorf("Failed to get namespace %v: %v", name, err)
			return true, err

		}
		// if it is not cluster scope
	} else if len(watchNamespaces) != 0 && !util.Contains(strings.Split(watchNamespaces, ","), name) {
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
