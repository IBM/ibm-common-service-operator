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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"k8s.io/klog"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/size"
)

// ExtractCommonServiceConfigs extracts all configurations from CommonService CR
// This function is independent of reconciler context and can be used during bootstrap
func ExtractCommonServiceConfigs(
	cs *apiv3.CommonService,
	servicesNs string,
) ([]interface{}, map[string]string, error) {
	var newConfigs []interface{}

	// Extract feature configurations
	featureConfigs, err := extractFeatureConfigs(cs)
	if err != nil {
		return nil, nil, err
	}
	newConfigs = append(newConfigs, featureConfigs...)

	// Extract size configurations
	sizeConfigs, serviceControllerMapping, err := extractSizeConfigs(cs, servicesNs)
	if err != nil {
		return nil, nil, err
	}
	newConfigs = append(newConfigs, sizeConfigs...)

	return newConfigs, serviceControllerMapping, nil
}

// extractFeatureConfigs handles feature flag extraction
func extractFeatureConfigs(cs *apiv3.CommonService) ([]interface{}, error) {
	var configs []interface{}

	// Extract storageClass configuration
	if cs.Spec.StorageClass != "" {
		klog.Info("Extracting storageClass configuration")
		storageConfig, err := convertStringToSlice(strings.ReplaceAll(constant.StorageClassTemplate, "placeholder", cs.Spec.StorageClass))
		if err != nil {
			return nil, err
		}
		configs = append(configs, storageConfig...)
	}

	// Extract EnableInstanaMetricCollection configuration
	if cs.Spec.EnableInstanaMetricCollection {
		klog.Info("Extracting enableInstanaMetricCollection configuration")
		t := template.Must(template.New("template InstanaEnable").Parse(constant.InstanaEnableTemplate))
		var tmplWriter bytes.Buffer
		instanaEnable := struct {
			InstanaEnable bool
		}{
			InstanaEnable: cs.Spec.EnableInstanaMetricCollection,
		}
		if err := t.Execute(&tmplWriter, instanaEnable); err != nil {
			return nil, err
		}
		instanaConfig, err := convertStringToSlice(tmplWriter.String())
		if err != nil {
			return nil, err
		}
		configs = append(configs, instanaConfig...)
	}

	// Extract AutoScaleConfig configuration
	if cs.Spec.AutoScaleConfig {
		klog.Info("Extracting autoScaleConfig configuration")
		t := template.Must(template.New("template AutoScaleConfigTemplate").Parse(constant.AutoScaleConfigTemplate))
		var tmplWriter bytes.Buffer
		autoScaleConfigEnable := struct {
			AutoScaleConfigEnable bool
		}{
			AutoScaleConfigEnable: cs.Spec.AutoScaleConfig,
		}
		if err := t.Execute(&tmplWriter, autoScaleConfigEnable); err != nil {
			return nil, err
		}
		autoScaleConfig, err := convertStringToSlice(tmplWriter.String())
		if err != nil {
			return nil, err
		}
		configs = append(configs, autoScaleConfig...)
	}

	// Extract routeHost configuration
	if cs.Spec.RouteHost != "" {
		klog.Info("Extracting routeHost configuration")
		routeHostConfig, err := convertStringToSlice(strings.ReplaceAll(constant.RouteHostTemplate, "placeholder", cs.Spec.RouteHost))
		if err != nil {
			return nil, err
		}
		configs = append(configs, routeHostConfig...)
	}

	// Extract default admin username configuration
	if cs.Spec.DefaultAdminUser != "" {
		klog.Info("Extracting default admin username configuration")
		adminUsernameConfig, err := convertStringToSlice(strings.ReplaceAll(constant.DefaultAdminUserTemplate, "placeholder", cs.Spec.DefaultAdminUser))
		if err != nil {
			return nil, err
		}
		configs = append(configs, adminUsernameConfig...)
	}

	// Extract FIPS configuration
	if cs.Spec.FipsEnabled {
		klog.Info("Extracting fips configuration")
		fipsEnabledConfig, err := convertStringToSlice(strings.ReplaceAll(constant.FipsEnabledTemplate, "placeholder", strconv.FormatBool(cs.Spec.FipsEnabled)))
		if err != nil {
			return nil, err
		}
		configs = append(configs, fipsEnabledConfig...)
	}

	// Extract hugepages configuration
	if cs.Spec.HugePages != nil && cs.Spec.HugePages.Enable {
		klog.Info("Extracting hugepages configuration")
		for size, allocation := range cs.Spec.HugePages.HugePagesSizes {
			if !strings.HasPrefix(size, "hugepages-") {
				return nil, fmt.Errorf("invalid hugepage size format: %s", size)
			}
			if allocation == "" {
				allocation = constant.DefaultHugePageAllocation
			}
			replacer := strings.NewReplacer("placeholder1", size, "placeholder2", allocation)
			hugePagesConfig, err := convertStringToSlice(replacer.Replace(constant.HugePagesTemplate))
			if err != nil {
				return nil, err
			}
			configs = append(configs, hugePagesConfig...)
		}
	}

	// Extract API Catalog storageClass configuration
	if cs.Spec.Features != nil && cs.Spec.Features.APICatalog != nil && cs.Spec.Features.APICatalog.StorageClass != "" {
		klog.Info("Extracting API Catalog storageClass configuration")
		storageConfig, err := convertStringToSlice(strings.ReplaceAll(constant.APICatalogTemplate, "placeholder", cs.Spec.Features.APICatalog.StorageClass))
		if err != nil {
			return nil, err
		}
		configs = append(configs, storageConfig...)
	}

	// Extract labels configuration
	if cs.Spec.Labels != nil && len(cs.Spec.Labels) > 0 {
		klog.Info("Extracting label configuration")
		for key, value := range cs.Spec.Labels {
			replacer := strings.NewReplacer("placeholder1", key, "placeholder2", value)
			labelConfig, err := convertStringToSlice(replacer.Replace(constant.ServiceLabelTemplate))
			if err != nil {
				return nil, err
			}
			configs = append(configs, labelConfig...)
		}
	}

	return configs, nil
}

