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

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	// certmanagerv1alpha1 "github.com/ibm/ibm-cert-manager-operator/apis/certmanager/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	"github.com/IBM/ibm-common-service-operator/controllers/webhooks"
)

// CommonServiceReconciler reconciles a CommonService object
type CommonServiceReconciler struct {
	*bootstrap.Bootstrap
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

const (
	CRInitializing string = "Initializing"
	CRUpdating     string = "Updating"
	CRSucceeded    string = "Succeeded"
	CRFailed       string = "Failed"
)

// var ctx = context.Background()

func (r *CommonServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	klog.Infof("Reconciling CommonService: %s", req.NamespacedName)

	// Fetch the CommonService instance
	instance := &apiv3.CommonService{}
	if req.Name == constant.MasterCR && util.Contains(strings.Split(r.Bootstrap.CSData.WatchNamespaces, ","), req.Namespace) && req.Namespace != r.Bootstrap.CSData.OperatorNs {
		if err := r.Bootstrap.Client.Get(ctx, req.NamespacedName, instance); err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("Finished reconciling to delete CommonService: %s/%s", req.NamespacedName.Namespace, req.NamespacedName.Name)
			}
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return r.ReconileNonConfigurableCR(ctx, instance)
	}

	if err := r.Bootstrap.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			if err := r.handleDelete(ctx); err != nil {
				return ctrl.Result{}, err
			}
			// If it is BYOCert
			isBYOC, err := r.Bootstrap.IsBYOCert()
			if err != nil {
				return ctrl.Result{}, err
			}
			// Generate Issuer and Certificate CR
			if err := r.Bootstrap.DeployCertManagerCR(isBYOC); err != nil {
				return ctrl.Result{}, err
			}
			klog.Infof("Finished reconciling to delete CommonService: %s/%s", req.NamespacedName.Namespace, req.NamespacedName.Name)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if r.checkNamespace(req.NamespacedName.String()) {
		return r.ReconcileMasterCR(ctx, instance)
	}
	return r.ReconcileGeneralCR(ctx, instance)
}

