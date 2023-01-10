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
	"fmt"
	"reflect"
	"strings"

	utilyaml "github.com/ghodss/yaml"
	"github.com/mohae/deepcopy"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/rules"
)

var (
	nonDefaultProfileController = map[string]int{
		"turbo":      0,
		"turbonomic": 0,
		"vpa":        1,
	}
)

type Extreme string

const (
	Max Extreme = "max"
	Min Extreme = "min"
)

// mergeCRsIntoOperandConfig merges CRs by specific rules
func mergeCRsIntoOperandConfig(defaultMap map[string]interface{}, changedMap map[string]interface{}, rules map[string]interface{}, overwrite, directAssign bool) map[string]interface{} {
	if !overwrite {
		for key := range changedMap {
			// Remove the items not from the rules
			filterChangedMapWithRules(key, changedMap[key], rules[key], changedMap)
		}
	}
	for key := range defaultMap {
		if reflect.DeepEqual(defaultMap[key], changedMap[key]) {
			continue
		}
		// CR overwrites the existing OperandConfig
		mergeChangedMap(key, defaultMap[key], changedMap[key], changedMap, directAssign)
	}
	return changedMap
}

// shrinkSize merges CRs by picking the smaller size
func shrinkSize(defaultMap map[string]interface{}, changedMap map[string]interface{}, extreme Extreme) map[string]interface{} {
	//TODO: Only shrink the parameter with `Largest_value` rule
	for key := range defaultMap {
		if reflect.DeepEqual(defaultMap[key], changedMap[key]) {
			continue
		}
		mergeChangedMapWithExtremeSize(key, defaultMap[key], changedMap[key], defaultMap, extreme)
	}
	return defaultMap
}

func mergeProfileController(serviceControllerMappingSummary, serviceControllerMapping map[string]string) map[string]string {
	for operator, profileController := range serviceControllerMapping {
		if summaryProfileController, ok := serviceControllerMappingSummary[operator]; ok {
			// Independent profile controller has higher priority then default CS controller
			if _, ok := nonDefaultProfileController[profileController]; ok {
				if _, ok := nonDefaultProfileController[summaryProfileController]; !ok {
					serviceControllerMappingSummary[operator] = profileController
				}
			}
		} else {
			serviceControllerMappingSummary[operator] = profileController
		}
	}
	return serviceControllerMappingSummary
}

func mergeCSCRs(csSummary, csCR, ruleSlice []interface{}, serviceControllerMappingSummary map[string]string) []interface{} {
	for _, operator := range csCR {
		summaryCR := getItemByName(csSummary, operator.(map[string]interface{})["name"].(string))
		rules := getItemByName(ruleSlice, operator.(map[string]interface{})["name"].(string))
		if summaryCR == nil || summaryCR.(map[string]interface{})["spec"] == nil {
			summaryCR = map[string]interface{}{
				"name": operator.(map[string]interface{})["name"].(string),
				"spec": map[string]interface{}{},
			}
		}
		serviceController := serviceControllerMappingSummary["profileController"]
		if controller, ok := serviceControllerMappingSummary[operator.(map[string]interface{})["name"].(string)]; ok {
			serviceController = controller
		}
		for cr, spec := range operator.(map[string]interface{})["spec"].(map[string]interface{}) {
			if _, ok := nonDefaultProfileController[serviceController]; ok {
				// clean up merged CS CR
				operator.(map[string]interface{})["spec"].(map[string]interface{})[cr] = resetResourceInTemplate(spec.(map[string]interface{}), cr, rules)
			}
			if summaryCR.(map[string]interface{})["spec"].(map[string]interface{})[cr] == nil {
				summaryCR.(map[string]interface{})["spec"].(map[string]interface{})[cr] = map[string]interface{}{}
			}
			if rules != nil && rules.(map[string]interface{})["spec"] != nil && rules.(map[string]interface{})["spec"].(map[string]interface{})[cr] != nil {
				ruleForCR := rules.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{})
				sizeForCR := summaryCR.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{})
				summaryCR.(map[string]interface{})["spec"].(map[string]interface{})[cr] = mergeCRsIntoOperandConfig(sizeForCR, spec.(map[string]interface{}), ruleForCR, false, false)
			}
		}
		csSummary = setSpecByName(csSummary, operator.(map[string]interface{})["name"].(string), summaryCR.(map[string]interface{})["spec"])
	}
	return csSummary
}