// extractSizeConfigs handles size profile extraction
func extractSizeConfigs(
	cs *apiv3.CommonService,
	servicesNs string,
) ([]interface{}, map[string]string, error) {
	klog.Info("Extracting size configuration")

	serviceControllerMapping := make(map[string]string)
	serviceControllerMapping["profileController"] = "default"
	if cs.Spec.ProfileController != "" {
		serviceControllerMapping["profileController"] = cs.Spec.ProfileController
	}

	var sizeConfigs []interface{}
	var err error

	switch cs.Spec.Size {
	case "starterset", "starter":
		sizeConfigs, serviceControllerMapping, err = extractSizeTemplate(cs, size.StarterSet, serviceControllerMapping, servicesNs)
	case "small":
		sizeConfigs, serviceControllerMapping, err = extractSizeTemplate(cs, size.Small, serviceControllerMapping, servicesNs)
	case "medium":
		sizeConfigs, serviceControllerMapping, err = extractSizeTemplate(cs, size.Medium, serviceControllerMapping, servicesNs)
	case "large", "production":
		sizeConfigs, serviceControllerMapping, err = extractSizeTemplate(cs, size.Large, serviceControllerMapping, servicesNs)
	default:
		sizeConfigs, serviceControllerMapping = extractCustomSizeConfigs(cs, serviceControllerMapping)
	}

	if err != nil {
		return nil, nil, err
	}

	return sizeConfigs, serviceControllerMapping, nil
}

