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

// AggregatedConfigMergerFunc merges base config with an already aggregated multi-CR view.
// This lets bootstrap create OperandConfig in its final desired form before ODLM consumes it.
type AggregatedConfigMergerFunc func(baseConfig string, configs []interface{}, serviceControllerMapping map[string]string, servicesNs string) (string, error)

// SetConfigMerger sets the configuration merger function on the Bootstrap instance.
// Called during reconciler initialization to inject the merge logic.
func (b *Bootstrap) SetConfigMerger(merger ConfigMergerFunc) {
	b.configMerger = merger
}

// SetAggregatedConfigMerger sets the aggregated configuration merger function on the Bootstrap instance.
func (b *Bootstrap) SetAggregatedConfigMerger(merger AggregatedConfigMergerFunc) {
	b.aggregatedConfigMerger = merger
}

// mergeConfigs calls the injected merger function if available.
// It also determines the largest size from ALL CommonService CRs and overrides the current CR's size.
// A context.Context is required to propagate cancellation into the client.List call.
// When b.Client is nil (e.g. in unit tests) the size-override step is skipped entirely.
func (b *Bootstrap) mergeConfigs(ctx context.Context, baseConfig string, cs *apiv3.CommonService) (string, error) {
	// Determine the largest size from all CommonService CRs — only when a live client is available.
	// Unit tests create Bootstrap without a client; skipping avoids a nil-pointer panic there.
	if b.Client != nil {
		largestSize, err := b.getLargestSizeFromAllCRs(ctx)
		if err != nil {
			return "", err
		}

		// Work on a shallow copy to avoid mutating the caller's object.
		if largestSize != "" {
			csCopy := *cs
			csCopy.Spec.Size = largestSize
			cs = &csCopy
			klog.Infof("Bootstrap: Overriding CommonService CR %s/%s size from '%s' to largest size '%s'",
				cs.Namespace, cs.Name, largestSize, largestSize)
		}
	}

	if b.configMerger != nil {
		return b.configMerger(baseConfig, cs, b.CSData.ServicesNs)
	}
	// If no merger is set, return base config unchanged
	return baseConfig, nil
}

// mergeAggregatedConfigs calls the injected aggregated merger function if available.
func (b *Bootstrap) mergeAggregatedConfigs(baseConfig string, configs []interface{}, serviceControllerMapping map[string]string) (string, error) {
	if b.aggregatedConfigMerger != nil {
		return b.aggregatedConfigMerger(baseConfig, configs, serviceControllerMapping, b.CSData.ServicesNs)
	}
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
