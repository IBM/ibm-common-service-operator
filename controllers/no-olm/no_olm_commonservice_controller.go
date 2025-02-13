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

package noolm

import (
	"context"
	"strings"

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

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/v4/controllers/common"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/configurationcollector"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/constant"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
)

// CommonServiceReconciler reconciles a CommonService object
type NoOLMCommonServiceReconciler struct {
	*bootstrap.Bootstrap
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

const (
	CRInitializing string = "Initializing"
	CRUpdating     string = "Updating"
	CRPending      string = "Pending"
	CRSucceeded    string = "Succeeded"
	CRFailed       string = "Failed"
)

func (r *NoOLMCommonServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

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
		// return r.ReconcileNonConfigurableCR(ctx, instance)
	}

	if !instance.Spec.License.Accept {
		klog.Error("Accept license by changing .spec.license.accept to true in the CommonService CR. Operator will not proceed until then")
	}

	klog.Infof("Reconciling CommonService: %s in non OLM environment", req.NamespacedName)

	// create ibm-cpp-config configmap
	if err := configurationcollector.CreateUpdateConfig(r.Bootstrap); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	// deploy Cert Manager CR
	if err := r.Bootstrap.DeployCertManagerCR(); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	// Temporary solution for EDB image ConfigMap reference
	klog.Infof("It is a non-OLM mode, skip creating EDB Image ConfigMap...")

	klog.Infof("Start to Create ODLM CR in the namespace %s", r.Bootstrap.CSData.OperatorNs)
	// Check if ODLM OperandRegistry and OperandConfig are created
	klog.Info("Checking if OperandRegistry and OperandConfig CRD already exist")
	existOpreg, _ := r.Bootstrap.CheckCRD(constant.OpregAPIGroupVersion, constant.OpregKind)
	existOpcon, _ := r.Bootstrap.CheckCRD(constant.OpregAPIGroupVersion, constant.OpconKind)
	// Install/update Opreg and Opcon resources before installing ODLM if CRDs exist
	if existOpreg && existOpcon {
		klog.Info("Installing/Updating OperandRegistry")
		if err := r.Bootstrap.InstallOrUpdateOpreg(true, ""); err != nil {
			klog.Errorf("Fail to Installing/Updating OperandConfig: %v", err)
			return ctrl.Result{}, err
		}

		klog.Info("Installing/Updating OperandConfig")
		if err := r.Bootstrap.InstallOrUpdateOpcon(true); err != nil {
			klog.Errorf("Fail to Installing/Updating OperandConfig: %v", err)
			return ctrl.Result{}, err
		}
	} else {
		klog.Error("ODLM CRD not ready, waiting for it to be ready")
	}

	return ctrl.Result{}, nil

}

func (r *NoOLMCommonServiceReconciler) mappingToCsRequest() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		CsInstance := []reconcile.Request{}
		cmName := object.GetName()
		cmNs := object.GetNamespace()
		if cmName == constant.CsMapConfigMap && cmNs == "kube-public" {
			CsInstance = append(CsInstance, reconcile.Request{NamespacedName: types.NamespacedName{Name: constant.MasterCR, Namespace: r.Bootstrap.CSData.OperatorNs}})
		}
		return CsInstance
	}
}

func (r *NoOLMCommonServiceReconciler) mappingToCsRequestForOperandRegistry() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		operandRegistry, ok := object.(*odlm.OperandRegistry)
		if !ok {
			// It's not an OperandRegistry, ignore
			return nil
		}
		if operandRegistry.Name == constant.MasterCR && operandRegistry.Namespace == r.Bootstrap.CSData.ServicesNs {
			if shouldReconcile(operandRegistry) {
				// Enqueue a reconciliation request for the corresponding CommonService
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{Name: constant.MasterCR, Namespace: r.Bootstrap.CSData.OperatorNs}},
				}
			}
		}
		return nil
	}
}

// shouldReconcile checks the conditions for reconciliation
func shouldReconcile(operandRegistry *odlm.OperandRegistry) bool {

	if operandRegistry.Status.OperatorsStatus != nil {
		// List all requested operators
		for operator := range operandRegistry.Status.OperatorsStatus {
			// If there is a requested operator's installMode is "no-op", then skip reconcile
			for _, op := range operandRegistry.Spec.Operators {
				if op.Name == operator && op.InstallMode == "no-op" {
					klog.Infof("The operator %s with 'no-op' installMode is still requested in OperandRegistry, skip reconciliation", operator)
					return false
				}
			}
		}
	}
	return true
}

func (r *NoOLMCommonServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	controller := ctrl.NewControllerManagedBy(mgr).
		// AnnotationChangedPredicate is intended to be used in conjunction with the GenerationChangedPredicate
		For(&apiv3.CommonService{}, builder.WithPredicates(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
				predicate.LabelChangedPredicate{}))).
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
			}))
	if isOpregAPI, err := r.Bootstrap.CheckCRD(constant.OpregAPIGroupVersion, constant.OpregKind); err != nil {
		klog.Errorf("Failed to check if OperandRegistry CRD exists: %v", err)
		return err
	} else if isOpregAPI {
		controller = controller.Watches(
			&source.Kind{Type: &odlm.OperandRegistry{}},
			handler.EnqueueRequestsFromMapFunc(r.mappingToCsRequestForOperandRegistry()),
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldOperandRegistry, ok := e.ObjectOld.(*odlm.OperandRegistry)
					if !ok {
						return false
					}

					newOperandRegistry, ok := e.ObjectNew.(*odlm.OperandRegistry)
					if !ok {
						return false
					}

					// Return true if the length of .status.operatorsStatus array has changed, indicating that a operator has been added or removed
					return len(oldOperandRegistry.Status.OperatorsStatus) != len(newOperandRegistry.Status.OperatorsStatus)
				},
			},
			))
	}
	return controller.Complete(r)
}
