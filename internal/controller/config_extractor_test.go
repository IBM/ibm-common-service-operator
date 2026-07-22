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
	pgv1 "github.ibm.com/ibm-pg/ibm-pg-types/pkg/api/v1"
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
// produces only autoScaleConfig with default value false and a default profileController mapping.
func TestExtractCommonServiceConfigs_EmptyCS(t *testing.T) {
	cs := newCS()
	configs, mapping, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs, "empty CS should produce autoScaleConfig with default false")
	assert.Equal(t, "default", mapping["profileController"])

	// Verify that autoScaleConfig is set to false by default
	foundAutoScale := false
	for _, c := range configs {
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "\"autoScaleConfig\":false") {
			foundAutoScale = true
			break
		}
	}
	assert.True(t, foundAutoScale, "empty CS should produce autoScaleConfig=false by default")

	// Verify that no other feature configs are present (storageClass, routeHost, etc.)
	configJSON, _ := json.Marshal(configs)
	configStr := string(configJSON)
	assert.NotContains(t, configStr, "storageClass", "empty CS should not contain storageClass config")
	assert.NotContains(t, configStr, "routeHost", "empty CS should not contain routeHost config")
	assert.NotContains(t, configStr, "defaultAdminUser", "empty CS should not contain defaultAdminUser config")
	assert.NotContains(t, configStr, "fipsEnabled", "empty CS should not contain fipsEnabled config")
	assert.NotContains(t, configStr, "hugepages", "empty CS should not contain hugepages config")
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

func TestExtractCommonServiceConfigs_AutoScaleConfigFalse(t *testing.T) {
	cs := newCS()
	falseVal := false
	cs.Spec.AutoScaleConfig = &falseVal

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs)

	found := false
	for _, c := range configs {
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "\"autoScaleConfig\":false") {
			found = true
			break
		}
	}
	assert.True(t, found, "extracted configs should contain autoScaleConfig=false")
}

func TestExtractCommonServiceConfigs_AutoScaleConfigTrue(t *testing.T) {
	cs := newCS()
	trueVal := true
	cs.Spec.AutoScaleConfig = &trueVal

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs)

	found := false
	for _, c := range configs {
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "\"autoScaleConfig\":true") {
			found = true
			break
		}
	}
	assert.True(t, found, "extracted configs should contain autoScaleConfig=true")
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

// TestExtractCommonServiceConfigs_PostgreSQLReplica verifies that CSPostgreSQLReplica
// configuration is correctly extracted into the configs slice.
func TestExtractCommonServiceConfigs_PostgreSQLReplica(t *testing.T) {
	cs := newCS()
	enabled := true
	cs.Spec.CSPostgreSQLReplica = &apiv3.CSPostgreSQLReplicaConfig{
		Replica: pgv1.ReplicaClusterConfiguration{
			Enabled: &enabled,
			Source:  "primary-cluster",
		},
		ExternalClusters: []pgv1.ExternalCluster{
			{
				Name: "primary-cluster",
				ConnectionParameters: map[string]string{
					"host":   "primary-rw.primary-ns.svc.cluster.local",
					"port":   "5432",
					"user":   "streaming_replica",
					"dbname": "postgres",
				},
			},
		},
		Bootstrap: pgv1.BootstrapConfiguration{
			PgBaseBackup: &pgv1.BootstrapPgBaseBackup{
				Source:   "primary-cluster",
				Database: "postgres",
				Owner:    "postgres",
			},
		},
	}

	configs, _, err := ExtractCommonServiceConfigs(cs, testServicesNs)
	require.NoError(t, err)
	require.NotEmpty(t, configs, "CSPostgreSQLReplica should produce at least one config entry")

	// Find the config entry for common-service-cnpg service
	found := false
	for _, c := range configs {
		configMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := configMap["name"].(string); ok && name == "common-service-cnpg" {
			// Verify the resources array exists
			resources, ok := configMap["resources"].([]interface{})
			require.True(t, ok, "config should have resources field")
			require.NotEmpty(t, resources, "resources should not be empty")

			// Check the Cluster resource
			for _, res := range resources {
				resMap, ok := res.(map[string]interface{})
				if !ok {
					continue
				}
				if kind, _ := resMap["kind"].(string); kind == "Cluster" {
					if resName, _ := resMap["name"].(string); resName == "common-service-db" {
						found = true
						// Verify the data.spec contains replica config
						data, ok := resMap["data"].(map[string]interface{})
						require.True(t, ok, "Cluster resource should have data field")
						spec, ok := data["spec"].(map[string]interface{})
						require.True(t, ok, "data should have spec field")
						// Check for replica-specific fields
						_, hasReplica := spec["replica"]
						_, hasExternal := spec["externalClusters"]
						_, hasBootstrap := spec["bootstrap"]
						assert.True(t, hasReplica || hasExternal || hasBootstrap,
							"spec should contain replica configuration fields")
						break
					}
				}
			}
			break
		}
	}
	assert.True(t, found, "extracted configs should contain common-service-cnpg with Cluster replica config")
}

