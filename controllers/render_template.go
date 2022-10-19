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
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"

	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	"github.com/IBM/ibm-common-service-operator/controllers/size"
)

var (
	clusterScopeOperators = []string{"ibm-cert-manager-operator", "ibm-licensing-operator"}
)

func (r *CommonServiceReconciler) getNewConfigs(cs *unstructured.Unstructured, inScope bool) ([]interface{}, map[string]string, error) {
	var newConfigs []interface{}
	var err error
	// Update storageclass in OperandConfig
	if cs.Object["spec"].(map[string]interface{})["storageClass"] != nil {
		klog.Info("Applying storageClass configuration")
		storageConfig, err := convertStringToSlice(strings.ReplaceAll(constant.StorageClassTemplate, "placeholder", cs.Object["spec"].(map[string]interface{})["storageClass"].(string)))
		if err != nil {
			return nil, nil, err
		}
		newConfigs = append(newConfigs, storageConfig...)
	}

	// Update routeHost
	if cs.Object["spec"].(map[string]interface{})["routeHost"] != nil {
		klog.Info("Applying routeHost configuration")
		routeHostConfig, err := convertStringToSlice(strings.ReplaceAll(constant.RouteHostTemplate, "placeholder", cs.Object["spec"].(map[string]interface{})["routeHost"].(string)))
		if err != nil {
			return nil, nil, err
		}
		newConfigs = append(newConfigs, routeHostConfig...)
	}

	// Update multipleInstancesEnabled when multi-instances
	if r.Bootstrap.MultiInstancesEnable {
		klog.Info("Applying multipleInstancesEnabled configuration")
		multipleinstancesenabledConfig, err := convertStringToSlice(strings.ReplaceAll(constant.MultipleInstancesEnabledTemplate, "placeholder", "true"))
		if err != nil {
			return nil, nil, err
		}
		newConfigs = append(newConfigs, multipleinstancesenabledConfig...)
	}

	// set fipsEnabled to false by default
	fipsEnabled := false
	// if there is a fipsEnabled field for overall
	if enabled := cs.Object["spec"].(map[string]interface{})["fipsEnabled"]; enabled != nil {
		klog.Info("Applying fipsEnabled")
		fipsEnabled = enabled.(bool)
	}
	// update config for all three services
	fipsEnabledConfig, err := convertStringToSlice(strings.ReplaceAll(constant.FipsEnabledTemplate, "placeholder", strconv.FormatBool(fipsEnabled)))
	if err != nil {
		return nil, nil, err
	}
	newConfigs = append(newConfigs, fipsEnabledConfig...)

	// Update storageclass for API Catalog
	if features := cs.Object["spec"].(map[string]interface{})["features"]; features != nil {
		if apiCatalog := features.(map[string]interface{})["apiCatalog"]; apiCatalog != nil {
			if storageClass := apiCatalog.(map[string]interface{})["storageClass"]; storageClass != nil {
				storageConfig, err := convertStringToSlice(strings.ReplaceAll(constant.APICatalogTemplate, "placeholder", storageClass.(string)))
				if err != nil {
					return nil, nil, err
				}
				newConfigs = append(newConfigs, storageConfig...)
			}
		}
	}

	klog.Info("Applying size configuration")
	var sizeConfigs []interface{}
	serviceControllerMapping := make(map[string]string)
	serviceControllerMapping["profileController"] = "default"
	if controller, ok := cs.Object["spec"].(map[string]interface{})["profileController"]; ok {
		serviceControllerMapping["profileController"] = controller.(string)
	}

	switch cs.Object["spec"].(map[string]interface{})["size"] {
	case "starterset", "starter":
		sizeConfigs, serviceControllerMapping, err = applySizeTemplate(cs, size.StarterSet, serviceControllerMapping, inScope)
		if err != nil {
			return sizeConfigs, serviceControllerMapping, err
		}
	case "small":
		sizeConfigs, serviceControllerMapping, err = applySizeTemplate(cs, size.Small, serviceControllerMapping, inScope)
		if err != nil {
			return sizeConfigs, serviceControllerMapping, err
		}
	case "medium":
		sizeConfigs, serviceControllerMapping, err = applySizeTemplate(cs, size.Medium, serviceControllerMapping, inScope)
		if err != nil {
			return sizeConfigs, serviceControllerMapping, err
		}
	case "large", "production":
		sizeConfigs, serviceControllerMapping, err = applySizeTemplate(cs, size.Large, serviceControllerMapping, inScope)
		if err != nil {
			return sizeConfigs, serviceControllerMapping, err
		}
	default:
		sizeConfigs, serviceControllerMapping = applySizeConfigs(cs, serviceControllerMapping, inScope)
	}
	newConfigs = append(newConfigs, sizeConfigs...)

	return newConfigs, serviceControllerMapping, nil
}

func applySizeConfigs(cs *unstructured.Unstructured, serviceControllerMapping map[string]string, inScope bool) ([]interface{}, map[string]string) {
	var dest []interface{}
	if cs.Object["spec"].(map[string]interface{})["services"] != nil {
		for _, configSize := range cs.Object["spec"].(map[string]interface{})["services"].([]interface{}) {
			if !inScope {
				isClusterScope := false
				for _, operator := range clusterScopeOperators {
					if configSize.(map[string]interface{})["name"].(string) == operator {
						isClusterScope = true
						break
					}
				}
				if !isClusterScope {
					continue
				}
			}
			if controller, ok := configSize.(map[string]interface{})["managementStrategy"]; ok {
				serviceControllerMapping[configSize.(map[string]interface{})["name"].(string)] = controller.(string)
			}
			dest = append(dest, configSize)
		}
	}

	return dest, serviceControllerMapping
}

func applySizeTemplate(cs *unstructured.Unstructured, sizeTemplate string, serviceControllerMapping map[string]string, inScope bool) ([]interface{}, map[string]string, error) {

	var src []interface{}
	if cs.Object["spec"].(map[string]interface{})["services"] != nil {
		src = cs.Object["spec"].(map[string]interface{})["services"].([]interface{})
	}

	// Convert sizes string to slice
	sizes, err := convertStringToSlice(sizeTemplate)
	if err != nil {
		klog.Errorf("convert size to interface slice: %v", err)
		return nil, nil, err
	}
	var newSizes []interface{}
	if !inScope {
		// delete all namespace-scoped operator's template
		for _, configSize := range sizes {
			for _, operator := range clusterScopeOperators {
				if configSize.(map[string]interface{})["name"].(string) == operator {
					newSizes = append(newSizes, configSize)
				}
			}
		}
		sizes = newSizes
	}

	for _, configSize := range sizes {
		config := getItemByName(src, configSize.(map[string]interface{})["name"].(string))
		if config == nil {
			continue
		}
		if controller, ok := config.(map[string]interface{})["managementStrategy"]; ok {
			serviceControllerMapping[configSize.(map[string]interface{})["name"].(string)] = controller.(string)
		}
		if configSize == nil {
			continue
		}
		for cr, size := range mergeSizeProfile(configSize.(map[string]interface{})["spec"].(map[string]interface{}), config.(map[string]interface{})["spec"].(map[string]interface{})) {
			configSize.(map[string]interface{})["spec"].(map[string]interface{})[cr] = size
		}
	}
	return sizes, serviceControllerMapping, nil
}
