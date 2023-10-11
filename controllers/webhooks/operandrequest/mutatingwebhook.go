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
	"encoding/json"
	"net/http"
	"strings"

	// certmanagerv1alpha1 "github.com/ibm/ibm-cert-manager-operator/apis/certmanager/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-operator-ibm-com-v1alpha1-operandrequest,mutating=true,failurePolicy=ignore,sideEffects=None,groups=operator.ibm.com,resources=operandrequests,verbs=create;update,versions=v1alpha1,name=moperandrequest.kb.io,admissionReviewVersions=v1

// OperandRequestDefaulter points to correct RegistryNamespace
type Defaulter struct {
	Reader    client.Reader
	Client    client.Client
	IsDormant bool
	decoder   *admission.Decoder
}

// podAnnotator adds an annotation to every incoming pods.
func (r *Defaulter) Handle(ctx context.Context, req admission.Request) admission.Response {
	klog.Infof("Webhook is invoked by OperandRequest %s/%s", req.AdmissionRequest.Namespace, req.AdmissionRequest.Name)
	opreq := &odlm.OperandRequest{}

	err := r.decoder.Decode(req, opreq)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	copy := opreq.DeepCopy()

	if !r.IsDormant {
		r.Default(copy)
	}

	marshaledCopy, err := json.Marshal(copy)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	marshaledOpreq, err := json.Marshal(opreq)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(marshaledOpreq, marshaledCopy)
}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Defaulter) Default(instance *odlm.OperandRequest) {
	watchNamespaces := util.GetWatchNamespace()
	for i, req := range instance.Spec.Requests {
		if req.RegistryNamespace == "" {
			continue
		}
		regNs := req.RegistryNamespace
		isDefaulting := false
		// watchNamespace is empty in All namespace mode
		if len(watchNamespaces) == 0 {
			ctx := context.Background()
			ns := &corev1.Namespace{}
			nsKey := types.NamespacedName{
				Name: regNs,
			}
			if err := r.Client.Get(ctx, nsKey, ns); err != nil {
				if errors.IsNotFound(err) {
					klog.Infof("Not found registryNamespace %v for OperandRequest %v/%v", regNs, instance.Namespace, instance.Name)
					isDefaulting = true
				} else {
					klog.Errorf("Failed to get namespace %v: %v", regNs, err)
				}
			}
		} else if len(watchNamespaces) != 0 && !util.Contains(strings.Split(watchNamespaces, ","), regNs) {
			isDefaulting = true
		}
		if isDefaulting {
			serviceNs := util.GetServicesNamespace(r.Reader)
			instance.Spec.Requests[i].RegistryNamespace = serviceNs
			klog.V(2).Infof("Setting %vth RegistryNamespace for OperandRequest %v/%v to %v", i, instance.Namespace, instance.Name, serviceNs)
		}
	}
}

func (r *Defaulter) InjectDecoder(decoder *admission.Decoder) error {
	r.decoder = decoder
	return nil
}

func (r *Defaulter) SetupWebhookWithManager(mgr ctrl.Manager) error {

	mgr.GetWebhookServer().
		Register("/mutate-operator-ibm-com-v1alpha1-operandrequest",
			&webhook.Admission{Handler: r})

	return nil
}
