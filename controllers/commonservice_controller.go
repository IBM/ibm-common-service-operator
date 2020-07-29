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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/size"
)

// CommonServiceReconciler reconciles a CommonService object
type CommonServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=*,resources=*,verbs=*

func (r *CommonServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("commonservice", req.NamespacedName)
	log.Info("Reconciling CommonService")

	// Fetch the CommonService instance
	instance := &apiv3.CommonService{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	var newConfigs []apiv3.ServiceConfig

	if instance.Spec.Size == "small" {
		newConfigs = deepMerge(instance.Spec.Services, size.Small)
	}

	if instance.Spec.Size == "medium" {
		newConfigs = deepMerge(instance.Spec.Services, size.Medium)
	}

	if instance.Spec.Size == "large" {
		newConfigs = deepMerge(instance.Spec.Services, size.Large)
	}

	if isConfigChanged() {
		err := patchOperandConfig(newConfigs)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func deepMerge(src []apiv3.ServiceConfig, dest string) []apiv3.ServiceConfig {
	return nil
}

func patchOperandConfig(newConfigs []apiv3.ServiceConfig) error {
	return nil
}

func isConfigChanged() bool {
	return false
}

func (r *CommonServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv3.CommonService{}).
		Complete(r)
}
