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
	"encoding/json"
	"reflect"
	"strings"

	utilyaml "github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/deploy"
	"github.com/IBM/ibm-common-service-operator/controllers/size"
	"github.com/IBM/ibm-common-service-operator/controllers/storageclass"
)

// CommonServiceReconciler reconciles a CommonService object
type CommonServiceReconciler struct {
	client.Client
	client.Reader
	*deploy.Manager
	*bootstrap.Bootstrap
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var ctx = context.Background()

func (r *CommonServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	klog.Infof("Reconciling CommonService: %s", req.NamespacedName)

	// Fetch the CommonService instance
	instance := &apiv3.CommonService{}

	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Init common service bootstrap resource
	if err := r.Bootstrap.InitResources(instance.Spec.ManualManagement); err != nil {
		klog.Errorf("Failed to initialize resources: %v", err)
		return ctrl.Result{}, err
	}

	newConfigs, err := r.getNewConfigs(req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = r.updateOpcon(newConfigs); err != nil {
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *CommonServiceReconciler) getNewConfigs(req ctrl.Request) ([]interface{}, error) {
	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	if err := r.Client.Get(ctx, req.NamespacedName, cs); err != nil {
		return nil, err
	}
	var newConfigs []interface{}

	if cs.Object["spec"].(map[string]interface{})["services"] != nil {
		newConfigs = cs.Object["spec"].(map[string]interface{})["services"].([]interface{})
	}

	if cs.Object["spec"].(map[string]interface{})["storageClass"] != nil {
		newConfigs = deepMerge(newConfigs, strings.ReplaceAll(storageclass.Template, "placeholder", cs.Object["spec"].(map[string]interface{})["storageClass"].(string)))
	}

	switch cs.Object["spec"].(map[string]interface{})["size"] {
	case "small":
		newConfigs = deepMerge(newConfigs, size.Small)
	case "medium":
		newConfigs = deepMerge(newConfigs, size.Medium)
	case "large":
		newConfigs = deepMerge(newConfigs, size.Large)
	}

	return newConfigs, nil
}

func (r *CommonServiceReconciler) updateOpcon(newConfigs []interface{}) error {
	opcon := util.NewUnstructured("operator.ibm.com", "OperandConfig", "v1alpha1")
	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: "ibm-common-services",
	}
	if err := r.Reader.Get(ctx, opconKey, opcon); err != nil {
		klog.Errorf("failed to get OperandConfig %s: %v", opconKey.String(), err)
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

	if err := r.Update(ctx, opcon); err != nil {
		klog.Errorf("failed to update OperandConfig %s: %v", opconKey.String(), err)
		return err
	}

	return nil
}

func deepMerge(src []interface{}, dest string) []interface{} {

	jsonSpec, err := utilyaml.YAMLToJSON([]byte(dest))

	if err != nil {
		klog.Errorf("failed to convert yaml to json: %v", err)
	}

	// Create a slice for sizes
	var sizes []interface{}

	// Convert sizes string to slice
	err = json.Unmarshal(jsonSpec, &sizes)

	if err != nil {
		klog.Errorf("failed to convert string to slice: %v", err)
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

// Check if the request's NamespacedName is equal "ibm-common-services/common-service"
func checkNamespace(key string) bool {
	if key != "ibm-common-services/common-service" {
		klog.Infof("Ignore reconcile when commonservices.operator.ibm.com is NamespacedName '%s', only reconcile for NamespacedName 'ibm-common-services/common-service'", key)
		return false
	}
	return true
}

func filterNamespacePredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Only reconcle when NamespacedName equal "ibm-common-services/common-service"
			key := e.Meta.GetNamespace() + "/" + e.Meta.GetName()
			return checkNamespace(key)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Only reconcle when NamespacedName equal "ibm-common-services/common-service"
			key := e.MetaNew.GetNamespace() + "/" + e.MetaNew.GetName()
			return checkNamespace(key)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			// Only reconcle when NamespacedName equal "ibm-common-services/common-service"
			key := e.Meta.GetNamespace() + "/" + e.Meta.GetName()
			return !e.DeleteStateUnknown && checkNamespace(key)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			// Only reconcle when NamespacedName equal "ibm-common-services/common-service"
			key := e.Meta.GetNamespace() + "/" + e.Meta.GetName()
			return checkNamespace(key)
		},
	}
}

func (r *CommonServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv3.CommonService{}).
		WithEventFilter(filterNamespacePredicate()).
		Complete(r)
}
