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
	"encoding/json"
	"reflect"

	utilyaml "github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
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
	klog.Info("Reconciling CommonService")
	// Fetch the CommonService instance
	instance := &unstructured.Unstructured{}
	instance.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.ibm.com", Kind: "CommonService", Version: "v3"})

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

	var newConfigs []interface{}

	if instance.Object["spec"] == nil {
		return ctrl.Result{}, nil
	}

	switch instance.Object["spec"].(map[string]interface{})["size"] {
	case "small":
		if instance.Object["spec"].(map[string]interface{})["services"] == nil {
			newConfigs = deepMerge(newConfigs, size.Small)
		} else {
			newConfigs = deepMerge(instance.Object["spec"].(map[string]interface{})["services"].([]interface{}), size.Small)
		}
	case "medium":
		if instance.Object["spec"].(map[string]interface{})["services"] == nil {
			newConfigs = deepMerge(newConfigs, size.Medium)
		} else {
			newConfigs = deepMerge(instance.Object["spec"].(map[string]interface{})["services"].([]interface{}), size.Medium)
		}
	case "large":
		if instance.Object["spec"].(map[string]interface{})["services"] == nil {
			newConfigs = deepMerge(newConfigs, size.Large)
		} else {
			newConfigs = deepMerge(instance.Object["spec"].(map[string]interface{})["services"].([]interface{}), size.Large)
		}
	}

	err = r.updateOpcon(ctx, newConfigs)

	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *CommonServiceReconciler) updateOpcon(ctx context.Context, newConfigs []interface{}) error {
	opcon := &unstructured.Unstructured{}
	opcon.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.ibm.com", Kind: "OperandConfig", Version: "v1alpha1"})
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      "common-service",
		Namespace: "ibm-common-services",
	}, opcon)
	if err != nil {
		klog.Error(err)
		return err
	}
	services := opcon.Object["spec"].(map[string]interface{})["services"].([]interface{})
	for _, service := range services {
		for _, size := range newConfigs {
			if service.(map[string]interface{})["name"].(string) == size.(map[string]interface{})["name"].(string) {
				for cr, spec := range service.(map[string]interface{})["spec"].(map[string]interface{}) {
					if size.(map[string]interface{})["spec"].(map[string]interface{})[cr] == nil {
						continue
					}
					service.(map[string]interface{})["spec"].(map[string]interface{})[cr] = mergeConfig(spec.(map[string]interface{}), size.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{}))
				}
			}
		}
	}

	opcon.Object["spec"].(map[string]interface{})["services"] = services

	err = r.Client.Update(ctx, opcon)
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

func deepMerge(src []interface{}, dest string) []interface{} {

	jsonSpec, err := utilyaml.YAMLToJSON([]byte(dest))

	if err != nil {
		klog.Error(err)
	}

	// Create a slice for sizes
	var sizes []interface{}

	// Convert sizes string to slice
	err = json.Unmarshal(jsonSpec, &sizes)

	if err != nil {
		klog.Error(err)
	}

	for _, configSize := range sizes {
		for _, config := range src {
			if config.(map[string]interface{})["name"].(string) == configSize.(map[string]interface{})["name"].(string) {
				if config == nil {
					continue
				}
				if configSize == nil {
					configSize = config
					continue
				}
				for cr, size := range mergeConfig(configSize.(map[string]interface{})["spec"].(map[string]interface{}), config.(map[string]interface{})["spec"].(map[string]interface{})) {
					configSize.(map[string]interface{})["spec"].(map[string]interface{})[cr] = size
				}
			}
		}
	}
	return sizes
}

// mergeConfig deep merge two configs
func mergeConfig(defaultMap map[string]interface{}, changedMap map[string]interface{}) map[string]interface{} {
	for key := range defaultMap {
		checkKeyBeforeMerging(key, defaultMap[key], changedMap[key], changedMap)
	}
	return changedMap
}

func checkKeyBeforeMerging(key string, defaultMap interface{}, changedMap interface{}, finalMap map[string]interface{}) {
	if !reflect.DeepEqual(defaultMap, changedMap) {
		switch defaultMap := defaultMap.(type) {
		case map[string]interface{}:
			//Check that the changed map value doesn't contain this map at all and is nil
			if changedMap == nil {
				finalMap[key] = defaultMap
			} else if _, ok := changedMap.(map[string]interface{}); ok { //Check that the changed map value is also a map[string]interface
				defaultMapRef := defaultMap
				changedMapRef := changedMap.(map[string]interface{})
				for newKey := range defaultMapRef {
					checkKeyBeforeMerging(newKey, defaultMapRef[newKey], changedMapRef[newKey], finalMap[key].(map[string]interface{}))
				}
			}
		default:
			//Check if the value was set, otherwise set it
			if changedMap == nil {
				finalMap[key] = defaultMap
			}
		}
	}
}

func (r *CommonServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv3.CommonService{}).
		Complete(r)
}
