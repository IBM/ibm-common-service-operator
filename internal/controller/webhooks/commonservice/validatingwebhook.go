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

package commonservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	operatorv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	controller "github.com/IBM/ibm-common-service-operator/v4/internal/controller"
	util "github.com/IBM/ibm-common-service-operator/v4/internal/controller/common"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
)

// +kubebuilder:webhook:path=/validate-operator-ibm-com-v3-commonservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=operator.ibm.com,resources=commonservices,verbs=create;update,versions=v3,name=vcommonservice.kb.io,admissionReviewVersions=v1

// CommonServiceDefaulter points to correct ServicesNamespace
type Defaulter struct {
	Reader    client.Reader
	Client    client.Client
	IsDormant bool
	// decoder is stored as an interface value (not a pointer) as per Go best practices.
	// admission.Decoder is an interface type, and interfaces should not be stored as pointers.
	decoder admission.Decoder
}

// podAnnotator adds an annotation to every incoming pods.
func (r *Defaulter) Handle(ctx context.Context, req admission.Request) admission.Response {
	klog.Infof("Webhook is invoked by Commonservice %s/%s", req.AdmissionRequest.Namespace, req.AdmissionRequest.Name)

	// If operator is not in the operatorNamespace, it is dormant
	if r.IsDormant {
		return admission.Allowed("")
	}

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
	cs := &operatorv3.CommonService{}
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

		// deny if only spec.catalogName or spec.catalogNamespace is set, should be set together
		if (cs.Spec.CatalogName != "" && cs.Spec.CatalogNamespace == "") || (cs.Spec.CatalogName == "" && cs.Spec.CatalogNamespace != "") {
			return admission.Denied("Both User-Definded CatalogSource Name and Namespace must be set together in CommonService CR")
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
	}

	// check HugePageSetting
	deniedHugePage, err := r.HugePageSettingDenied(csUnstrcuted)
	if err != nil || deniedHugePage {
		return admission.Denied(fmt.Sprintf("HugePageSetting is invalid: %v", err))
	}

	// Validate replica configuration against existing OperandConfig.
	// Skip for non-configurable CRs: these are copies of the master CR that the reconciler
	// pushes to other watch namespaces (same name "common-service", different namespace).
	// They carry the same CSPostgreSQLReplica field as the master, so the OperandConfig will
	// already contain the replica config written by the master — rejecting the copy here
	// would be a false positive.
	isNonConfigurableCR := req.AdmissionRequest.Name == constant.MasterCR && req.AdmissionRequest.Namespace != operatorNs

	// Only run the replica-uniqueness check when CSPostgreSQLReplica is newly introduced
	// by this request. On UPDATE requests where the field was already present in the old
	// object, the OperandConfig already contains the replica config written from that CR —
	// re-checking it would always produce a false-positive rejection.
	isNewReplica := false
	if cs.Spec.CSPostgreSQLReplica != nil {
		if req.Operation == admissionv1.Update {
			oldCs := &operatorv3.CommonService{}
			if err := r.decoder.DecodeRaw(req.OldObject, oldCs); err == nil {
				isNewReplica = oldCs.Spec.CSPostgreSQLReplica == nil
			}
		} else {
			// CREATE
			isNewReplica = true
		}
	}

	if !isNonConfigurableCR && isNewReplica {
		if err := r.validateReplicaConfig(ctx, cs, serviceNs); err != nil {
			return admission.Denied(fmt.Sprintf("Replica configuration validation failed: %v", err))
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
			hugePagesStruct, err := controller.UnmarshalHugePages(hugespages)
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

// validateReplicaConfig checks if adding a replica configuration would violate the single-replica constraint
// by examining the actual OperandConfig resource in the cluster
func (r *Defaulter) validateReplicaConfig(ctx context.Context, cs *operatorv3.CommonService, serviceNs string) error {
	// Check if this CR has a CSPostgreSQLReplica configuration
	if cs.Spec.CSPostgreSQLReplica == nil {
		return nil
	}

	// Check the actual OperandConfig in the cluster for existing replica configurations
	opcon := &unstructured.Unstructured{}
	opcon.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operator.ibm.com",
		Version: "v1alpha1",
		Kind:    "OperandConfig",
	})

	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: serviceNs,
	}

	if err := r.Reader.Get(ctx, opconKey, opcon); err != nil {
		// If OperandConfig doesn't exist yet, this is the first one - allow it
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return fmt.Errorf("failed to get OperandConfig: %v", err)
	}

	// Check if OperandConfig already has a replica configuration
	spec, ok := opcon.Object["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	services, ok := spec["services"].([]interface{})
	if !ok {
		return nil
	}

	for _, svc := range services {
		svcMap, ok := svc.(map[string]interface{})
		if !ok {
			continue
		}

		serviceName, _ := svcMap["name"].(string)
		if serviceName != "common-service-cnpg" {
			continue
		}

		resources, ok := svcMap["resources"].([]interface{})
		if !ok {
			continue
		}

		for _, res := range resources {
			resMap, ok := res.(map[string]interface{})
			if !ok {
				continue
			}

			kind, _ := resMap["kind"].(string)
			name, _ := resMap["name"].(string)
			if kind != "Cluster" || name != "common-service-db" {
				continue
			}

			data, ok := resMap["data"].(map[string]interface{})
			if !ok {
				continue
			}

			resSpec, ok := data["spec"].(map[string]interface{})
			if !ok {
				continue
			}

			// Check for existing replica configuration
			if _, hasReplica := resSpec["replica"]; hasReplica {
				return fmt.Errorf("a CSPostgreSQLReplica configuration already exists in the OperandConfig; only one replica configuration is allowed per tenant")
			}
			if _, hasExternal := resSpec["externalClusters"]; hasExternal {
				return fmt.Errorf("a CSPostgreSQLReplica configuration already exists in the OperandConfig; only one replica configuration is allowed per tenant")
			}
			if bootstrap, ok := resSpec["bootstrap"].(map[string]interface{}); ok {
				if _, hasPgBasebackup := bootstrap["pg_basebackup"]; hasPgBasebackup {
					return fmt.Errorf("a CSPostgreSQLReplica configuration already exists in the OperandConfig; only one replica configuration is allowed per tenant")
				}
			}
		}
	}

	return nil
}

func (r *Defaulter) InjectDecoder(decoder admission.Decoder) error {
	r.decoder = decoder
	return nil
}

func (r *Defaulter) SetupWebhookWithManager(mgr ctrl.Manager) error {

	mgr.GetWebhookServer().
		Register("/validate-operator-ibm-com-v3-commonservice",
			&webhook.Admission{Handler: r})

	// Inject the decoder
	decoder := admission.NewDecoder(mgr.GetScheme())
	if err := r.InjectDecoder(decoder); err != nil {
		return err
	}

	return nil
}