// mergeCRsIntoOperandConfig merges CRs by specific rules
func mergeCRsIntoOperandConfigWithDefaultRules(defaultMap map[string]interface{}, changedMap map[string]interface{}, directAssign bool) map[string]interface{} {
	// TODO: Apply default rules
	for key := range defaultMap {
		if reflect.DeepEqual(defaultMap[key], changedMap[key]) {
			continue
		}
		mergeChangedMap(key, defaultMap[key], changedMap[key], changedMap, directAssign)
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
			klog.V(3).Info("delete " + key)
			delete(finalMap, key)
		}
	}
}

func mergeChangedMap(key string, defaultMap interface{}, changedMap interface{}, finalMap map[string]interface{}, directAssign bool) {
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
					mergeChangedMap(newKey, defaultMapRef[newKey], changedMapRef[newKey], finalMap[key].(map[string]interface{}), directAssign)
				}
			}
		default:
			//Check if the value was set, otherwise set it
			if changedMap == nil {
				finalMap[key] = defaultMap
			} else {
				var comparableKeys = map[string]bool{
					"replicas":     true,
					"cpu":          true,
					"memory":       true,
					"profile":      true,
					"fipsEnabled":  true,
					"fips_enabled": true,
				}
				if _, ok := comparableKeys[key]; ok {
					if directAssign {
						// Merge current CS CR into OperandConfig
						finalMap[key] = changedMap
					} else {
						finalMap[key], _ = rules.ResourceComparison(defaultMap, changedMap)
					}

				}
			}
		}
	}
}

func mergeChangedMapWithExtremeSize(key string, defaultMap interface{}, changedMap interface{}, finalMap map[string]interface{}, extreme Extreme) {
	if !reflect.DeepEqual(defaultMap, changedMap) {
		switch changedMap.(type) {
		case map[string]interface{}:
			if _, ok := defaultMap.(map[string]interface{}); ok {
				defaultMapRef := defaultMap.(map[string]interface{})
				changedMapRef := changedMap.(map[string]interface{})
				for newKey := range changedMapRef {
					mergeChangedMapWithExtremeSize(newKey, defaultMapRef[newKey], changedMapRef[newKey], finalMap[key].(map[string]interface{}), extreme)
				}
			}
		default:
			//Check if the value was set, otherwise set it
			if changedMap != nil && defaultMap != nil {
				if extreme == Max {
					finalMap[key], _ = rules.ResourceComparison(defaultMap, changedMap)
				} else if extreme == Min {
					_, finalMap[key] = rules.ResourceComparison(defaultMap, changedMap)
				}
			} else if changedMap != nil && defaultMap == nil {
				finalMap[key] = changedMap
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

func (r *CommonServiceReconciler) updateOperandConfig(ctx context.Context, newConfigs []interface{}, serviceControllerMapping map[string]string) (bool, error) {
	opcon := util.NewUnstructured("operator.ibm.com", "OperandConfig", "v1alpha1")
	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: r.Bootstrap.CSData.MasterNs,
	}
	if err := r.Reader.Get(ctx, opconKey, opcon); err != nil {
		klog.Errorf("failed to get OperandConfig %s: %v", opconKey.String(), err)
		return true, err
	}

	// Keep a version of existing config for comparasion later
	opconServices := opcon.Object["spec"].(map[string]interface{})["services"].([]interface{})
	existingOpconServices := deepcopy.Copy(opconServices)

	// Convert rules string to slice
	ruleSlice, err := convertStringToSlice(rules.ConfigurationRules)
	if err != nil {
		return true, err
	}

	for _, newConfigForOperator := range newConfigs {
		if newConfigForOperator == nil {
			continue
		}
		opService := getItemByName(opconServices, newConfigForOperator.(map[string]interface{})["name"].(string))
		if opService == nil {
			continue
		}
		serviceController := serviceControllerMapping["profileController"]
		if controller, ok := serviceControllerMapping[newConfigForOperator.(map[string]interface{})["name"].(string)]; ok {
			serviceController = controller
		}
		// Fetch newConfigForOperator and rules for an operator
		rules := getItemByName(ruleSlice, opService.(map[string]interface{})["name"].(string))

		for cr, spec := range opService.(map[string]interface{})["spec"].(map[string]interface{}) {
			if _, ok := nonDefaultProfileController[serviceController]; ok {
				// clean up OperandConfig
				opService.(map[string]interface{})["spec"].(map[string]interface{})[cr] = resetResourceInTemplate(spec.(map[string]interface{}), cr, rules)
			}

			if newConfigForOperator.(map[string]interface{})["spec"].(map[string]interface{})[cr] == nil {
				continue
			}
			newConfigForCR := newConfigForOperator.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{})

			var overwrite bool
			if opcon.Object["status"] != nil && opcon.Object["status"].(map[string]interface{})["serviceStatus"] != nil {
				overwrite = checkCRFromOperandConfig(opcon.Object["status"].(map[string]interface{})["serviceStatus"].(map[string]interface{}), opService.(map[string]interface{})["name"].(string), cr)
			} else {
				overwrite = true
			}
			if rules != nil && rules.(map[string]interface{})["spec"] != nil && rules.(map[string]interface{})["spec"].(map[string]interface{})[cr] != nil {
				ruleForCR := rules.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{})
				opService.(map[string]interface{})["spec"].(map[string]interface{})[cr] = mergeCRsIntoOperandConfig(spec.(map[string]interface{}), newConfigForCR, ruleForCR, overwrite, true)
			} else {
				if overwrite {
					opService.(map[string]interface{})["spec"].(map[string]interface{})[cr] = mergeCRsIntoOperandConfigWithDefaultRules(spec.(map[string]interface{}), newConfigForCR, false)
				}
			}
		}
	}

	// Checking all the common service CRs to get the minimal(unique largest) size
	opconServices, err = r.getExtremeizes(ctx, opconServices, ruleSlice, Max)
	if err != nil {
		return true, err
	}

	// Compare to see whether new resource sizing is introduced into opconServices
	isEqual := true
	for _, opService := range opconServices {
		existingOpService := getItemByName(existingOpconServices.([]interface{}), opService.(map[string]interface{})["name"].(string))
		if opService.(map[string]interface{})["spec"] == nil {
			continue
		}
		for cr, spec := range opService.(map[string]interface{})["spec"].(map[string]interface{}) {
			existingCrSpec := existingOpService.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{})
			if isEqual = rules.ResourceEqualComparison(existingCrSpec, spec); !isEqual {
				break
			}
		}
		if !isEqual {
			break
		}
	}

	opcon.Object["spec"].(map[string]interface{})["services"] = opconServices

	if err := r.Update(ctx, opcon); err != nil {
		klog.Errorf("failed to update OperandConfig %s: %v", opconKey.String(), err)
		return true, err
	}

	return isEqual, nil
}

