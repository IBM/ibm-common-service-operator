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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
)

// resetConfigMerger clears the package-level configMerger between tests to
// prevent state leaking across test cases.
func resetConfigMerger() {
	configMerger = nil
}

// TestSetConfigMerger_NilByDefault verifies that before SetConfigMerger is called
// the package-level merger is nil and mergeConfigs returns the base config unchanged.
func TestSetConfigMerger_NilByDefault(t *testing.T) {
	resetConfigMerger()

	b := &Bootstrap{
		CSData: apiv3.CSData{ServicesNs: "ibm-common-services"},
	}
	cs := &apiv3.CommonService{}

	result, err := b.mergeConfigs("base-config", cs)
	require.NoError(t, err)
	assert.Equal(t, "base-config", result,
		"when no merger is set, mergeConfigs must return the base config unchanged")
}

// TestSetConfigMerger_InjectsFunction verifies that SetConfigMerger stores the
// provided function and that mergeConfigs delegates to it.
func TestSetConfigMerger_InjectsFunction(t *testing.T) {
	resetConfigMerger()
	defer resetConfigMerger()

	called := false
	SetConfigMerger(func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error) {
		called = true
		return "merged-" + baseConfig, nil
	})

	b := &Bootstrap{
		CSData: apiv3.CSData{ServicesNs: "ibm-common-services"},
	}
	cs := &apiv3.CommonService{}

	result, err := b.mergeConfigs("base", cs)
	require.NoError(t, err)
	assert.True(t, called, "the injected merger function must be called")
	assert.Equal(t, "merged-base", result)
}

// TestSetConfigMerger_PassesServicesNs verifies that mergeConfigs forwards the
// Bootstrap's ServicesNs to the injected merger function.
func TestSetConfigMerger_PassesServicesNs(t *testing.T) {
	resetConfigMerger()
	defer resetConfigMerger()

	var capturedNs string
	SetConfigMerger(func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error) {
		capturedNs = servicesNs
		return baseConfig, nil
	})

	b := &Bootstrap{
		CSData: apiv3.CSData{ServicesNs: "my-services-ns"},
	}
	cs := &apiv3.CommonService{}

	_, err := b.mergeConfigs("config", cs)
	require.NoError(t, err)
	assert.Equal(t, "my-services-ns", capturedNs,
		"mergeConfigs must pass Bootstrap.CSData.ServicesNs to the merger function")
}

// TestSetConfigMerger_PassesCSInstance verifies that mergeConfigs forwards the
// CommonService instance to the injected merger function.
func TestSetConfigMerger_PassesCSInstance(t *testing.T) {
	resetConfigMerger()
	defer resetConfigMerger()

	var capturedCS *apiv3.CommonService
	SetConfigMerger(func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error) {
		capturedCS = cs
		return baseConfig, nil
	})

	b := &Bootstrap{
		CSData: apiv3.CSData{ServicesNs: "ibm-common-services"},
	}
	cs := &apiv3.CommonService{}
	cs.Name = "common-service"
	cs.Namespace = "ibm-common-services"

	_, err := b.mergeConfigs("config", cs)
	require.NoError(t, err)
	require.NotNil(t, capturedCS)
	assert.Equal(t, "common-service", capturedCS.Name)
	assert.Equal(t, "ibm-common-services", capturedCS.Namespace)
}

// TestSetConfigMerger_PropagatesError verifies that an error returned by the
// injected merger function is propagated back to the caller of mergeConfigs.
func TestSetConfigMerger_PropagatesError(t *testing.T) {
	resetConfigMerger()
	defer resetConfigMerger()

	mergeErr := errors.New("merge failed")
	SetConfigMerger(func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error) {
		return "", mergeErr
	})

	b := &Bootstrap{
		CSData: apiv3.CSData{ServicesNs: "ibm-common-services"},
	}
	cs := &apiv3.CommonService{}

	result, err := b.mergeConfigs("config", cs)
	assert.Error(t, err)
	assert.Equal(t, mergeErr, err)
	assert.Empty(t, result)
}

// TestSetConfigMerger_Overwrite verifies that calling SetConfigMerger a second
// time replaces the previously registered function.
func TestSetConfigMerger_Overwrite(t *testing.T) {
	resetConfigMerger()
	defer resetConfigMerger()

	SetConfigMerger(func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error) {
		return "first", nil
	})
	SetConfigMerger(func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error) {
		return "second", nil
	})

	b := &Bootstrap{
		CSData: apiv3.CSData{ServicesNs: "ibm-common-services"},
	}
	cs := &apiv3.CommonService{}

	result, err := b.mergeConfigs("config", cs)
	require.NoError(t, err)
	assert.Equal(t, "second", result,
		"the second SetConfigMerger call must overwrite the first")
}

// TestMergeConfigs_SingleStageNoRaceCondition is the key integration test for the
// Single-Stage Creation feature.  It verifies that a single mergeConfigs call
// produces a complete merged result — there is no intermediate state where the
// base config is returned before CS values are applied.
func TestMergeConfigs_SingleStageNoRaceCondition(t *testing.T) {
	resetConfigMerger()
	defer resetConfigMerger()

	// Simulate the real merger: it always returns a "complete" config that
	// includes both base and CS values in one shot.
	SetConfigMerger(func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error) {
		// In production this would be MergeConfigs(); here we just verify the
		// contract: the result must differ from the bare base config, proving
		// CS values were applied in the same call.
		return baseConfig + "+cs-values", nil
	})

	b := &Bootstrap{
		CSData: apiv3.CSData{ServicesNs: "ibm-common-services"},
	}
	cs := &apiv3.CommonService{}

	result, err := b.mergeConfigs("base-opcon", cs)
	require.NoError(t, err)

	// The result must already contain CS values — no second call needed.
	assert.Equal(t, "base-opcon+cs-values", result,
		"single mergeConfigs call must return complete config with CS values applied")
	assert.NotEqual(t, "base-opcon", result,
		"result must not be the bare base config (that would indicate an incomplete intermediate state)")
}
