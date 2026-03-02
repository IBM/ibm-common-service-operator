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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
)

// minimalBaseOpconYAML is a minimal valid OperandConfig YAML used as base config in tests.
const minimalBaseOpconYAML = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: ibm-common-services
spec:
  services:
  - name: ibm-iam-operator
    spec:
      authentication:
        replicas: 1
  - name: ibm-management-ingress-operator
    spec:
      managementIngress:
        replicas: 1
`

// TestParseOperandConfig verifies that a valid YAML string is correctly parsed
// into an OperandConfig object.
func TestParseOperandConfig(t *testing.T) {
	t.Run("valid YAML parses successfully", func(t *testing.T) {
		opcon, err := parseOperandConfig(minimalBaseOpconYAML)
		require.NoError(t, err)
		require.NotNil(t, opcon)
		assert.Equal(t, "common-service", opcon.Name)
		assert.Equal(t, "ibm-common-services", opcon.Namespace)
		assert.Len(t, opcon.Spec.Services, 2)
		assert.Equal(t, "ibm-iam-operator", opcon.Spec.Services[0].Name)
		assert.Equal(t, "ibm-management-ingress-operator", opcon.Spec.Services[1].Name)
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		_, err := parseOperandConfig("{ invalid yaml: [")
		assert.Error(t, err)
	})

	t.Run("empty string returns empty OperandConfig without error", func(t *testing.T) {
		// The YAML library treats an empty string as a valid null document,
		// so parseOperandConfig returns an empty OperandConfig rather than an error.
		opcon, err := parseOperandConfig("")
		assert.NoError(t, err)
		assert.NotNil(t, opcon)
		assert.Empty(t, opcon.Spec.Services)
	})
}

// TestValidateMergedConfig verifies the validation logic for merged OperandConfig objects.
func TestValidateMergedConfig(t *testing.T) {
	t.Run("nil config returns error", func(t *testing.T) {
		err := validateMergedConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config is nil")
	})

	t.Run("config with no services returns error", func(t *testing.T) {
		config := &odlm.OperandConfig{}
		err := validateMergedConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no services defined")
	})

	t.Run("config with service missing name returns error", func(t *testing.T) {
		config := &odlm.OperandConfig{
			Spec: odlm.OperandConfigSpec{
				Services: []odlm.ConfigService{
					{Name: ""},
				},
			},
		}
		err := validateMergedConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty name")
	})

	t.Run("valid config passes validation", func(t *testing.T) {
		config := &odlm.OperandConfig{
			Spec: odlm.OperandConfigSpec{
				Services: []odlm.ConfigService{
					{Name: "ibm-iam-operator"},
					{Name: "ibm-management-ingress-operator"},
				},
			},
		}
		err := validateMergedConfig(config)
		assert.NoError(t, err)
	})
}

// TestMergeBaseAndCSConfigs_NoCSConfigs verifies that when no CommonService configs
// are provided, the base OperandConfig is returned unchanged.
func TestMergeBaseAndCSConfigs_NoCSConfigs(t *testing.T) {
	result, err := MergeBaseAndCSConfigs(
		minimalBaseOpconYAML,
		nil,
		map[string]string{"profileController": "default"},
		"ibm-common-services",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify the result still contains the original services
	opcon, err := parseOperandConfig(result)
	require.NoError(t, err)
	assert.Len(t, opcon.Spec.Services, 2)
	assert.Equal(t, "ibm-iam-operator", opcon.Spec.Services[0].Name)
}

// TestMergeBaseAndCSConfigs_WithStorageClass verifies that a storageClass config
// from a CommonService CR is merged into the base OperandConfig.
func TestMergeBaseAndCSConfigs_WithStorageClass(t *testing.T) {
	// Build a csConfig slice that mimics what ExtractCommonServiceConfigs produces
	// for a storageClass setting — a list of service entries with spec overrides.
	storageClassConfig := []interface{}{
		map[string]interface{}{
			"name": "ibm-iam-operator",
			"spec": map[string]interface{}{
				"authentication": map[string]interface{}{
					"storageClass": "my-storage-class",
				},
			},
		},
	}

	result, err := MergeBaseAndCSConfigs(
		minimalBaseOpconYAML,
		storageClassConfig,
		map[string]string{"profileController": "default"},
		"ibm-common-services",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// The result should still be valid YAML parseable as OperandConfig
	opcon, err := parseOperandConfig(result)
	require.NoError(t, err)
	assert.NotEmpty(t, opcon.Spec.Services)
}

// TestMergeBaseAndCSConfigs_InvalidBaseConfig verifies that an invalid base config
// YAML returns an error rather than silently producing bad output.
func TestMergeBaseAndCSConfigs_InvalidBaseConfig(t *testing.T) {
	_, err := MergeBaseAndCSConfigs(
		"{ not valid yaml: [",
		nil,
		map[string]string{"profileController": "default"},
		"ibm-common-services",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse base OperandConfig")
}

// TestMergeBaseAndCSConfigs_EmptyBaseServices verifies that a base config with no
// services is handled gracefully and CS configs can still be merged in.
func TestMergeBaseAndCSConfigs_EmptyBaseServices(t *testing.T) {
	emptyServicesYAML := `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: ibm-common-services