func (r *CommonServiceReconciler) ReconcileMasterCR(ctx context.Context, instance *apiv3.CommonService) (ctrl.Result, error) {
	originalInstance := instance.DeepCopy()

	operatorDeployed, servicesDeployed := r.Bootstrap.CheckDeployStatus(ctx)
	instance.UpdateConfigStatus(&r.Bootstrap.CSData, operatorDeployed, servicesDeployed)

	r.Bootstrap.CSData.CPFSNs = string(instance.Status.ConfigStatus.OperatorPlane.OperatorNamespace)
	r.Bootstrap.CSData.ServicesNs = string(instance.Status.ConfigStatus.ServicesPlane.ServicesNamespace)
	r.Bootstrap.CSData.CatalogSourceName = string(instance.Status.ConfigStatus.CatalogPlane.CatalogName)
	r.Bootstrap.CSData.CatalogSourceNs = string(instance.Status.ConfigStatus.CatalogPlane.CatalogNamespace)
	var forceUpdateODLMCRs bool
	if !reflect.DeepEqual(originalInstance.Status, instance.Status) {
		forceUpdateODLMCRs = true
	}

	if err := r.Client.Status().Patch(ctx, instance, client.MergeFrom(originalInstance)); err != nil {
		return ctrl.Result{}, fmt.Errorf("error while patching CommonService.Status: %v", err)
	}

	if instance.Status.Phase == "" {
		if err := r.updatePhase(ctx, instance, CRInitializing); err != nil {
			klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
			return ctrl.Result{}, err
		}
	} else {
		if err := r.updatePhase(ctx, instance, CRUpdating); err != nil {
			klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
			return ctrl.Result{}, err
		}
	}
	// Reconcile the webhooks if it is ocp
	if r.Bootstrap.CSData.IsOCP {
		if err := webhooks.Config.Reconcile(context.TODO(), r.Client, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Init common service bootstrap resource
	// Including namespace-scope configmap
	// Deploy OperandConfig and OperandRegistry
	if err := r.Bootstrap.InitResources(instance, forceUpdateODLMCRs); err != nil {
		klog.Errorf("Failed to initialize resources: %v", err)
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	if err := r.Bootstrap.Client.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, cs); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}
	// Generate Issuer and Certificate CR
	if err := r.Bootstrap.DeployCertManagerCR(false); err != nil {
		klog.Errorf("Failed to deploy cert manager CRs: %v", err)
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}
	newConfigs, serviceControllerMapping, err := r.getNewConfigs(cs, true)
	if err != nil {
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isEqual, err := r.updateOperandConfig(ctx, newConfigs, serviceControllerMapping)
	if err != nil {
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	// Create Event if there is no update in OperandConfig after applying current CR
	if isEqual {
		r.Recorder.Event(instance, corev1.EventTypeNormal, "Noeffect", fmt.Sprintf("No update, resource sizings in the OperandConfig %s/%s are larger than the profile from CommonService CR %s/%s", r.Bootstrap.CSData.OperatorNs, "common-service", instance.Namespace, instance.Name))
	}

	if err := r.Bootstrap.PropagateDefaultCR(instance); err != nil {
		klog.Error(err)
		return ctrl.Result{}, err
	}

	if err := r.updatePhase(ctx, instance, CRSucceeded); err != nil {
		klog.Error(err)
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s/%s", instance.Namespace, instance.Name)
	return ctrl.Result{}, nil
}

// ReconcileGeneralCR is for setting the OperandConfig
func (r *CommonServiceReconciler) ReconcileGeneralCR(ctx context.Context, instance *apiv3.CommonService) (ctrl.Result, error) {

	if instance.Status.Phase == "" {
		if err := r.updatePhase(ctx, instance, CRInitializing); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
	} else {
		if err := r.updatePhase(ctx, instance, CRUpdating); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
	}

	opcon := util.NewUnstructured("operator.ibm.com", "OperandConfig", "v1alpha1")
	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: r.Bootstrap.CSData.ServicesNs,
	}
	if err := r.Reader.Get(ctx, opconKey, opcon); err != nil {
		klog.Errorf("failed to get OperandConfig %s: %v", opconKey.String(), err)
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	if err := r.Bootstrap.Client.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, cs); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}
	// Generate Issuer and Certificate CR
	if err := r.Bootstrap.DeployCertManagerCR(false); err != nil {
		klog.Errorf("Failed to deploy cert manager CRs: %v", err)
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	newConfigs, serviceControllerMapping, err := r.getNewConfigs(cs, true)
	if err != nil {
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	isEqual, err := r.updateOperandConfig(ctx, newConfigs, serviceControllerMapping)
	if err != nil {
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	// Create Event if there is no update in OperandConfig after applying current CR
	if isEqual {
		r.Recorder.Event(instance, corev1.EventTypeNormal, "Noeffect", fmt.Sprintf("No update, resource sizings in the OperandConfig %s/%s are larger than the profile from CommonService CR %s/%s", r.Bootstrap.CSData.OperatorNs, "common-service", instance.Namespace, instance.Name))
	}

	if err := r.updatePhase(ctx, instance, CRSucceeded); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s/%s", instance.Namespace, instance.Name)
	return ctrl.Result{}, nil
}

// ReconileNonConfigurableCR is for setting the cloned Master CR status for advaned topologies
func (r *CommonServiceReconciler) ReconileNonConfigurableCR(ctx context.Context, instance *apiv3.CommonService) (ctrl.Result, error) {

	if instance.Status.Phase == "" {
		if err := r.updatePhase(ctx, instance, CRInitializing); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
	} else {
		if err := r.updatePhase(ctx, instance, CRUpdating); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
	}

	originalInstance := instance.DeepCopy()

	instance.Status.ConfigStatus.OperatorPlane.OperatorNamespace = apiv3.OperatorNamespace(r.Bootstrap.CSData.OperatorNs)
	instance.Status.ConfigStatus.ServicesPlane.ServicesNamespace = apiv3.ServicesNamespace(r.Bootstrap.CSData.ServicesNs)
	instance.Status.ConfigStatus.CatalogPlane.CatalogName = apiv3.CatalogName(r.Bootstrap.CSData.CatalogSourceName)
	instance.Status.ConfigStatus.CatalogPlane.CatalogNamespace = apiv3.CatalogNamespace(r.Bootstrap.CSData.CatalogSourceNs)
	instance.Status.Configurable = false

	if !reflect.DeepEqual(originalInstance.Status, instance.Status) {
		r.Recorder.Event(instance, corev1.EventTypeNormal, "Noeffect", fmt.Sprintf("No update, this resource is the clone of Common Service CR named %s from namespace %s", constant.MasterCR, r.Bootstrap.CSData.OperatorNs))
		if err := r.Client.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.updatePhase(ctx, instance, CRSucceeded); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s/%s", instance.Namespace, instance.Name)
	return ctrl.Result{}, nil
}

func (r *CommonServiceReconciler) mappingToCsRequest() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		CsInstance := []reconcile.Request{}
		cmName := object.GetName()
		cmNs := object.GetNamespace()
		if cmName == constant.CsMapConfigMap && cmNs == "kube-public" {
			CsInstance = append(CsInstance, reconcile.Request{NamespacedName: types.NamespacedName{Name: "common-service", Namespace: r.Bootstrap.CSData.OperatorNs}})
		}
		return CsInstance
	}
}

func (r *CommonServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv3.CommonService{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.mappingToCsRequest()),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool {
					return true
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					return !e.DeleteStateUnknown
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					return true
				},
			})).Complete(r)
}