// TestExtractPostgreSQLReplicaConfig verifies the extraction of PostgreSQL replica config
func TestExtractPostgreSQLReplicaConfig(t *testing.T) {
	enabled := true
	replicaConfig := &apiv3.CSPostgreSQLReplicaConfig{
		Replica: pgv1.ReplicaClusterConfiguration{
			Enabled: &enabled,
			Source:  "primary-cluster",
		},
		ExternalClusters: []pgv1.ExternalCluster{
			{
				Name: "primary-cluster",
				ConnectionParameters: map[string]string{
					"host": "primary-rw.primary-ns.svc.cluster.local",
				},
			},
		},
		Bootstrap: pgv1.BootstrapConfiguration{
			PgBaseBackup: &pgv1.BootstrapPgBaseBackup{
				Source: "primary-cluster",
			},
		},
	}

	result, err := extractPostgreSQLReplicaConfig(replicaConfig)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify structure: should be a service with resources
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok, "result should be a map")
	assert.Equal(t, "common-service-cnpg", resultMap["name"])

	resources, ok := resultMap["resources"].([]interface{})
	require.True(t, ok, "result should have resources field")
	require.NotEmpty(t, resources, "resources should not be empty")

	// Verify the Cluster resource
	clusterRes, ok := resources[0].(map[string]interface{})
	require.True(t, ok, "first resource should be a map")
	assert.Equal(t, "pg.ibm.com/v1", clusterRes["apiVersion"])
	assert.Equal(t, "Cluster", clusterRes["kind"])
	assert.Equal(t, "common-service-db", clusterRes["name"])

	// Verify data.spec contains replica config
	data, ok := clusterRes["data"].(map[string]interface{})
	require.True(t, ok, "Cluster should have data field")

	spec, ok := data["spec"].(map[string]interface{})
	require.True(t, ok, "data should have spec field")

	// Verify replica configuration fields
	replica, ok := spec["replica"].(map[string]interface{})
	require.True(t, ok, "spec should have replica field")
	assert.Equal(t, true, replica["enabled"])
	assert.Equal(t, "primary-cluster", replica["source"])

	// Verify externalClusters
	_, hasExternal := spec["externalClusters"]
	assert.True(t, hasExternal, "spec should have externalClusters field")

	// Verify bootstrap
	bootstrap, hasBootstrap := spec["bootstrap"].(map[string]interface{})
	require.True(t, hasBootstrap, "spec should have bootstrap field")

	// Verify bootstrap.pg_basebackup is present
	_, hasPgBaseBackup := bootstrap["pg_basebackup"]
	assert.True(t, hasPgBaseBackup, "bootstrap should have pg_basebackup field")

	// Verify bootstrap.initdb is explicitly set to nil (to remove base template's initdb)
	initdb, hasInitdb := bootstrap["initdb"]
	assert.True(t, hasInitdb, "bootstrap should have initdb field set to nil")
	assert.Nil(t, initdb, "bootstrap.initdb should be nil to remove base template's initdb")
}
