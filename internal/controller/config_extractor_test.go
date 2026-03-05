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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
)

const testServicesNs = "ibm-common-services"

// newCS is a helper that returns a minimal CommonService CR.
func newCS() *apiv3.CommonService {
	return &apiv3.CommonService{}
}

// rawJSON is a helper that wraps a JSON-encoded value in an ExtensionWithMarker.
func rawJSON(t *testing.T, v interface{}) apiv3.ExtensionWithMarker {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return apiv3.ExtensionWithMarker{RawExtension: runtime.RawExtension{Raw: b}}
}

// TestExtractCommonServiceConfigs_EmptyCS verifies that an empty CommonService CR
// produces no configs and a default profileController mapping.
func TestExtractCommonServiceConfigs_EmptyCS(t *testing.T) {
	cs := newCS()
	configs, mapping, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	assert.Empty(t, configs, "empty CS should produce no configs")
	assert.Equal(t, "default", mapping["profileController"])
}

// TestExtractCommonServiceConfigs_StorageClass verifies that a storageClass value
// in the CS spec is extracted into the configs slice.
func TestExtractCommonServiceConfigs_StorageClass(t *testing.T) {
	cs := newCS()
	cs.Spec.StorageClass = "my-storage-class"

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs, "storageClass should produce at least one config entry")

	// At least one entry should reference the storage class value
	found := false
	for _, c := range configs {
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "my-storage-class") {
			found = true
			break
		}
	}
	assert.True(t, found, "extracted configs should contain the storageClass value")
}

// TestExtractCommonServiceConfigs_RouteHost verifies that a routeHost value is
// extracted into the configs slice.
func TestExtractCommonServiceConfigs_RouteHost(t *testing.T) {
	cs := newCS()
	cs.Spec.RouteHost = "cp-console.apps.example.com"

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs)

	found := false
	for _, c := range configs {
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "cp-console.apps.example.com") {
			found = true
			break
		}
	}
	assert.True(t, found, "extracted configs should contain the routeHost value")
}

// TestExtractCommonServiceConfigs_DefaultAdminUser verifies that a defaultAdminUser
// value is extracted into the configs slice.
func TestExtractCommonServiceConfigs_DefaultAdminUser(t *testing.T) {
	cs := newCS()
	cs.Spec.DefaultAdminUser = "myadmin"

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs)

	found := false
	for _, c := range configs {
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "myadmin") {
			found = true
			break
		}
	}
	assert.True(t, found, "extracted configs should contain the defaultAdminUser value")
}

// TestExtractCommonServiceConfigs_FipsEnabled verifies that fipsEnabled=true is
// extracted into the configs slice.
func TestExtractCommonServiceConfigs_FipsEnabled(t *testing.T) {
	cs := newCS()
	cs.Spec.FipsEnabled = true

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs)

	found := false
	for _, c := range configs {
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "true") {
			found = true
			break
		}
	}
	assert.True(t, found, "extracted configs should contain fipsEnabled=true")
}

// TestExtractCommonServiceConfigs_ProfileController verifies that a custom
// profileController is reflected in the serviceControllerMapping.
func TestExtractCommonServiceConfigs_ProfileController(t *testing.T) {
	cs := newCS()
	cs.Spec.ProfileController = "turbo"

	_, mapping, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	assert.Equal(t, "turbo", mapping["profileController"])
}

// TestExtractCommonServiceConfigs_SizeSmall verifies that the "small" size profile
// produces a non-empty configs slice.
func TestExtractCommonServiceConfigs_SizeSmall(t *testing.T) {
	cs := newCS()
	cs.Spec.Size = "small"

	configs, mapping, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	assert.NotEmpty(t, configs, "small size profile should produce configs")
	assert.Equal(t, "default", mapping["profileController"])
}

// TestExtractCommonServiceConfigs_SizeMedium verifies that the "medium" size profile
// produces a non-empty configs slice.
func TestExtractCommonServiceConfigs_SizeMedium(t *testing.T) {
	cs := newCS()
	cs.Spec.Size = "medium"

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	assert.NotEmpty(t, configs, "medium size profile should produce configs")
}

// TestExtractCommonServiceConfigs_SizeLarge verifies that the "large" size profile
// produces a non-empty configs slice.
func TestExtractCommonServiceConfigs_SizeLarge(t *testing.T) {
	cs := newCS()
	cs.Spec.Size = "large"

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	assert.NotEmpty(t, configs, "large size profile should produce configs")
}

// TestExtractCommonServiceConfigs_CustomServices verifies that custom service
// entries in the CS spec are extracted and the managementStrategy is captured
// in the serviceControllerMapping.
func TestExtractCommonServiceConfigs_CustomServices(t *testing.T) {
	cs := newCS()
	cs.Spec.Services = []apiv3.ServiceConfig{
		{
			Name:               "ibm-iam-operator",
			ManagementStrategy: "turbo",
			Spec: map[string]apiv3.ExtensionWithMarker{
				"authentication": rawJSON(t, map[string]interface{}{"replicas": 2}),
			},
		},
	}

	configs, mapping, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs)

	// The management strategy should be captured
	assert.Equal(t, "turbo", mapping["ibm-iam-operator"])

	// The service name should appear in the configs
	found := false
	for _, c := range configs {
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "ibm-iam-operator") {
			found = true
			break
		}
	}
	assert.True(t, found, "custom service name should appear in extracted configs")
}

// TestExtractCommonServiceConfigs_HugePagesInvalidFormat verifies that an invalid
// hugepage size format returns an error.
func TestExtractCommonServiceConfigs_HugePagesInvalidFormat(t *testing.T) {
	cs := newCS()
	cs.Spec.HugePages = &apiv3.HugePages{
		Enable: true,
		HugePagesSizes: map[string]string{
			"invalid-size": "1Gi",
		},
	}

	_, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hugepage size format")
}

// TestExtractCommonServiceConfigs_HugePagesValid verifies that a valid hugepage
// configuration is extracted without error.
func TestExtractCommonServiceConfigs_HugePagesValid(t *testing.T) {
	cs := newCS()
	cs.Spec.HugePages = &apiv3.HugePages{
		Enable: true,
		HugePagesSizes: map[string]string{
			"hugepages-2Mi": "1Gi",
		},
	}

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs)
}

// TestExtractCommonServiceConfigs_MultipleFeatures verifies that multiple feature
// flags set simultaneously all produce entries in the configs slice.
func TestExtractCommonServiceConfigs_MultipleFeatures(t *testing.T) {
	cs := newCS()
	cs.Spec.StorageClass = "fast-storage"
	cs.Spec.RouteHost = "cp-console.apps.example.com"
	cs.Spec.DefaultAdminUser = "admin2"

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)

	// All three features should contribute entries
	assert.GreaterOrEqual(t, len(configs), 3,
		"three feature flags should produce at least 3 config entries")
}
