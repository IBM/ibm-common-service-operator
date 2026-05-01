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

package bootstrap

import (
	"context"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"k8s.io/klog"
)

// ConfigMergerFunc is a function type that merges base config with CommonService configs
// This allows bootstrap to use controller merge logic without import cycles
type ConfigMergerFunc func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error)

// SetConfigMerger sets the configuration merger function on the Bootstrap instance.
// Called during reconciler initialization to inject the merge logic.
func (b *Bootstrap) SetConfigMerger(merger ConfigMergerFunc) {
	b.configMerger = merger
}

// mergeConfigs calls the injected merger function if available
// It also determines the largest size from ALL CommonService CRs and overrides the current CR's size
func (b *Bootstrap) mergeConfigs(baseConfig string, cs *apiv3.CommonService) (string, error) {
	// First, determine the largest size from all CommonService CRs
	largestSize, err := b.getLargestSizeFromAllCRs(ctx)
	if err != nil {
		return "", err
	}

	// Override the current CR's size with the largest size if found
	originalSize := cs.Spec.Size
	if largestSize != "" {
		cs.Spec.Size = largestSize
		klog.Infof("Bootstrap: Overriding CommonService CR %s/%s size from '%s' to largest size '%s'", cs.Namespace, cs.Name, originalSize, largestSize)
	}

	if b.configMerger != nil {
		return b.configMerger(baseConfig, cs, b.CSData.ServicesNs)
	}
	// If no merger is set, return base config unchanged
	return baseConfig, nil
}

// getLargestSizeFromAllCRs determines the largest size across all CommonService CRs
func (b *Bootstrap) getLargestSizeFromAllCRs(ctx context.Context) (string, error) {
	csObjectList := &apiv3.CommonServiceList{}
	if err := b.Client.List(ctx, csObjectList); err != nil {
		return "", err
	}

	sizePriority := map[string]int{
		"starterset": 0,
		"starter":    0,
		"small":      1,
		"medium":     2,
		"large":      3,
	}

	largestSize := ""
	largestPriority := -1

	for _, cs := range csObjectList.Items {
		if cs.GetDeletionTimestamp() != nil {
			continue
		}

		csSize := cs.Spec.Size
		if priority, ok := sizePriority[csSize]; ok {
			klog.Infof("Bootstrap: CommonService CR %s/%s has size: %s (priority: %d)", cs.Namespace, cs.Name, csSize, priority)
			if priority > largestPriority {
				largestPriority = priority
				largestSize = csSize
				klog.Infof("Bootstrap: New largest size found: %s (priority: %d)", largestSize, largestPriority)
			}
		} else if csSize != "" {
			klog.Infof("Bootstrap: CommonService CR %s/%s has custom size configuration (not a predefined size)", cs.Namespace, cs.Name)
		}
	}

	if largestSize == "" {
		klog.Info("Bootstrap: No predefined size found in any CommonService CR, will use starterset")
		return "starterset", nil
	}

	klog.Infof("Bootstrap: FINAL DECISION - Largest size across all CommonService CRs is: %s (priority: %d)", largestSize, largestPriority)
	return largestSize, nil
}
