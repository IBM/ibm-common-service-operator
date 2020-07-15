//
// Copyright 2020 IBM Corporation
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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	operatorv3 "github.com/IBM/ibm-common-service-operator/pkg/apis/operator/v3"
	bootstrap "github.com/IBM/ibm-common-service-operator/pkg/bootstrap"
	odlmv1alpha1 "github.com/IBM/operand-deployment-lifecycle-manager/pkg/apis/operator/v1alpha1"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CommonService Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCommonService{client: mgr.GetClient(), reader: mgr.GetAPIReader(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("commonservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CommonService
	err = c.Watch(&source.Kind{Type: &operatorv3.CommonService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CommonService
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv3.CommonService{},
	})
	if err != nil {
		return err
	}

	// Predicate funcs for watch OperandRegistry and OperandConfig
	predFuncs := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Return false if the Object is not created by cs operator
			if _, ok := e.Meta.GetAnnotations()["version"]; !ok {
				return false
			}
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}
	// ToRequestFunc for watch OperandRegistry and OperandConfig
	toReq := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			rrs := []reconcile.Request{}
			// This is a fake NamespacedName, only use to trigger a reconcile
			key := types.NamespacedName{Name: "bootstrap-init-name", Namespace: "bootstrap-init-namespace"}
			return append(rrs, reconcile.Request{NamespacedName: key})
		})
	// Watch for changes to resource OperandRegistry
	if err = c.Watch(&source.Kind{Type: &odlmv1alpha1.OperandRegistry{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: toReq}, predFuncs); err != nil {
		return err
	}

	// Watch for changes to resource OperandConfig
	if err = c.Watch(&source.Kind{Type: &odlmv1alpha1.OperandConfig{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: toReq}, predFuncs); err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCommonService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCommonService{}

// ReconcileCommonService reconciles a CommonService object
type ReconcileCommonService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	reader client.Reader
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CommonService object and makes changes based on the state read
// and what is in the CommonService.Spec
func (r *ReconcileCommonService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if request.String() == "bootstrap-init-namespace/bootstrap-init-name" {
		// Check if OperandRegistry or OperandConfig existing
		if err := r.createOdlmCr(); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	klog.Infof("Reconciling CommonService: %s", request.NamespacedName)

	// Fetch the CommonService instance
	instance := &operatorv3.CommonService{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	return reconcile.Result{}, nil
}

// createOdlmCr create ODLM resource OperandRegistry and OperandConfig
func (r *ReconcileCommonService) createOdlmCr() error {
	registryKey := types.NamespacedName{Name: "common-service", Namespace: "ibm-common-services"}
	registryObj := &odlmv1alpha1.OperandRegistry{}
	registryIsFound, err := r.foundRuntimeObject(registryKey, registryObj)
	if err != nil {
		return err
	}

	configKey := registryKey
	configObj := &odlmv1alpha1.OperandConfig{}
	configIsFound, err := r.foundRuntimeObject(configKey, configObj)
	if err != nil {
		return err
	}
	if !registryIsFound || !configIsFound {
		klog.Info("Create or Update OperandRegistry or OperandConfig")
		annotations, err := bootstrap.GetAnnotations(r.reader)
		if err != nil {
			return err
		}
		err = bootstrap.CreateOrUpdateResources(annotations, bootstrap.OdlmCrResources, r.client, r.reader)
		if err != nil {
			return err
		}
	}
	return nil
}

// foundRuntimeObject found the runtime.Object
func (r *ReconcileCommonService) foundRuntimeObject(key types.NamespacedName, obj runtime.Object) (bool, error) {
	if err := r.client.Get(context.TODO(), key, obj); err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}
