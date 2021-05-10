//
// Copyright 2021 IBM Corporation
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

var ctx = context.Background()

func (r *CommonServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	klog.Infof("Reconciling CommonService: %s", req.NamespacedName)

	// Validate common-service-maps and filter the namespace of CommonService CR
	isScoped := true
	cm, err := util.GetCmOfMapCs(r.Client)
	if err == nil {
		if err := util.ValidateCsMaps(cm); err != nil {
			klog.Errorf("Unsupported common-service-maps: %v", err)
			return reconcile.Result{RequeueAfter: constant.DefaultRequeueDuration}, err
		}
		csScope, err := util.GetCsScope(cm, r.Bootstrap.CSData.MasterNs)
		if err != nil {
			return ctrl.Result{}, err
		}
		if isScoped = r.checkScope(csScope, req.NamespacedName.Namespace); !isScoped {
			klog.Infof("CommonService CR %v is not in the scope, only reconciles its configuration of cluster scope resource", req.NamespacedName.String())
		}
	} else if !errors.IsNotFound(err) {
		klog.Errorf("Failed to get common-service-maps: %v", err)
		return ctrl.Result{}, err
	}

	// Fetch the CommonService instance
	instance := &apiv3.CommonService{}

	if err := r.Bootstrap.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			if err := r.handleDelete(); err != nil {
				return ctrl.Result{}, err
			}
			klog.Info("Deleted reconciling CommonService CR successfully")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if r.checkNamespace(req.NamespacedName.String()) {
		return r.ReconcileMasterCR(instance, isScoped)
	}
	return r.ReconcileGeneralCR(instance, isScoped)
}

func (r *CommonServiceReconciler) ReconcileMasterCR(instance *apiv3.CommonService, isScoped bool) (ctrl.Result, error) {

	if instance.Status.Phase == "" {
		if err := r.updatePhase(instance, CRInitializing); err != nil {
			klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
			return ctrl.Result{}, err
		}
	} else {
		if err := r.updatePhase(instance, CRUpdating); err != nil {
			klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
			return ctrl.Result{}, err
		}
	}

	// Init common service bootstrap resource
	// Including namespace-scope configmap, nss operator, nss CR
	// Webhook Operator and Secretshare
	// Delete ODLM from openshift-operators and deploy it in the masterNamespaces
	// Deploy OperandConfig and OperandRegistry
	if err := r.Bootstrap.InitResources(instance); err != nil {
		klog.Errorf("Failed to initialize resources: %v", err)
		if err := r.updatePhase(instance, CRFailed); err != nil {
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

	newConfigs, err := r.getNewConfigs(cs, isScoped)
	if err != nil {
		if err := r.updatePhase(instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isEqual, err := r.updateOperandConfig(newConfigs)
	if err != nil {
		if err := r.updatePhase(instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	// Create Event if there is no update in OperandConfig after applying current CR
	if isEqual {
		r.Recorder.Event(instance, corev1.EventTypeNormal, "Noeffect", fmt.Sprintf("No update, resource sizings in the OperandConfig %s/%s are larger than the profile from CommonService CR %s/%s", r.Bootstrap.CSData.MasterNs, "common-service", instance.Namespace, instance.Name))
	}

	if err := r.updatePhase(instance, CRSucceeded); err != nil {
		klog.Error(err)
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s/%s", instance.Namespace, instance.Name)
	return ctrl.Result{}, nil
}

// ReconcileGeneralCR is for setting the OperandConfig
func (r *CommonServiceReconciler) ReconcileGeneralCR(instance *apiv3.CommonService, isScoped bool) (ctrl.Result, error) {

	if instance.Status.Phase == "" {
		if err := r.updatePhase(instance, CRInitializing); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
	} else {
		if err := r.updatePhase(instance, CRUpdating); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
	}

	opcon := util.NewUnstructured("operator.ibm.com", "OperandConfig", "v1alpha1")
	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: r.Bootstrap.CSData.MasterNs,
	}
	if err := r.Reader.Get(ctx, opconKey, opcon); err != nil {
		klog.Errorf("failed to get OperandConfig %s: %v", opconKey.String(), err)
		if err := r.updatePhase(instance, CRFailed); err != nil {
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

	newConfigs, err := r.getNewConfigs(cs, isScoped)
	if err != nil {
		if err := r.updatePhase(instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	isEqual, err := r.updateOperandConfig(newConfigs)
	if err != nil {
		if err := r.updatePhase(instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	// Create Event if there is no update in OperandConfig after applying current CR
	if isEqual {
		r.Recorder.Event(instance, corev1.EventTypeNormal, "Noeffect", fmt.Sprintf("No update, resource sizings in the OperandConfig %s/%s are larger than the profile from CommonService CR %s/%s", r.Bootstrap.CSData.MasterNs, "common-service", instance.Namespace, instance.Name))
	}

	if err := r.updatePhase(instance, CRSucceeded); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s/%s", instance.Namespace, instance.Name)
	return ctrl.Result{}, nil
}

func (r *CommonServiceReconciler) toCsRequest() handler.ToRequestsFunc {
	return func(object handler.MapObject) []reconcile.Request {
		CsInstance := []reconcile.Request{}
		cmName := object.Meta.GetName()
		cmNs := object.Meta.GetNamespace()
		if cmName == constant.CsMapConfigMap && cmNs == "kube-public" {
			CsInstance = append(CsInstance, reconcile.Request{NamespacedName: types.NamespacedName{Name: "common-service", Namespace: r.Bootstrap.CSData.MasterNs}})
		}
		return CsInstance
	}
}

func (r *CommonServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv3.CommonService{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: r.toCsRequest()},
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
