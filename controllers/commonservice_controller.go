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
	"fmt"
	"reflect"

	utilyaml "github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	"github.com/IBM/ibm-common-service-operator/controllers/deploy"
	"github.com/IBM/ibm-common-service-operator/controllers/rules"
	"github.com/IBM/ibm-common-service-operator/controllers/size"
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

	if checkNamespace(req.NamespacedName.String()) {
		return r.ReconcileMasterCR(req)
	}
	return r.ReconcileGeneralCR(req)
}

func (r *CommonServiceReconciler) ReconcileMasterCR(req ctrl.Request) (ctrl.Result, error) {
	// Fetch the CommonService instance
	instance := &apiv3.CommonService{}

	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Init common service bootstrap resource
	// Including namespace-scope configmap, nss operator, nss CR
	// Webhook Operator and Secretshare
	// Delete ODLM from openshift-operators and deploy it in the masterNamespaces
	// Deploy OperandConfig and OperandRegistry
	if err := r.Bootstrap.InitResources(instance.Spec.ManualManagement); err != nil {
		klog.Errorf("Failed to initialize resources: %v", err)
		return ctrl.Result{}, err
	}

	newConfigs, err := r.getNewConfigs(req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = r.updateOperandConfig(newConfigs); err != nil {
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s", req.NamespacedName)
	return ctrl.Result{}, nil
}

// ReconcileGeneralCR is for setting the OperandConfig
func (r *CommonServiceReconciler) ReconcileGeneralCR(req ctrl.Request) (ctrl.Result, error) {

	opcon := util.NewUnstructured("operator.ibm.com", "OperandConfig", "v1alpha1")
	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: constant.MasterNamespace,
	}
	if err := r.Reader.Get(ctx, opconKey, opcon); err != nil {
		klog.Errorf("failed to get OperandConfig %s: %v", opconKey.String(), err)
		return ctrl.Result{}, err
	}

	newConfigs, err := r.getNewConfigs(req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = r.updateOperandConfig(newConfigs); err != nil {
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
	switch cs.Object["spec"].(map[string]interface{})["size"] {
	case "small":
		newConfigs = applySizeTemplate(cs, size.Small)
	case "medium":
		newConfigs = applySizeTemplate(cs, size.Medium)
	case "large":
		newConfigs = applySizeTemplate(cs, size.Large)
	default:
		if cs.Object["spec"].(map[string]interface{})["services"] != nil {
			newConfigs = cs.Object["spec"].(map[string]interface{})["services"].([]interface{})
		}
	}
	return newConfigs, nil
}

func applySizeTemplate(cs *unstructured.Unstructured, sizeTemplate string) []interface{} {

	var src []interface{}
	if cs.Object["spec"].(map[string]interface{})["services"] != nil {
		src = cs.Object["spec"].(map[string]interface{})["services"].([]interface{})
	}

	// Convert sizes string to slice
	sizes, err := convertStringToSlice(sizeTemplate)
	if err != nil {
		klog.Errorf("convert size to interface slice: %v", err)
	}

	for _, configSize := range sizes {
		config := getSpecByName(src, configSize.(map[string]interface{})["name"].(string))
		if config == nil {
			continue
		}
		if configSize == nil {
			configSize = config
			continue
		}
		for cr, size := range mergeSizeProfile(configSize.(map[string]interface{})["spec"].(map[string]interface{}), config.(map[string]interface{})["spec"].(map[string]interface{})) {
			configSize.(map[string]interface{})["spec"].(map[string]interface{})[cr] = size
		}
	}

	return sizes
}

// mergeCRsFromOperandConfig merges CRs by specific rules
func mergeCRsFromOperandConfig(defaultMap map[string]interface{}, changedMap map[string]interface{}, rules map[string]interface{}) map[string]interface{} {
	for key := range changedMap {
		filterChangedMapWithRules(key, changedMap[key], rules[key], changedMap)
	}
	for key := range defaultMap {
		if reflect.DeepEqual(defaultMap[key], changedMap[key]) {
			continue
		}
		mergeChangedMap(key, defaultMap[key], changedMap[key], changedMap)
	}
	return changedMap
}

// mergeCRsFromOperandConfig merges CRs by specific rules
func mergeCRsFromOperandConfigWithDefaultRules(defaultMap map[string]interface{}, changedMap map[string]interface{}) map[string]interface{} {
	for key := range defaultMap {
		if reflect.DeepEqual(defaultMap[key], changedMap[key]) {
			continue
		}
		mergeChangedMap(key, defaultMap[key], changedMap[key], changedMap)
	}
	return changedMap
}

func filterChangedMapWithRules(key string, changedMap interface{}, rules interface{}, finalMap map[string]interface{}) {
	switch changedMap.(type) {
	case map[string]interface{}:
		//Check that the changed map value doesn't contain this map at all and is nil
		if rules == nil {
			delete(finalMap, key)
		} else {
			if _, ok := rules.(map[string]interface{}); ok {
				rulesRef := rules.(map[string]interface{})
				changedMapRef := changedMap.(map[string]interface{})
				for newKey := range changedMapRef {
					filterChangedMapWithRules(newKey, changedMapRef[newKey], rulesRef[newKey], finalMap[key].(map[string]interface{}))
				}
			} else {
				delete(finalMap, key)
			}
		}
	default:
		if rules == nil && changedMap != nil {
			delete(finalMap, key)
		}
	}
}

func mergeChangedMap(key string, defaultMap interface{}, changedMap interface{}, finalMap map[string]interface{}) {
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
					mergeChangedMap(newKey, defaultMapRef[newKey], changedMapRef[newKey], finalMap[key].(map[string]interface{}))
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

// mergeSizeProfile deep merge two configs
func mergeSizeProfile(defaultMap map[string]interface{}, changedMap map[string]interface{}) map[string]interface{} {
	for key := range defaultMap {
		if reflect.DeepEqual(defaultMap[key], changedMap[key]) {
			continue
		}
		deepMergeTwoMaps(key, defaultMap[key], changedMap[key], changedMap)
	}
	return changedMap
}

func deepMergeTwoMaps(key string, defaultMap interface{}, changedMap interface{}, finalMap map[string]interface{}) {
	switch defaultMap := defaultMap.(type) {
	case map[string]interface{}:
		//Check that the changed map value doesn't contain this map at all and is nil
		if changedMap == nil {
			finalMap[key] = defaultMap
		} else if _, ok := changedMap.(map[string]interface{}); ok { //Check that the changed map value is also a map[string]interface
			defaultMapRef := defaultMap
			changedMapRef := changedMap.(map[string]interface{})
			for newKey := range defaultMapRef {
				deepMergeTwoMaps(newKey, defaultMapRef[newKey], changedMapRef[newKey], finalMap[key].(map[string]interface{}))
			}
		}
	default:
		//Check if the value was set, otherwise set it
		if changedMap == nil {
			finalMap[key] = defaultMap
		}
	}
}

func (r *CommonServiceReconciler) updateOperandConfig(newConfigs []interface{}) error {
	opcon := util.NewUnstructured("operator.ibm.com", "OperandConfig", "v1alpha1")
	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: constant.MasterNamespace,
	}
	if err := r.Reader.Get(ctx, opconKey, opcon); err != nil {
		klog.Errorf("failed to get OperandConfig %s: %v", opconKey.String(), err)
		return err
	}

	opconServices := opcon.Object["spec"].(map[string]interface{})["services"].([]interface{})

	// Convert rules string to slice
	ruleSlice, err := convertStringToSlice(rules.ConfigurationRules)
	if err != nil {
		return err
	}

	for _, opService := range opconServices {
		size := getSpecByName(newConfigs, opService.(map[string]interface{})["name"].(string))
		rules := getSpecByName(ruleSlice, opService.(map[string]interface{})["name"].(string))
		if size == nil {
			continue
		}
		for cr, spec := range opService.(map[string]interface{})["spec"].(map[string]interface{}) {
			if size.(map[string]interface{})["spec"].(map[string]interface{})[cr] == nil {
				continue
			}
			sizeForCR := size.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{})
			if rules != nil && rules.(map[string]interface{})["spec"] != nil && rules.(map[string]interface{})["spec"].(map[string]interface{})[cr] != nil {
				ruleForCR := rules.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{})
				opService.(map[string]interface{})["spec"].(map[string]interface{})[cr] = mergeCRsFromOperandConfig(spec.(map[string]interface{}), sizeForCR, ruleForCR)
			} else {
				opService.(map[string]interface{})["spec"].(map[string]interface{})[cr] = mergeCRsFromOperandConfigWithDefaultRules(spec.(map[string]interface{}), sizeForCR)
			}
		}

	}

	opcon.Object["spec"].(map[string]interface{})["services"] = opconServices

	if err := r.Update(ctx, opcon); err != nil {
		klog.Errorf("failed to update OperandConfig %s: %v", opconKey.String(), err)
		return err
	}

	return nil
}

func convertStringToSlice(str string) ([]interface{}, error) {

	jsonSpec, err := utilyaml.YAMLToJSON([]byte(str))
	if err != nil {
		return nil, fmt.Errorf("failed to convert yaml to json: %v", err)
	}

	// Create a slice
	var slice []interface{}
	// Convert sizes string to slice
	err = json.Unmarshal(jsonSpec, &slice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert string to slice: %v", err)
	}

	return slice, nil
}

func getSpecByName(slice []interface{}, name string) interface{} {
	for _, item := range slice {
		if item.(map[string]interface{})["name"].(string) == name {
			return item
		}
	}
	return nil
}

// Check if the request's NamespacedName is the "master" CR
func checkNamespace(key string) bool {
	klog.Info(key)
	return key == constant.MasterNamespace+"/common-service"
}

func (r *CommonServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv3.CommonService{}).
		Complete(r)
}
