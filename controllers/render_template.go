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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	util "github.com/IBM/ibm-common-service-operator/v4/controllers/common"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/constant"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/size"
)

func (r *CommonServiceReconciler) getNewConfigs(cs *unstructured.Unstructured) ([]interface{}, map[string]string, error) {
	var newConfigs []interface{}
	var err error

	csObject := &apiv3.CommonService{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cs.GetName(), Namespace: cs.GetNamespace()}, csObject); err != nil {
		return nil, nil, err
	}

	// Update storageclass in OperandConfig
	if cs.Object["spec"].(map[string]interface{})["storageClass"] != nil {
		klog.Info("Applying storageClass configuration")
		storageConfig, err := convertStringToSlice(strings.ReplaceAll(constant.StorageClassTemplate, "placeholder", cs.Object["spec"].(map[string]interface{})["storageClass"].(string)))
		if err != nil {
			return nil, nil, err
		}
		newConfigs = append(newConfigs, storageConfig...)
	}

	// Update EnableInstanaMetricCollection in OperandConfig
	if cs.Object["spec"].(map[string]interface{})["enableInstanaMetricCollection"] != nil {
		klog.Info("Applying enableInstanaMetricCollection configuration")

		t := template.Must(template.New("template InstanaEnable").Parse(constant.InstanaEnableTemplate))
		var tmplWriter bytes.Buffer
		instanaEnable := struct {
			InstanaEnable bool
		}{
			InstanaEnable: cs.Object["spec"].(map[string]interface{})["enableInstanaMetricCollection"].(bool),
		}
		if err := t.Execute(&tmplWriter, instanaEnable); err != nil {
			return nil, nil, err
		}
		s := tmplWriter.String()
		instanaConfig, err := convertStringToSlice(s)
		if err != nil {
			return nil, nil, err
		}
		newConfigs = append(newConfigs, instanaConfig...)
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

	// Specify default Admin Username
	if cs.Object["spec"].(map[string]interface{})["defaultAdminUser"] != nil {
		klog.Info("Applying the default admin username")
		adminUsernameConfig, err := convertStringToSlice(strings.ReplaceAll(constant.DefaultAdminUserTemplate, "placeholder", cs.Object["spec"].(map[string]interface{})["defaultAdminUser"].(string)))
		if err != nil {
			return nil, nil, err
		}
		newConfigs = append(newConfigs, adminUsernameConfig...)
	}

	// if there is a fipsEnabled field for overall
	if enabled := cs.Object["spec"].(map[string]interface{})["fipsEnabled"]; enabled != nil {
		klog.Info("Applying fips configuration")
		// update config for all three services
		fipsEnabledConfig, err := convertStringToSlice(strings.ReplaceAll(constant.FipsEnabledTemplate, "placeholder", strconv.FormatBool(enabled.(bool))))
		if err != nil {
			return nil, nil, err
		}
		newConfigs = append(newConfigs, fipsEnabledConfig...)
	}

	// if there is a hugepage setting enabled
	if hugespages := cs.Object["spec"].(map[string]interface{})["hugepages"]; hugespages != nil {
		klog.Info("Applying hugepages configuration")
		if enable := hugespages.(map[string]interface{})["enable"]; enable != nil && enable.(bool) {
			hugePagesStruct, err := UnmarshalHugePages(hugespages)
			if err != nil {
				return nil, nil, err
			}
			for size, allocation := range hugePagesStruct.HugePagesSizes {
				if !strings.HasPrefix(size, "hugepages-") {
					return nil, nil, fmt.Errorf("invalid hugepage size format: %s", size)
				}

				if allocation == "" {
					allocation = constant.DefaultHugePageAllocation
				}
				replacer := strings.NewReplacer("placeholder1", size, "placeholder2", allocation)
				hugePagesConfig, err := convertStringToSlice(replacer.Replace(constant.HugePagesTemplate))
				if err != nil {
					return nil, nil, err
				}
				newConfigs = append(newConfigs, hugePagesConfig...)
			}

		}
	}

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

	if labels := cs.Object["spec"].(map[string]interface{})["labels"]; labels != nil {
		klog.Info("Applying label configuration")
		labelset := csObject.Spec.Labels
		for key, value := range labelset {
			replacer := strings.NewReplacer("placeholder1", key, "placeholder2", value)
			labelConfig, err := convertStringToSlice(replacer.Replace(constant.ServiceLabelTemplate))
			if err != nil {
				return nil, nil, err
			}
			newConfigs = append(newConfigs, labelConfig...)
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
		sizeConfigs, serviceControllerMapping, err = applySizeTemplate(cs, size.StarterSet, serviceControllerMapping, r.CSData.ServicesNs)
		if err != nil {
			return sizeConfigs, serviceControllerMapping, err
		}
	case "small":
		sizeConfigs, serviceControllerMapping, err = applySizeTemplate(cs, size.Small, serviceControllerMapping, r.CSData.ServicesNs)
		if err != nil {
			return sizeConfigs, serviceControllerMapping, err
		}
	case "medium":
		sizeConfigs, serviceControllerMapping, err = applySizeTemplate(cs, size.Medium, serviceControllerMapping, r.CSData.ServicesNs)
		if err != nil {
			return sizeConfigs, serviceControllerMapping, err
		}
	case "large", "production":
		sizeConfigs, serviceControllerMapping, err = applySizeTemplate(cs, size.Large, serviceControllerMapping, r.CSData.ServicesNs)
		if err != nil {
			return sizeConfigs, serviceControllerMapping, err
		}
	default:
		sizeConfigs, serviceControllerMapping = applySizeConfigs(cs, serviceControllerMapping)
	}
	newConfigs = append(newConfigs, sizeConfigs...)

	return newConfigs, serviceControllerMapping, nil
}

func applySizeConfigs(cs *unstructured.Unstructured, serviceControllerMapping map[string]string) ([]interface{}, map[string]string) {
	var dest []interface{}

	if cs.Object["spec"].(map[string]interface{})["services"] != nil {
		for _, configSize := range cs.Object["spec"].(map[string]interface{})["services"].([]interface{}) {
			if controller, ok := configSize.(map[string]interface{})["managementStrategy"]; ok {
				serviceControllerMapping[configSize.(map[string]interface{})["name"].(string)] = controller.(string)
			}
			dest = append(dest, configSize)
		}
	}

	return dest, serviceControllerMapping
}

func applySizeTemplate(cs *unstructured.Unstructured, sizeTemplate string, serviceControllerMapping map[string]string, opconNs string) ([]interface{}, map[string]string, error) {

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

	for i, configSize := range sizes {
		if configSize == nil {
			continue
		}
		config := getItemByName(src, configSize.(map[string]interface{})["name"].(string))
		if config == nil {
			continue
		}
		if controller, ok := config.(map[string]interface{})["managementStrategy"]; ok {
			serviceControllerMapping[configSize.(map[string]interface{})["name"].(string)] = controller.(string)
		}
		// check if configSize['spec'] and config['spec'] are not nil
		if configSize.(map[string]interface{})["spec"] != nil && config.(map[string]interface{})["spec"] != nil {
			for cr, size := range mergeSizeProfile(configSize.(map[string]interface{})["spec"].(map[string]interface{}), config.(map[string]interface{})["spec"].(map[string]interface{})) {
				configSize.(map[string]interface{})["spec"].(map[string]interface{})[cr] = size
			}
		}
		// check if configSize['resources'] and config['resources'] are not nil
		if configSize.(map[string]interface{})["resources"] != nil && config.(map[string]interface{})["resources"] != nil {
			// loop through configSize['resources'] and config['resources']
			for i, res := range configSize.(map[string]interface{})["resources"].([]interface{}) {
				var apiVersion, kind, name, namespace string
				if res.(map[string]interface{})["apiVersion"] != nil {
					apiVersion = res.(map[string]interface{})["apiVersion"].(string)
				}
				if res.(map[string]interface{})["kind"] != nil {
					kind = res.(map[string]interface{})["kind"].(string)
				}
				if res.(map[string]interface{})["name"] != nil {
					name = res.(map[string]interface{})["name"].(string)
				}
				if res.(map[string]interface{})["namespace"] != nil {
					namespace = res.(map[string]interface{})["namespace"].(string)
				}
				// check if above 4 fields are all set
				if apiVersion == "" || kind == "" || name == "" {
					klog.Warningf("Skipping merging resource %s/%s/%s/%s, because apiVersion, kind or name is not set", apiVersion, kind, name, namespace)
					continue
				}
				// check if namespace is set, if not, set it to OperandConfig namespace
				if namespace == "" {
					namespace = opconNs
				}
				newConfig := getItemByGVKNameNamespace(config.(map[string]interface{})["resources"].([]interface{}), opconNs, apiVersion, kind, name, namespace)
				if newConfig != nil {
					configSize.(map[string]interface{})["resources"].([]interface{})[i] = mergeSizeProfile(res.(map[string]interface{}), newConfig.(map[string]interface{}))
				}
			}
			sizes[i].(map[string]interface{})["resources"] = configSize.(map[string]interface{})["resources"]
		}
	}
	return sizes, serviceControllerMapping, nil
}

// UnmarshalHugePages unmarshals the hugepages map to HugePages struct
func UnmarshalHugePages(hugespages interface{}) (*apiv3.HugePages, error) {
	hugespagesBytes, err := json.Marshal(hugespages)
	if err != nil {
		return nil, err
	}

	hugePagesStruct := &apiv3.HugePages{}
	if err := json.Unmarshal(hugespagesBytes, hugePagesStruct); err != nil {
		return nil, err
	}

	hugespagesBytesSanitized, err := json.Marshal(util.SanitizeData(hugespages, "string", true))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(hugespagesBytesSanitized, &hugePagesStruct.HugePagesSizes); err != nil {
		return nil, err
	}

	return hugePagesStruct, nil
}
