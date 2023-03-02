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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	operatorv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	"github.com/IBM/operand-deployment-lifecycle-manager/controllers/util"
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

	// check operatornamespace
	opNs := cs.Spec.OperatorNamespace
	deniedOpNs, err := r.CheckNamespace(string(opNs))
	if err != nil {
		return admission.Denied(fmt.Sprintf("Can't check operatorNamespace. Found error: %v", err))
	}
	if deniedOpNs {
		return admission.Denied(fmt.Sprintf("Operator Namespace: %v should be one of WATCH_NAMESPACE", opNs))
	}

	// check servicenamespace
	serviceNs := cs.Spec.ServicesNamespace
	deniedServiceNs, err := r.CheckNamespace(string(serviceNs))
	if err != nil {
		return admission.Denied(fmt.Sprintf("Can't check serviceNamespace. Found error: %v", err))
	}
	if deniedServiceNs {
		return admission.Denied(fmt.Sprintf("Service Namespace: %v should be one of WATCH_NAMESPACE", serviceNs))
	}

	// admission.PatchResponse generates a Response containing patches.
	return admission.Allowed("")
}

func (r *Defaulter) CheckNamespace(name string) (bool, error) {
	denied := false
	if name == "" {
		return false, nil
	}
	// in cluster scope
	if len(r.Bootstrap.CSData.WatchNamespaces) == 0 {
		ctx := context.Background()
		ns := &corev1.Namespace{}
		nsKey := types.NamespacedName{
			Name: name,
		}
		// check if this namespace exist
		if err := r.Client.Get(ctx, nsKey, ns); err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("Not found Namespace %v ", name)
				return true, err
			} else {
				klog.Errorf("Failed to get namespace %v: %v", name, err)
				return true, err
			}
		}
		// if it is not cluster scope
	} else if len(r.Bootstrap.CSData.WatchNamespaces) != 0 && !util.Contains(strings.Split(r.Bootstrap.CSData.WatchNamespaces, ","), name) {
		denied = true
	}
	return denied, nil
}

func (r *Defaulter) InjectDecoder(decoder *admission.Decoder) error {
	r.decoder = decoder
	return nil
}
