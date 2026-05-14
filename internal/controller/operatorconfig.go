//
// Copyright 2024 IBM Corporation
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
	"fmt"
	"strings"

	"k8s.io/klog"

	v3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/common"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
)

func (r *CommonServiceReconciler) updateOperatorConfig(ctx context.Context, configList []v3.OperatorConfig) (bool, error) {
	klog.Info("Applying OperatorConfig")

	// Aggregate OperatorConfigs from all CommonService CRs in the cluster
	aggregatedConfigs, err := r.aggregateOperatorConfigsFromAllCRs(ctx)
	if err != nil {
		return false, err
	}

	if len(aggregatedConfigs) == 0 {
		klog.Info("No OperatorConfigs found across all CommonService CRs")
		return true, nil
	}

	// Group configs by operator type using the service lists from constant package
	var edbConfigs, ibmPGConfigs []v3.OperatorConfig
	for _, config := range aggregatedConfigs {
		if config.Replicas == nil {
			continue
		}

		if common.Contains(constant.EDBOperatorServices, config.Name) {
			edbConfigs = append(edbConfigs, config)
			klog.Infof("Found EDB operator config: %s with %d replicas", config.Name, *config.Replicas)
		} else if common.Contains(constant.IBMPGOperatorServices, config.Name) {
			ibmPGConfigs = append(ibmPGConfigs, config)
			klog.Infof("Found IBM PG operator config: %s with %d replicas", config.Name, *config.Replicas)
		} else {
			return false, fmt.Errorf("failed to update OperatorConfig. Operator '%s' is not supported for HA configuration", config.Name)
		}
	}

	if len(edbConfigs) == 0 && len(ibmPGConfigs) == 0 {
		klog.Info("No replicas specified in any CommonService CR OperatorConfigs")
		return true, nil
	}

	// Process EDB operators
	if len(edbConfigs) > 0 {
		if err := r.processOperatorConfigs(ctx, edbConfigs, constant.PostGresOperatorConfig, constant.CloudNativePostgreSQLPackage); err != nil {
			return false, err
		}
	}

	// Process IBM PG operators
	if len(ibmPGConfigs) > 0 {
		if err := r.processOperatorConfigs(ctx, ibmPGConfigs, constant.IBMPGOperatorConfig, constant.IBMPGOperatorPackage); err != nil {
			return false, err
		}
	}

	return false, nil
}

// processOperatorConfigs processes a group of operator configs and applies them
func (r *CommonServiceReconciler) processOperatorConfigs(ctx context.Context, configs []v3.OperatorConfig, configTemplate string, packageName string) error {
	if len(configs) == 0 {
		return nil
	}

	// Validate package name and get corresponding OperatorConfig name
	var operatorConfigName string
	if packageName == constant.CloudNativePostgreSQLPackage {
		operatorConfigName = constant.CloudNativePostgreSQLOperatorConfigName
	} else if packageName == constant.IBMPGOperatorPackage {
		operatorConfigName = constant.IBMPGOperatorConfigName
	} else {
		return fmt.Errorf("unsupported package name: %s", packageName)
	}

	// Use the aggregated replica value (maximum across all CRs)
	// Find the first non-nil replica count defensively instead of assuming configs[0] is always valid.
	var replicas *int32
	for _, config := range configs {
		if config.Replicas != nil {
			replicas = config.Replicas
			break
		}
	}
	if replicas == nil {
		return fmt.Errorf("invalid config for %s: replicas is nil", operatorConfigName)
	}
	replicaCount := *replicas
	klog.Infof("Applying OperatorConfig for %s with %d replicas (aggregated from all CommonService CRs)", packageName, replicaCount)
	replacer := strings.NewReplacer("placeholder-size", fmt.Sprintf("%d", replicaCount))
	updatedConfig := replacer.Replace(configTemplate)
	klog.V(2).Infof("OperatorConfig to be applied for %s will be: %v", packageName, updatedConfig)

	if err := r.Bootstrap.InstallOrUpdateOperatorConfig(updatedConfig, true); err != nil {
		return err
	}
	return nil
}

// aggregateOperatorConfigsFromAllCRs collects and merges OperatorConfigs from all CommonService CRs
// For replica values, it takes the maximum value across all CRs
func (r *CommonServiceReconciler) aggregateOperatorConfigsFromAllCRs(ctx context.Context) ([]v3.OperatorConfig, error) {
	csObjectList := &v3.CommonServiceList{}
	if err := r.Client.List(ctx, csObjectList); err != nil {
		klog.Errorf("Failed to list CommonService CRs: %v", err)
		return nil, err
	}

	// Map to track the maximum replica value for each operator
	operatorConfigMap := make(map[string]*v3.OperatorConfig)

	for _, cs := range csObjectList.Items {
		if cs.GetDeletionTimestamp() != nil {
			klog.V(2).Infof("Skipping CommonService CR %s/%s (being deleted)", cs.Namespace, cs.Name)
			continue
		}

		if cs.Spec.OperatorConfigs == nil {
			continue
		}

		klog.Infof("Processing OperatorConfigs from CommonService CR %s/%s", cs.Namespace, cs.Name)
		for _, config := range cs.Spec.OperatorConfigs {
			if config.Replicas == nil {
				klog.V(2).Infof("Skipping OperatorConfig %s from CR %s/%s (no replicas specified)", config.Name, cs.Namespace, cs.Name)
				continue
			}

			existingConfig, exists := operatorConfigMap[config.Name]
			if !exists {
				configCopy := config
				operatorConfigMap[config.Name] = &configCopy
				klog.Infof("Added OperatorConfig %s with %d replicas from CR %s/%s", config.Name, *config.Replicas, cs.Namespace, cs.Name)
			} else {
				// Take the maximum replica value
				if *config.Replicas > *existingConfig.Replicas {
					existingConfig.Replicas = config.Replicas
					klog.Infof("Updated OperatorConfig %s to %d replicas (from CR %s/%s)", config.Name, *config.Replicas, cs.Namespace, cs.Name)
				} else {
					klog.V(2).Infof("Keeping existing replica value %d for OperatorConfig %s (CR %s/%s has %d)", *existingConfig.Replicas, config.Name, cs.Namespace, cs.Name, *config.Replicas)
				}
			}
		}
	}

	// Convert map to slice
	result := make([]v3.OperatorConfig, 0, len(operatorConfigMap))
	for _, config := range operatorConfigMap {
		result = append(result, *config)
	}

	klog.Infof("Aggregated %d OperatorConfig(s) from %d CommonService CR(s)", len(result), len(csObjectList.Items))
	return result, nil
}