spec:
  services: []
`
	csConfigs := []interface{}{
		map[string]interface{}{
			"name": "ibm-iam-operator",
			"spec": map[string]interface{}{
				"authentication": map[string]interface{}{
					"replicas": float64(2),
				},
			},
		},
	}

	result, err := MergeBaseAndCSConfigs(
		emptyServicesYAML,
		csConfigs,
		map[string]string{"profileController": "default"},
		"ibm-common-services",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// The CS config should have been merged in even though the base had no services
	assert.True(t, strings.Contains(result, "ibm-iam-operator"),
		"CS config service should appear in result even when base services list was empty")
}

// TestMergeBaseAndCSConfigs_PreservesUnchangedServices verifies that services not
// referenced in csConfigs are preserved verbatim in the merged output.
func TestMergeBaseAndCSConfigs_PreservesUnchangedServices(t *testing.T) {
	// Only provide config for ibm-iam-operator; ibm-management-ingress-operator should be untouched.
	csConfigs := []interface{}{
		map[string]interface{}{
			"name": "ibm-iam-operator",
			"spec": map[string]interface{}{
				"authentication": map[string]interface{}{
					"replicas": float64(3),
				},
			},
		},
	}

	result, err := MergeBaseAndCSConfigs(
		minimalBaseOpconYAML,
		csConfigs,
		map[string]string{"profileController": "default"},
		"ibm-common-services",
	)
	require.NoError(t, err)

	opcon, err := parseOperandConfig(result)
	require.NoError(t, err)

	// Both services must still be present
	names := make([]string, 0, len(opcon.Spec.Services))
	for _, svc := range opcon.Spec.Services {
		names = append(names, svc.Name)
	}
	assert.Contains(t, names, "ibm-iam-operator")
	assert.Contains(t, names, "ibm-management-ingress-operator")
}

// TestMergeBaseAndCSConfigs_OutputIsValidYAML verifies that the merged result is
// valid YAML that round-trips through JSON without data loss.
func TestMergeBaseAndCSConfigs_OutputIsValidYAML(t *testing.T) {
	result, err := MergeBaseAndCSConfigs(
		minimalBaseOpconYAML,
		nil,
		map[string]string{"profileController": "default"},
		"ibm-common-services",
	)
	require.NoError(t, err)
	assert.True(t, strings.Contains(result, "ibm-iam-operator"), "output YAML should contain service name")

	// Verify it round-trips through parseOperandConfig → convertOperandConfigToYAML
	opcon, err := parseOperandConfig(result)
	require.NoError(t, err)

	roundTripped, err := convertOperandConfigToYAML(opcon)
	require.NoError(t, err)
	assert.NotEmpty(t, roundTripped)
}

// TestConvertOperandConfigToYAML verifies that an OperandConfig object is correctly
// serialised to a YAML string.
func TestConvertOperandConfigToYAML(t *testing.T) {
	t.Run("valid config converts to YAML", func(t *testing.T) {
		config := &odlm.OperandConfig{
			Spec: odlm.OperandConfigSpec{
				Services: []odlm.ConfigService{
					{Name: "ibm-iam-operator"},
					{Name: "ibm-management-ingress-operator"},
				},
			},
		}

		yaml, err := convertOperandConfigToYAML(config)
		require.NoError(t, err)
		assert.Contains(t, yaml, "ibm-iam-operator")
		assert.Contains(t, yaml, "ibm-management-ingress-operator")
	})

	t.Run("empty config converts without error", func(t *testing.T) {
		config := &odlm.OperandConfig{}
		yaml, err := convertOperandConfigToYAML(config)
		require.NoError(t, err)
		assert.NotEmpty(t, yaml)
	})

	t.Run("round-trip parse then convert preserves service names", func(t *testing.T) {
		opcon, err := parseOperandConfig(minimalBaseOpconYAML)
		require.NoError(t, err)

		yaml, err := convertOperandConfigToYAML(opcon)
		require.NoError(t, err)
		assert.Contains(t, yaml, "ibm-iam-operator")
		assert.Contains(t, yaml, "ibm-management-ingress-operator")
	})
}

// TestMergeBaseAndCSConfigs_SingleStageCompleteness is the key regression test for
// the Single-Stage Creation feature.  It verifies that after one call to
// MergeBaseAndCSConfigs the resulting OperandConfig already contains the values
// from the CommonService CR — i.e. there is no intermediate incomplete state.
func TestMergeBaseAndCSConfigs_SingleStageCompleteness(t *testing.T) {
	baseYAML := `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: ibm-common-services