func checkCRFromOperandConfig(serviceStatus map[string]interface{}, operatorName, crName string) bool {
	opStatus, ok := serviceStatus[operatorName]
	if !ok {
		return true
	}

	if opStatus.(map[string]interface{})["customResourceStatus"] == nil {
		return true
	}

	for cr := range opStatus.(map[string]interface{})["customResourceStatus"].(map[string]interface{}) {
		if strings.EqualFold(cr, crName) {
			return false
		}
	}
	return true
}

func (r *CommonServiceReconciler) getExtremeizes(ctx context.Context, opconServices, ruleSlice []interface{}, extreme Extreme) ([]interface{}, error) {
	// Fetch all the CommonService instances
	csList := util.NewUnstructuredList("operator.ibm.com", "CommonService", "v3")
	if err := r.Client.List(ctx, csList); err != nil {
		return []interface{}{}, err
	}
	var configSummary []interface{}
	tmpConfigsSlice := make(map[int][]interface{})
	serviceControllerMappingSummary := make(map[string]string)
	for _, cs := range csList.Items {
		if cs.GetDeletionTimestamp() != nil {
			continue
		}

		inScope := true
		cm, err := util.GetCmOfMapCs(r.Client)
		if err == nil {
			csScope, err := util.GetCsScope(cm, r.Bootstrap.CSData.MasterNs)
			if err != nil {
				return configSummary, err
			}
			inScope = r.checkScope(csScope, cs.GetNamespace())
		} else if !errors.IsNotFound(err) {
			klog.Errorf("Failed to get common-service-maps: %v", err)
			return configSummary, err
		}

		csConfigs, serviceControllerMapping, err := r.getNewConfigs(&cs, inScope)
		if err != nil {
			return []interface{}{}, err
		}

		serviceControllerMappingSummary = mergeProfileController(serviceControllerMappingSummary, serviceControllerMapping)
		tmpConfigsSlice[len(tmpConfigsSlice)] = csConfigs
	}
	for _, csConfigs := range tmpConfigsSlice {
		configSummary = mergeCSCRs(configSummary, csConfigs, ruleSlice, serviceControllerMappingSummary)
	}

	for _, opService := range opconServices {
		crSummary := getItemByName(configSummary, opService.(map[string]interface{})["name"].(string))
		if opService.(map[string]interface{})["spec"] == nil {
			continue
		}
		rules := getItemByName(ruleSlice, opService.(map[string]interface{})["name"].(string))
		serviceController := serviceControllerMappingSummary["profileController"]
		if controller, ok := serviceControllerMappingSummary[opService.(map[string]interface{})["name"].(string)]; ok {
			serviceController = controller
		}
		for cr, spec := range opService.(map[string]interface{})["spec"].(map[string]interface{}) {
			if _, ok := nonDefaultProfileController[serviceController]; ok {
				// clean up OperandConfig
				opService.(map[string]interface{})["spec"].(map[string]interface{})[cr] = resetResourceInTemplate(spec.(map[string]interface{}), cr, rules)
			}
			if crSummary == nil || crSummary.(map[string]interface{})["spec"] == nil || crSummary.(map[string]interface{})["spec"].(map[string]interface{})[cr] == nil {
				continue
			}
			serviceForCR := crSummary.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{})
			opService.(map[string]interface{})["spec"].(map[string]interface{})[cr] = shrinkSize(spec.(map[string]interface{}), serviceForCR, extreme)
		}
	}

	return opconServices, nil
}

