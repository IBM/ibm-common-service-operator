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
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"

	routehost "github.com/IBM/ibm-common-service-operator/controllers/routeHost"
	"github.com/IBM/ibm-common-service-operator/controllers/size"
	storageclass "github.com/IBM/ibm-common-service-operator/controllers/storageClass"
)

var (
	clusterScopeOperators = []string{"ibm-cert-manager-operator", "ibm-licensing-operator"}
)

func (r *CommonServiceReconciler) getNewConfigs(cs *unstructured.Unstructured, inScope bool) ([]interface{}, error) {
	var newConfigs []interface{}
	var err error
	// Update storageclass in OperandConfig
	if cs.Object["spec"].(map[string]interface{})["storageClass"] != nil {
		klog.Info("Applying storageClass configuration")
		storageConfig, err := convertStringToSlice(strings.ReplaceAll(storageclass.Template, "placeholder", cs.Object["spec"].(map[string]interface{})["storageClass"].(string)))
		if err != nil {
			return nil, err
		}
		newConfigs = append(newConfigs, storageConfig...)
	}

	// Update routeHost
	if cs.Object["spec"].(map[string]interface{})["routeHost"] != nil {
		klog.Info("Applying routeHost configuration")
		routeHostConfig, err := convertStringToSlice(strings.ReplaceAll(routehost.RouteHostTemplate, "placeholder", cs.Object["spec"].(map[string]interface{})["routeHost"].(string)))
		if err != nil {
			return nil, err
		}
		newConfigs = append(newConfigs, routeHostConfig...)
	}

	klog.Info("Applying size configuration")
	var sizeConfigs []interface{}
	switch cs.Object["spec"].(map[string]interface{})["size"] {
	case "small":
		sizeConfigs, err = applySizeTemplate(cs, size.Small, inScope)
		if err != nil {
			return sizeConfigs, err
		}
	case "medium":
		sizeConfigs, err = applySizeTemplate(cs, size.Medium, inScope)
		if err != nil {
			return sizeConfigs, err
		}
	case "large":
		sizeConfigs, err = applySizeTemplate(cs, size.Large, inScope)
		if err != nil {
			return sizeConfigs, err
		}
	default:
		sizeConfigs = applySizeConfigs(cs, inScope)
	}
	newConfigs = append(newConfigs, sizeConfigs...)

	return newConfigs, nil
}

func applySizeConfigs(cs *unstructured.Unstructured, inScope bool) []interface{} {
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
			dest = append(dest, configSize)
		}
	}

	return dest
}

func applySizeTemplate(cs *unstructured.Unstructured, sizeTemplate string, inScope bool) ([]interface{}, error) {

	var src []interface{}
	if cs.Object["spec"].(map[string]interface{})["services"] != nil {
		src = cs.Object["spec"].(map[string]interface{})["services"].([]interface{})
	}

	// Convert sizes string to slice
	sizes, err := convertStringToSlice(sizeTemplate)
	if err != nil {
		klog.Errorf("convert size to interface slice: %v", err)
		return nil, err
	}

	for _, configSize := range sizes {
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
		config := getItemByName(src, configSize.(map[string]interface{})["name"].(string))
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
	return sizes, nil
}
