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
	"encoding/json"
	"fmt"

	utilyaml "github.com/ghodss/yaml"
	"k8s.io/klog"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/rules"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
)

// MergeBaseAndCSConfigs merges base OperandConfig templates with CommonService configurations
// OperandConfig is created complete on first reconciliation
func MergeBaseAndCSConfigs(
	baseConfig string,
	csConfigs []interface{},
	serviceControllerMapping map[string]string,
	servicesNs string,
) (string, error) {
	klog.Info("Merging base OperandConfig with CommonService configurations")

	// Parse base config YAML to OperandConfig object
	baseOpcon, err := parseOperandConfig(baseConfig)
	if err != nil {
		return "", fmt.Errorf("failed to parse base OperandConfig: %v", err)
	}

	// Get base services from OperandConfig
	baseServices := baseOpcon.Spec.Services
	if baseServices == nil {
		baseServices = []odlm.ConfigService{}
	}

	// Convert to interface slice for merging
	baseServicesInterface := make([]interface{}, len(baseServices))
	for i, svc := range baseServices {
		svcBytes, err := json.Marshal(svc)
		if err != nil {
			return "", fmt.Errorf("failed to marshal base service: %v", err)
		}
		var svcMap map[string]interface{}
		if err := json.Unmarshal(svcBytes, &svcMap); err != nil {
			return "", fmt.Errorf("failed to unmarshal base service: %v", err)
		}
		baseServicesInterface[i] = svcMap
	}

	// Convert configuration rules to slice
	ruleSlice, err := convertStringToSlice(rules.ConfigurationRules)
	if err != nil {
		return "", fmt.Errorf("failed to convert configuration rules: %v", err)
	}

	// Merge CommonService configs into base services using existing merge logic
	mergedServices := mergeCSCRs(baseServicesInterface, csConfigs, ruleSlice, serviceControllerMapping, servicesNs)

	// Convert merged services back to OperandConfig format
	mergedServicesBytes, err := json.Marshal(mergedServices)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merged services: %v", err)
	}

	var configServices []odlm.ConfigService
	if err := json.Unmarshal(mergedServicesBytes, &configServices); err != nil {
		return "", fmt.Errorf("failed to unmarshal merged services: %v", err)
	}

	// Update OperandConfig with merged services
	baseOpcon.Spec.Services = configServices

	// Validate merged configuration
	if err := validateMergedConfig(baseOpcon); err != nil {
		klog.Warningf("Merged configuration validation warning: %v", err)
	}

	// Convert back to YAML string
	mergedYAML, err := convertOperandConfigToYAML(baseOpcon)
	if err != nil {
		return "", fmt.Errorf("failed to convert merged config to YAML: %v", err)
	}

	klog.Info("Successfully merged CommonService configurations with base OperandConfig")
	return mergedYAML, nil
}

// parseOperandConfig parses YAML string to OperandConfig object
func parseOperandConfig(yamlStr string) (*odlm.OperandConfig, error) {
	jsonBytes, err := utilyaml.YAMLToJSON([]byte(yamlStr))
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %v", err)
	}

	var opcon odlm.OperandConfig
	if err := json.Unmarshal(jsonBytes, &opcon); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OperandConfig: %v", err)
	}

	return &opcon, nil
}

// convertOperandConfigToYAML converts OperandConfig object to YAML string
func convertOperandConfigToYAML(opcon *odlm.OperandConfig) (string, error) {
	jsonBytes, err := json.Marshal(opcon)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OperandConfig: %v", err)
	}

	yamlBytes, err := utilyaml.JSONToYAML(jsonBytes)
	if err != nil {
		return "", fmt.Errorf("failed to convert JSON to YAML: %v", err)
	}

	return string(yamlBytes), nil
}

// validateMergedConfig validates the merged OperandConfig
// Returns error if critical validation fails, warning for non-critical issues
func validateMergedConfig(config *odlm.OperandConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if config.Spec.Services == nil || len(config.Spec.Services) == 0 {
		return fmt.Errorf("no services defined in OperandConfig")
	}

	// Validate each service has required fields
	for i, svc := range config.Spec.Services {
		if svc.Name == "" {
			return fmt.Errorf("service at index %d has empty name", i)
		}
	}

	klog.V(2).Infof("Validated merged OperandConfig with %d services", len(config.Spec.Services))
	return nil
}

// MergeConfigs is a convenience function that combines extraction and merging
// Used by bootstrap to create complete OperandConfig in single stage
func MergeConfigs(
	r *CommonServiceReconciler,
	baseConfig string,
	cs *apiv3.CommonService,
) (string, error) {
	// Extract CommonService configurations
	csConfigs, serviceControllerMapping, err := ExtractCommonServiceConfigs(cs, r.Bootstrap.CSData.ServicesNs)
	if err != nil {
		return "", fmt.Errorf("failed to extract CommonService configs: %v", err)
	}

	// Merge with base templates
	mergedConfig, err := MergeBaseAndCSConfigs(
		baseConfig,
		csConfigs,
		serviceControllerMapping,
		r.Bootstrap.CSData.ServicesNs,
	)
	if err != nil {
		return "", fmt.Errorf("failed to merge configurations: %v", err)
	}

	return mergedConfig, nil
}

// CreateMergerFunc creates a ConfigMergerFunc for the given reconciler
// This is used to inject the merge logic into bootstrap without import cycles
func CreateMergerFunc(r *CommonServiceReconciler) func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error) {
	return func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error) {
		return MergeConfigs(r, baseConfig, cs)
	}
}
