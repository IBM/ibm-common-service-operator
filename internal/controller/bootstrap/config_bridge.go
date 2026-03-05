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
	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
)

// ConfigMergerFunc is a function type that merges base config with CommonService configs
// This allows bootstrap to use controller merge logic without import cycles
type ConfigMergerFunc func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error)

// configMerger holds the merger function set by the reconciler
var configMerger ConfigMergerFunc

// SetConfigMerger sets the configuration merger function
// Called during reconciler initialization to inject the merge logic
func SetConfigMerger(merger ConfigMergerFunc) {
	configMerger = merger
}

// mergeConfigs calls the injected merger function if available
func (b *Bootstrap) mergeConfigs(baseConfig string, cs *apiv3.CommonService) (string, error) {
	if configMerger != nil {
		return configMerger(baseConfig, cs, b.CSData.ServicesNs)
	}
	// If no merger is set, return base config unchanged
	return baseConfig, nil
}