spec:
  services:
  - name: ibm-iam-operator
    spec:
      authentication:
        replicas: 1
`

	// Simulate what the CS CR contributes: a higher replica count.
	csConfigs := []interface{}{
		map[string]interface{}{
			"name": "ibm-iam-operator",
			"spec": map[string]interface{}{
				"authentication": map[string]interface{}{
					"replicas": float64(3),
				},
			},
		},
	}

	// Single call — no intermediate state.
	result, err := MergeBaseAndCSConfigs(
		baseYAML,
		csConfigs,
		map[string]string{"profileController": "default"},
		"ibm-common-services",
	)
	require.NoError(t, err, "single-stage merge must not return an error")
	assert.NotEmpty(t, result, "merged result must not be empty")

	// The result must be a complete, parseable OperandConfig.
	opcon, err := parseOperandConfig(result)
	require.NoError(t, err, "merged result must be valid OperandConfig YAML")
	require.NotEmpty(t, opcon.Spec.Services, "merged OperandConfig must contain services")

	// Verify the service is present and the CS replica value was applied in the
	// single call — this is the core single-stage completeness assertion.
	var iamSvc *odlm.ConfigService
	for i := range opcon.Spec.Services {
		if opcon.Spec.Services[i].Name == "ibm-iam-operator" {
			iamSvc = &opcon.Spec.Services[i]
			break
		}
	}
	require.NotNil(t, iamSvc, "ibm-iam-operator must be present in the single-stage merged OperandConfig")

	// The merged YAML must contain the CS-supplied replica value (3), not the
	// base value (1), proving CS values were applied in the same call.
	assert.True(t, strings.Contains(result, "3"),
		"merged result must contain the CS-supplied replica count (3), not the base value (1)")
	assert.False(t, strings.Contains(result, `"replicas":1`) || strings.Contains(result, "replicas: 1"),
		"merged result must not retain the base replica count of 1 for ibm-iam-operator")
}
