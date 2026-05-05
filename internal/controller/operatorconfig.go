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
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	v3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
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

	// TODO: remove when this feature is generalized to all other operators
	replicasProvided := false
	for _, config := range aggregatedConfigs {
		packageName, err := r.fetchPackageNameFromOpReg(ctx, config.Name)
		if err != nil {
			return false, err
		}
		if packageName != "cloud-native-postgresql" {
			return false, errors.New("failed to update OperatorConfig. This feature is only available for cloud-native-postgresql operator")
		}
		if config.Replicas != nil {
			replicasProvided = true
		}
	}

	if !replicasProvided {
		klog.Info("No replicas specified in any CommonService CR OperatorConfigs")
		return true, nil
	}

	operatorConfig := &odlm.OperatorConfig{}
	if err := r.Reader.Get(ctx, types.NamespacedName{
		Name:      "cloud-native-postgresql-operator-config",
		Namespace: r.Bootstrap.CSData.ServicesNs,
	}, operatorConfig); err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("failed to get OperatorConfig %s/%s: %v", operatorConfig.GetNamespace(), operatorConfig.GetName(), err)
			return true, err
		}
	}

	// Use the aggregated replica value (maximum across all CRs)
	replicas := *aggregatedConfigs[0].Replicas
	klog.Infof("Applying OperatorConfig with %d replicas (aggregated from all CommonService CRs)", replicas)
	replacer := strings.NewReplacer("placeholder-size", fmt.Sprintf("%d", replicas))
	updatedConfig := replacer.Replace(constant.PostGresOperatorConfig)
	klog.V(2).Infof("OperatorConfig to be applied will be: %v", updatedConfig)

	if err := r.Bootstrap.InstallOrUpdateOperatorConfig(updatedConfig, true); err != nil {
		return false, err
	}
	return false, nil
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
		// Skip CRs that are being deleted
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
				// First time seeing this operator config
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

func (r *CommonServiceReconciler) fetchPackageNameFromOpReg(ctx context.Context, name string) (string, error) {
	registry, err := r.GetOperandRegistry(ctx, "common-service", r.CSData.ServicesNs)
	if err != nil {
		return "", err
	}

	for _, r := range registry.Spec.Operators {
		operator := r
		if operator.Name == name {
			return operator.PackageName, nil
		}
	}
	return "", nil
}