// extractCustomSizeConfigs extracts custom size configurations from services field
func extractCustomSizeConfigs(cs *apiv3.CommonService, serviceControllerMapping map[string]string) ([]interface{}, map[string]string) {
	var dest []interface{}

	if cs.Spec.Services != nil {
		for _, service := range cs.Spec.Services {
			serviceMap := make(map[string]interface{})

			// Convert service name
			serviceMap["name"] = service.Name

			// Convert service spec
			if service.Spec != nil {
				specMap := make(map[string]interface{})
				for key, val := range service.Spec {
					var rawValue interface{}
					if err := json.Unmarshal(val.Raw, &rawValue); err == nil {
						specMap[key] = rawValue
					}
				}
				serviceMap["spec"] = specMap
			}

			// Convert service resources
			if service.Resources != nil {
				var resources []interface{}
				for _, res := range service.Resources {
					var rawResource interface{}
					if err := json.Unmarshal(res.Raw, &rawResource); err == nil {
						resources = append(resources, rawResource)
					}
				}
				serviceMap["resources"] = resources
			}

			// Handle management strategy
			if service.ManagementStrategy != "" {
				serviceMap["managementStrategy"] = service.ManagementStrategy
				serviceControllerMapping[service.Name] = service.ManagementStrategy
			}

			dest = append(dest, serviceMap)
		}
	}

	return dest, serviceControllerMapping
}

// extractSizeTemplate applies a size template and merges with custom services
func extractSizeTemplate(cs *apiv3.CommonService, sizeTemplate string, serviceControllerMapping map[string]string, opconNs string) ([]interface{}, map[string]string, error) {
	// Convert custom services to []interface{}
	var src []interface{}
	if cs.Spec.Services != nil {
		for _, service := range cs.Spec.Services {
			serviceMap := make(map[string]interface{})
			serviceMap["name"] = service.Name

			if service.Spec != nil {
				specMap := make(map[string]interface{})
				for key, val := range service.Spec {
					var rawValue interface{}
					if err := json.Unmarshal(val.Raw, &rawValue); err == nil {
						specMap[key] = rawValue
					}
				}
				serviceMap["spec"] = specMap
			}

			if service.Resources != nil {
				var resources []interface{}
				for _, res := range service.Resources {
					var rawResource interface{}
					if err := json.Unmarshal(res.Raw, &rawResource); err == nil {
						resources = append(resources, rawResource)
					}
				}
				serviceMap["resources"] = resources
			}

			if service.ManagementStrategy != "" {
				serviceMap["managementStrategy"] = service.ManagementStrategy
			}

			src = append(src, serviceMap)
		}
	}

	// Convert size template string to slice
	sizes, err := convertStringToSlice(sizeTemplate)
	if err != nil {
		klog.Errorf("convert size to interface slice: %v", err)
		return nil, nil, err
	}

	// Merge size template with custom services using the same logic as applySizeTemplate
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
		// Merge spec
		if configSize.(map[string]interface{})["spec"] != nil && config.(map[string]interface{})["spec"] != nil {
			for cr, mergedSize := range mergeSizeProfile(configSize.(map[string]interface{})["spec"].(map[string]interface{}), config.(map[string]interface{})["spec"].(map[string]interface{})) {
				configSize.(map[string]interface{})["spec"].(map[string]interface{})[cr] = mergedSize
			}
		}
		// Merge resources
		if configSize.(map[string]interface{})["resources"] != nil && config.(map[string]interface{})["resources"] != nil {
			for j, res := range configSize.(map[string]interface{})["resources"].([]interface{}) {
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
				if apiVersion == "" || kind == "" || name == "" {
					klog.Warningf("Skipping merging resource %s/%s/%s/%s, because apiVersion, kind or name is not set", apiVersion, kind, name, namespace)
					continue
				}
				if namespace == "" {
					namespace = opconNs
				}
				newResource := getItemByGVKNameNamespace(config.(map[string]interface{})["resources"].([]interface{}), opconNs, apiVersion, kind, name, namespace)
				if newResource != nil {
					configSize.(map[string]interface{})["resources"].([]interface{})[j] = mergeSizeProfile(res.(map[string]interface{}), newResource.(map[string]interface{}))
				}
			}
		}
		sizes[i] = configSize
	}

	return sizes, serviceControllerMapping, nil
}

// Made with Bob