func (r *CommonServiceReconciler) handleDelete(ctx context.Context) error {
	opcon := util.NewUnstructured("operator.ibm.com", "OperandConfig", "v1alpha1")
	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: r.Bootstrap.CSData.MasterNs,
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
	opconServices, err = r.getExtremeizes(ctx, opconServices, ruleSlice, Min)
	if err != nil {
		return err
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

func getItemByName(slice []interface{}, name string) interface{} {
	for _, item := range slice {
		if item.(map[string]interface{})["name"].(string) == name {
			return item
		}
	}
	return nil
}

func setSpecByName(slice []interface{}, name string, spec interface{}) []interface{} {
	for _, item := range slice {
		if item.(map[string]interface{})["name"].(string) == name {
			item.(map[string]interface{})["spec"] = spec
			return slice
		}
	}
	newItem := map[string]interface{}{
		"name": name,
		"spec": spec,
	}
	return append(slice, newItem)
}

// Check if the request's NamespacedName is the "master" CR
func (r *CommonServiceReconciler) checkNamespace(key string) bool {
	return key == r.Bootstrap.CSData.MasterNs+"/common-service"
}

// updatePhase sets the current Phase status.
func (r *CommonServiceReconciler) updatePhase(ctx context.Context, instance *apiv3.CommonService, status string) error {
	instance.Status.Phase = status
	return r.Client.Status().Update(ctx, instance)
}

// checkScope checks whether the namespace is in scope
func (r *CommonServiceReconciler) checkScope(csScope []string, key string) bool {
	inScope := false
	if !r.Bootstrap.MultiInstancesEnable || len(csScope) == 0 {
		inScope = true
	} else {
		inScope = util.Contains(csScope, key)
	}
	return inScope
}

func resetResourceInTemplate(changedMap map[string]interface{}, cr string, rules interface{}) map[string]interface{} {
	var rulesForCR map[string]interface{}
	if rules != nil && rules.(map[string]interface{})["spec"] != nil && rules.(map[string]interface{})["spec"].(map[string]interface{})[cr] != nil {
		rulesForCR = rules.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{})
	}
	for key := range changedMap {
		resetChangedMap(key, changedMap[key], rulesForCR, changedMap)
	}
	return changedMap
}

func resetChangedMap(key string, changedMap interface{}, rulesForCR, finalMap map[string]interface{}) {
	var rules interface{}
	if rulesForCR != nil {
		rules = rulesForCR[key]
	}
	if rules != nil {
		switch changedMap := changedMap.(type) {
		case map[string]interface{}:
			if _, ok := rules.(map[string]interface{}); ok {
				rulesRef := rules.(map[string]interface{})
				changedMapRef := changedMap
				for newKey := range changedMapRef {
					resetChangedMap(newKey, changedMapRef[newKey], rulesRef, finalMap[key].(map[string]interface{}))
				}
			}

		default:
			var requiredResetKeys = map[string]bool{
				"replicas": true,
				"cpu":      true,
				"memory":   true,
				// "profile":  true,
			}
			if _, ok := requiredResetKeys[key]; ok {
				delete(finalMap, key)
			}
		}
	}
}
