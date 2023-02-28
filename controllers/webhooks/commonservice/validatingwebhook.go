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

	// certmanagerv1alpha1 "github.com/ibm/ibm-cert-manager-operator/apis/certmanager/v1alpha1"

	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	operatorv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
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

	cs := &operatorv3.CommonService{}

	err := r.decoder.Decode(req, cs)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// get targetNamespace from OperatorGroup
	existOG := &olmv1.OperatorGroupList{}
	if err := r.Bootstrap.Reader.List(context.TODO(), existOG, &client.ListOptions{Namespace: r.Bootstrap.CSData.CPFSNs}); err != nil {
		klog.Errorf("Failed to get OperatorGroup in %s namespace: %v, retry in 10 seconds", r.Bootstrap.CSData.CPFSNs, err)
		return admission.Errored(http.StatusBadRequest, err)
	}
	if len(existOG.Items) != 1 {
		klog.Errorf("The number of OperatorGroup in %s namespace is incorrect, Only one OperatorGroup is allowed in one namespace", r.Bootstrap.CSData.CPFSNs)
		return admission.Errored(http.StatusBadRequest, err)
	}

	originalOG := &existOG.Items[0]
	originalOGNs := originalOG.Status.Namespaces

	if originalOGNs[0] == "" {
		return admission.Allowed("")
	}

	// check operatornamespace
	checkedopNs := false
	opNs := cs.Spec.OperatorNamespace
	for _, ns := range originalOGNs {
		if ns == string(opNs) {
			checkedopNs = true
		}
	}
	if !checkedopNs {
		return admission.Denied(fmt.Sprintf("Operator Namespace: %v should be one of target namespace in OpertorGroup", opNs))
	}

	// check operatornamespace
	checkedserviceNs := false
	serviceNs := cs.Spec.ServicesNamespace
	for _, ns := range originalOGNs {
		if ns == string(serviceNs) {
			checkedserviceNs = true
		}
	}
	if !checkedserviceNs {
		return admission.Denied(fmt.Sprintf("Service Namespace: %v should be one of target namespace in OpertorGroup", opNs))
	}

	// admission.PatchResponse generates a Response containing patches.
	return admission.Allowed("")
}

func (r *Defaulter) InjectDecoder(decoder *admission.Decoder) error {
	r.decoder = decoder
	return nil
}
