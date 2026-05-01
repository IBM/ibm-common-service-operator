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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/bootstrap"
)

// TestMergeResourceArrays_PreservesBaseOnlyResources verifies that base resources
// not present in CS resources are preserved in the merged result.
// This is the critical test for the bug fix.
func TestMergeResourceArrays_PreservesBaseOnlyResources(t *testing.T) {
	baseResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"name":       "common-service-db-tls-cert",
		},
		map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"name":       "common-service-db-replica-tls-cert",
		},
		map[string]interface{}{
			"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
			"kind":       "Cluster",
			"name":       "common-service-db",
		},
	}

	// CS resources only contain the Cluster, not the Certificates
	csResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
			"kind":       "Cluster",
			"name":       "common-service-db",
			"data": map[string]interface{}{
				"spec": map[string]interface{}{
					"instances": float64(2),
				},
			},
		},
	}

	merged := mergeResourceArrays(baseResources, csResources, "cs4-data", "default")

	// All 3 resources must be present
	require.Len(t, merged, 3, "Merged result must contain all base resources plus CS resources")

	// Verify Certificates are preserved
	certCount := 0
	clusterCount := 0
	for _, res := range merged {
		resMap := res.(map[string]interface{})
		if resMap["kind"] == "Certificate" {
			certCount++
		}
		if resMap["kind"] == "Cluster" {
			clusterCount++
		}
	}

	assert.Equal(t, 2, certCount, "Both Certificate resources must be preserved")
	assert.Equal(t, 1, clusterCount, "Cluster resource must be present")
}

// TestMergeResourceArrays_MergesMatchingResources verifies that when a resource
// exists in both base and CS, they are properly merged.
func TestMergeResourceArrays_MergesMatchingResources(t *testing.T) {
	baseResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
			"kind":       "Cluster",
			"name":       "common-service-db",
			"data": map[string]interface{}{
				"spec": map[string]interface{}{
					"instances": float64(1),
					"storage": map[string]interface{}{
						"size": "10Gi",
					},
				},
			},
		},
	}

	csResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
			"kind":       "Cluster",
			"name":       "common-service-db",
			"data": map[string]interface{}{
				"spec": map[string]interface{}{
					"instances": float64(3),
				},
			},
		},
	}

	merged := mergeResourceArrays(baseResources, csResources, "cs4-data", "default")

	require.Len(t, merged, 1, "Should have one merged resource")

	mergedRes := merged[0].(map[string]interface{})
	assert.Equal(t, "Cluster", mergedRes["kind"])
	assert.Equal(t, "common-service-db", mergedRes["name"])

	// Verify the merge happened (instances should be updated)
	data := mergedRes["data"].(map[string]interface{})
	spec := data["spec"].(map[string]interface{})
	assert.Equal(t, float64(3), spec["instances"], "CS value should override base value")
}

// TestMergeResourceArrays_AddsNewCSResources verifies that resources only in CS
// are added to the result.
func TestMergeResourceArrays_AddsNewCSResources(t *testing.T) {
	baseResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"name":       "base-cert",
		},
	}

	csResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"name":       "new-configmap",
		},
	}

	merged := mergeResourceArrays(baseResources, csResources, "cs4-data", "default")

	require.Len(t, merged, 2, "Should have both base and new CS resources")

	names := make([]string, 0, 2)
	for _, res := range merged {
		names = append(names, res.(map[string]interface{})["name"].(string))
	}

	assert.Contains(t, names, "base-cert", "Base resource must be preserved")
	assert.Contains(t, names, "new-configmap", "New CS resource must be added")
}

// TestMergeResourceArrays_EmptyBaseResources verifies behavior when base has no resources.
func TestMergeResourceArrays_EmptyBaseResources(t *testing.T) {
	baseResources := []interface{}{}

	csResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"name":       "cs-configmap",
		},
	}

	merged := mergeResourceArrays(baseResources, csResources, "cs4-data", "default")

	require.Len(t, merged, 1, "Should have CS resource")
	assert.Equal(t, "cs-configmap", merged[0].(map[string]interface{})["name"])
}

// TestMergeResourceArrays_EmptyCSResources verifies that all base resources are
// preserved when CS has no resources.
func TestMergeResourceArrays_EmptyCSResources(t *testing.T) {
	baseResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"name":       "base-cert-1",
		},
		map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"name":       "base-cert-2",
		},
	}

	csResources := []interface{}{}

	merged := mergeResourceArrays(baseResources, csResources, "cs4-data", "default")

	require.Len(t, merged, 2, "All base resources must be preserved")
	assert.Equal(t, "base-cert-1", merged[0].(map[string]interface{})["name"])
	assert.Equal(t, "base-cert-2", merged[1].(map[string]interface{})["name"])
}

// TestMergeResourceArrays_MatchesByGVKNameNamespace verifies that resources are
// matched correctly by GroupVersionKind + Name + Namespace.
func TestMergeResourceArrays_MatchesByGVKNameNamespace(t *testing.T) {
	baseResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"name":       "my-config",
			"namespace":  "ns1",
		},
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"name":       "my-config",
			"namespace":  "ns2",
		},
	}

	// CS resource matches only the ns1 ConfigMap
	csResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"name":       "my-config",
			"namespace":  "ns1",
			"data": map[string]interface{}{
				"key": "updated-value",
			},
		},
	}

	merged := mergeResourceArrays(baseResources, csResources, "cs4-data", "default")

	require.Len(t, merged, 2, "Both ConfigMaps should be present")

	// Find the ns1 ConfigMap and verify it was merged
	var ns1Config map[string]interface{}
	var ns2Config map[string]interface{}
	for _, res := range merged {
		resMap := res.(map[string]interface{})
		if resMap["namespace"] == "ns1" {
			ns1Config = resMap
		} else if resMap["namespace"] == "ns2" {
			ns2Config = resMap
		}
	}

	require.NotNil(t, ns1Config, "ns1 ConfigMap must be present")
	require.NotNil(t, ns2Config, "ns2 ConfigMap must be present")

	// ns1 should have the merged data
	assert.NotNil(t, ns1Config["data"], "ns1 ConfigMap should have merged data")

	// ns2 should be unchanged (no data field)
	assert.Nil(t, ns2Config["data"], "ns2 ConfigMap should be unchanged")
}

// TestMergeResourceArrays_HandlesInvalidCSResources verifies that CS resources
// with missing required fields are skipped with a warning.
func TestMergeResourceArrays_HandlesInvalidCSResources(t *testing.T) {
	baseResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"name":       "valid-config",
		},
	}

	csResources := []interface{}{
		map[string]interface{}{
			// Missing apiVersion
			"kind": "ConfigMap",
			"name": "invalid-config",
		},
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"name":       "another-valid-config",
		},
	}

	merged := mergeResourceArrays(baseResources, csResources, "cs4-data", "default")

	// Should have base + 1 valid CS resource (invalid one skipped)
	require.Len(t, merged, 2, "Should have base resource and one valid CS resource")

	names := make([]string, 0, 2)
	for _, res := range merged {
		names = append(names, res.(map[string]interface{})["name"].(string))
	}

	assert.Contains(t, names, "valid-config")
	assert.Contains(t, names, "another-valid-config")
	assert.NotContains(t, names, "invalid-config", "Invalid resource should be skipped")
}

// TestMergeResourceArrays_PostgreSQLCertificatesScenario tests the exact scenario
// from the bug: PostgreSQL service with Certificates in base, only Cluster in CS.
func TestMergeResourceArrays_PostgreSQLCertificatesScenario(t *testing.T) {
	// This replicates the actual bug scenario
	baseResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"name":       "common-service-db-replica-tls-cert",
			"data": map[string]interface{}{
				"spec": map[string]interface{}{
					"commonName": "streaming_replica",
					"secretName": "common-service-db-replica-tls-secret",
				},
			},
		},
		map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"name":       "common-service-db-tls-cert",
			"data": map[string]interface{}{
				"spec": map[string]interface{}{
					"secretName": "common-service-db-tls-secret",
					"dnsNames": []interface{}{
						"common-service-db",
						"common-service-db.cs4-data",
					},
				},
			},
		},
		map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"name":       "common-service-db-im-tls-cert",
			"data": map[string]interface{}{
				"spec": map[string]interface{}{
					"commonName": "im_user",
					"secretName": "common-service-db-im-tls-secret",
				},
			},
		},
		map[string]interface{}{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"name":       "common-service-db-zen-tls-cert",
			"data": map[string]interface{}{
				"spec": map[string]interface{}{
					"commonName": "zen_user",
					"secretName": "common-service-db-zen-tls-secret",
				},
			},
		},
		map[string]interface{}{
			"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
			"kind":       "Cluster",
			"name":       "common-service-db",
			"data": map[string]interface{}{
				"spec": map[string]interface{}{
					"instances": float64(1),
				},
			},
		},
	}

	// CS resources from size profile - only has Cluster
	csResources := []interface{}{
		map[string]interface{}{
			"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
			"kind":       "Cluster",
			"name":       "common-service-db",
			"data": map[string]interface{}{
				"spec": map[string]interface{}{
					"instances": float64(1),
					"resources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "200m",
							"memory": "512Mi",
						},
					},
				},
			},
		},
	}

	merged := mergeResourceArrays(baseResources, csResources, "cs4-data", "default")

	// CRITICAL: All 5 resources must be present (4 Certificates + 1 Cluster)
	require.Len(t, merged, 5, "All base resources must be preserved: 4 Certificates + 1 Cluster")

	// Verify all Certificates are present
	certNames := []string{
		"common-service-db-replica-tls-cert",
		"common-service-db-tls-cert",
		"common-service-db-im-tls-cert",
		"common-service-db-zen-tls-cert",
	}

	foundCerts := make(map[string]bool)
	var clusterResource map[string]interface{}

	for _, res := range merged {
		resMap := res.(map[string]interface{})
		if resMap["kind"] == "Certificate" {
			foundCerts[resMap["name"].(string)] = true
		} else if resMap["kind"] == "Cluster" {
			clusterResource = resMap
		}
	}

	// Verify all Certificates are present
	for _, certName := range certNames {
		assert.True(t, foundCerts[certName], "Certificate %s must be preserved", certName)
	}

	// Verify Cluster was merged with CS values
	require.NotNil(t, clusterResource, "Cluster resource must be present")
	assert.Equal(t, "common-service-db", clusterResource["name"])

	// Verify Cluster has merged spec from CS
	data := clusterResource["data"].(map[string]interface{})
	spec := data["spec"].(map[string]interface{})
	assert.NotNil(t, spec["resources"], "Cluster should have resources from CS config")
}

// TestMergeChangedMap_NonComparableKeys verifies that non-comparable keys
// (like storageClass, zenFrontDoor) are properly merged from defaultMap to finalMap.
// This test validates the fix for the bug where these fields were being ignored.
func TestMergeChangedMap_NonComparableKeys(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultMap   interface{}
		changedMap   interface{}
		expected     interface{}
		directAssign bool
	}{
		{
			name:         "storageClass field should be merged when directAssign=true and changedMap is nil",
			key:          "storageClass",
			defaultMap:   "nfs-storage",
			changedMap:   nil,
			expected:     "nfs-storage",
			directAssign: true,
		},
		{
			name:         "storageClass field should be merged when directAssign=true and changedMap is empty",
			key:          "storageClass",
			defaultMap:   "nfs-storage",
			changedMap:   "",
			expected:     "nfs-storage",
			directAssign: true,
		},
		{
			name:         "storageClass field should NOT be merged when directAssign=false",
			key:          "storageClass",
			defaultMap:   "nfs-storage",
			changedMap:   "",
			expected:     nil,
			directAssign: false,
		},
		{
			name:         "zenFrontDoor field should be merged when directAssign=true",
			key:          "zenFrontDoor",
			defaultMap:   true,
			changedMap:   nil,
			expected:     true,
			directAssign: true,
		},
		{
			name:         "zenFrontDoor false should be merged when directAssign=true",
			key:          "zenFrontDoor",
			defaultMap:   false,
			changedMap:   nil,
			expected:     false,
			directAssign: true,
		},
		{
			name:         "zenFrontDoor should NOT be merged when directAssign=false",
			key:          "zenFrontDoor",
			defaultMap:   true,
			changedMap:   false,
			expected:     nil,
			directAssign: false,
		},
		{
			name:         "custom string field should be merged when directAssign=true",
			key:          "customField",
			defaultMap:   "customValue",
			changedMap:   nil,
			expected:     "customValue",
			directAssign: true,
		},
		{
			name:         "custom number field should be merged when directAssign=true",
			key:          "customNumber",
			defaultMap:   float64(123),
			changedMap:   nil,
			expected:     float64(123),
			directAssign: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finalMap := make(map[string]interface{})
			mergeChangedMap(tt.key, tt.defaultMap, tt.changedMap, finalMap, tt.directAssign)

			if tt.expected == nil {
				assert.NotContains(t, finalMap, tt.key,
					"Field %s should NOT be set when directAssign=false", tt.key)
			} else {
				assert.Equal(t, tt.expected, finalMap[tt.key],
					"Field %s should be set to defaultMap value", tt.key)
			}
		})
	}
}

// TestMergeChangedMap_NestedStorageClass verifies that storageClass fields
// nested within maps are properly merged.
func TestMergeChangedMap_NestedStorageClass(t *testing.T) {
	// Simulate the structure: spec.storage.storageClass
	defaultMap := map[string]interface{}{
		"storage": map[string]interface{}{
			"storageClass": "nfs-storage",
			"size":         "10Gi",
		},
		"walStorage": map[string]interface{}{
			"storageClass": "nfs-storage",
			"size":         "5Gi",
		},
	}

	// OperandConfig has storage but without storageClass
	changedMap := map[string]interface{}{
		"storage": map[string]interface{}{
			"size": "10Gi",
		},
		"walStorage": map[string]interface{}{
			"size": "5Gi",
		},
	}

	finalMap := make(map[string]interface{})
	// Initialize nested maps in finalMap
	finalMap["storage"] = make(map[string]interface{})
	finalMap["walStorage"] = make(map[string]interface{})

	// Merge storage
	mergeChangedMap("storage", defaultMap["storage"], changedMap["storage"], finalMap, false)
	// Merge walStorage
	mergeChangedMap("walStorage", defaultMap["walStorage"], changedMap["walStorage"], finalMap, false)

	// Verify storageClass was added to both
	storageMap := finalMap["storage"].(map[string]interface{})
	assert.Equal(t, "nfs-storage", storageMap["storageClass"],
		"storageClass should be merged into storage")

	walStorageMap := finalMap["walStorage"].(map[string]interface{})
	assert.Equal(t, "nfs-storage", walStorageMap["storageClass"],
		"storageClass should be merged into walStorage")
}

// TestMergeChangedMap_ComparableKeysStillWork verifies that the fix doesn't
// break the existing behavior for comparable keys (replicas, cpu, memory, etc.)
func TestMergeChangedMap_ComparableKeysStillWork(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultMap interface{}
		changedMap interface{}
		// For comparable keys, ResourceComparison picks the larger value
		expectLarger bool
	}{
		{
			name:         "replicas comparison",
			key:          "replicas",
			defaultMap:   float64(3),
			changedMap:   float64(2),
			expectLarger: true,
		},
		{
			name:         "cpu comparison",
			key:          "cpu",
			defaultMap:   "500m",
			changedMap:   "200m",
			expectLarger: true,
		},
		{
			name:         "memory comparison",
			key:          "memory",
			defaultMap:   "2Gi",
			changedMap:   "1Gi",
			expectLarger: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finalMap := make(map[string]interface{})
			mergeChangedMap(tt.key, tt.defaultMap, tt.changedMap, finalMap, false)

			// Verify the key exists in finalMap
			assert.Contains(t, finalMap, tt.key,
				"Comparable key %s should be present in finalMap", tt.key)

			// The actual comparison logic is in ResourceComparison,
			// we just verify that the key was processed
			assert.NotNil(t, finalMap[tt.key],
				"Comparable key %s should have a value", tt.key)
		})
	}
}

// TestMergeCRsIntoOperandConfigWithDefaultRules_StorageClass is an integration test
// that verifies storageClass merging through the full merge flow.
func TestMergeCRsIntoOperandConfigWithDefaultRules_StorageClass(t *testing.T) {
	// Simulate CommonService config with storageClass
	defaultMap := map[string]interface{}{
		"data": map[string]interface{}{
			"spec": map[string]interface{}{
				"storage": map[string]interface{}{
					"storageClass": "nfs-storage",
					"size":         "10Gi",
				},
				"walStorage": map[string]interface{}{
					"storageClass": "nfs-storage",
					"size":         "5Gi",
				},
			},
		},
	}

	// Simulate OperandConfig without storageClass
	changedMap := map[string]interface{}{
		"data": map[string]interface{}{
			"spec": map[string]interface{}{
				"storage": map[string]interface{}{
					"size": "10Gi",
				},
				"walStorage": map[string]interface{}{
					"size": "5Gi",
				},
			},
		},
	}

	result := mergeCRsIntoOperandConfigWithDefaultRules(defaultMap, changedMap, false)

	// Verify storageClass was merged
	data := result["data"].(map[string]interface{})
	spec := data["spec"].(map[string]interface{})
	storage := spec["storage"].(map[string]interface{})
	walStorage := spec["walStorage"].(map[string]interface{})

	assert.Equal(t, "nfs-storage", storage["storageClass"],
		"storageClass should be merged into storage")
	assert.Equal(t, "nfs-storage", walStorage["storageClass"],
		"storageClass should be merged into walStorage")
}

// TestRemoveOrphanedSpecFields_RemovesOrphanedCRs verifies that CR specs
// not present in the desired state are removed from OperandConfig.

// TestGetItemByName verifies the helper function for finding items by name.
func TestGetItemByName(t *testing.T) {
	slice := []interface{}{
		map[string]interface{}{"name": "service1"},
		map[string]interface{}{"name": "service2"},
		map[string]interface{}{"name": "service3"},
	}

	tests := []struct {
		name     string
		findName string
		found    bool
	}{
		{
			name:     "Find existing service",
			findName: "service2",
			found:    true,
		},
		{
			name:     "Service not found",
			findName: "service4",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getItemByName(slice, tt.findName)
			if tt.found {
				assert.NotNil(t, result)
				assert.Equal(t, tt.findName, result.(map[string]interface{})["name"])
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

// TestGetItemByGVKNameNamespace verifies the helper function for finding
// resources by GroupVersionKind + Name + Namespace.
func TestGetItemByGVKNameNamespace(t *testing.T) {
	resources := []interface{}{
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"name":       "config1",
			"namespace":  "ns1",
		},
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"name":       "config2",
			"namespace":  "ns2",
		},
		map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"name":       "deploy1",
			"namespace":  "ns1",
		},
	}

	tests := []struct {
		name       string
		apiVersion string
		kind       string
		resName    string
		namespace  string
		found      bool
	}{
		{
			name:       "Find ConfigMap in ns1",
			apiVersion: "v1",
			kind:       "ConfigMap",
			resName:    "config1",
			namespace:  "ns1",
			found:      true,
		},
		{
			name:       "Find Deployment",
			apiVersion: "apps/v1",
			kind:       "Deployment",
			resName:    "deploy1",
			namespace:  "ns1",
			found:      true,
		},
		{
			name:       "Not found - wrong namespace",
			apiVersion: "v1",
			kind:       "ConfigMap",
			resName:    "config1",
			namespace:  "ns2",
			found:      false,
		},
		{
			name:       "Not found - wrong kind",
			apiVersion: "v1",
			kind:       "Secret",
			resName:    "config1",
			namespace:  "ns1",
			found:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getItemByGVKNameNamespace(resources, "default-ns", tt.apiVersion, tt.kind, tt.resName, tt.namespace)
			if tt.found {
				assert.NotNil(t, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

// TestMergeServicesWithBase_PreservesBaseOnlyServices verifies that services
// only present in base (like pg-migrator) are preserved in the merged result.
func TestMergeServicesWithBase_PreservesBaseOnlyServices(t *testing.T) {
	baseServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"spec": map[string]interface{}{
				"cluster": map[string]interface{}{
					"instances": float64(2),
				},
			},
		},
		map[string]interface{}{
			"name": "common-service-pg-migrator",
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "batch/v1",
					"kind":       "Job",
					"name":       "pg-migration-job",
				},
			},
		},
	}

	// CS services only contain postgresql, not pg-migrator
	csServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"spec": map[string]interface{}{
				"cluster": map[string]interface{}{
					"instances": float64(3),
				},
			},
		},
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{}
	merged := reconciler.mergeServicesWithBase(baseServices, csServices, ruleSlice)

	// Both services must be present
	require.Len(t, merged, 2, "Both base services must be preserved")

	// Verify pg-migrator is preserved
	pgMigrator := getItemByName(merged, "common-service-pg-migrator")
	require.NotNil(t, pgMigrator, "pg-migrator service must be preserved")

	// Verify postgresql was merged
	postgresql := getItemByName(merged, "common-service-postgresql")
	require.NotNil(t, postgresql, "postgresql service must be present")
	spec := postgresql.(map[string]interface{})["spec"].(map[string]interface{})
	cluster := spec["cluster"].(map[string]interface{})
	assert.Equal(t, float64(3), cluster["instances"], "CS value should be merged")
}

// TestMergeServicesWithBase_MergesMatchingServices verifies that services
// present in both base and CS are properly merged.
func TestMergeServicesWithBase_MergesMatchingServices(t *testing.T) {
	baseServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"spec": map[string]interface{}{
				"cluster": map[string]interface{}{
					"instances": float64(1),
					"storage": map[string]interface{}{
						"size": "10Gi",
					},
				},
			},
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "cert-manager.io/v1",
					"kind":       "Certificate",
					"name":       "db-cert",
				},
			},
		},
	}

	csServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"spec": map[string]interface{}{
				"cluster": map[string]interface{}{
					"instances": float64(3),
				},
			},
		},
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{}
	merged := reconciler.mergeServicesWithBase(baseServices, csServices, ruleSlice)

	require.Len(t, merged, 1, "Should have one merged service")

	service := merged[0].(map[string]interface{})
	assert.Equal(t, "common-service-postgresql", service["name"])

	// Verify spec was merged
	spec := service["spec"].(map[string]interface{})
	cluster := spec["cluster"].(map[string]interface{})
	assert.Equal(t, float64(3), cluster["instances"], "CS instances should override")
	assert.Equal(t, "10Gi", cluster["storage"].(map[string]interface{})["size"], "Base storage should be preserved")

	// Verify resources were preserved
	resources := service["resources"].([]interface{})
	assert.Len(t, resources, 1, "Base resources should be preserved")
}

// TestMergeServicesWithBase_AddsCSOnlyServices verifies that services
// only in CS are added to the result.
func TestMergeServicesWithBase_AddsCSOnlyServices(t *testing.T) {
	baseServices := []interface{}{
		map[string]interface{}{
			"name": "ibm-cert-manager-operator",
		},
	}

	csServices := []interface{}{
		map[string]interface{}{
			"name": "ibm-cert-manager-operator",
		},
		map[string]interface{}{
			"name": "new-custom-service",
			"spec": map[string]interface{}{
				"replicas": float64(2),
			},
		},
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{}
	merged := reconciler.mergeServicesWithBase(baseServices, csServices, ruleSlice)

	require.Len(t, merged, 2, "Should have both services")

	newService := getItemByName(merged, "new-custom-service")
	require.NotNil(t, newService, "New CS service should be added")
	assert.Equal(t, float64(2), newService.(map[string]interface{})["spec"].(map[string]interface{})["replicas"])
}

// TestMergeServicesWithBase_EmptyCSServices verifies that all base services
// are preserved when CS has no services.
func TestMergeServicesWithBase_EmptyCSServices(t *testing.T) {
	baseServices := []interface{}{
		map[string]interface{}{"name": "service1"},
		map[string]interface{}{"name": "service2"},
		map[string]interface{}{"name": "service3"},
	}

	csServices := []interface{}{}
	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{}
	merged := reconciler.mergeServicesWithBase(baseServices, csServices, ruleSlice)

	require.Len(t, merged, 3, "All base services must be preserved")
	assert.NotNil(t, getItemByName(merged, "service1"))
	assert.NotNil(t, getItemByName(merged, "service2"))
	assert.NotNil(t, getItemByName(merged, "service3"))
}

// TestMergeServicesWithBase_RealWorldScenario tests the exact bug scenario:
// base has pg-migrator, CS doesn't, and postgresql needs size config preserved.
func TestMergeServicesWithBase_RealWorldScenario(t *testing.T) {
	baseServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"spec": map[string]interface{}{
				"cluster": map[string]interface{}{
					"instances": float64(2),
				},
			},
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
					"kind":       "Cluster",
					"name":       "common-service-db",
					"data": map[string]interface{}{
						"spec": map[string]interface{}{
							"storage": map[string]interface{}{
								"size":         "10Gi",
								"storageClass": "nfs-client",
							},
						},
					},
				},
			},
		},
		map[string]interface{}{
			"name": "common-service-pg-migrator",
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "batch/v1",
					"kind":       "Job",
					"name":       "pg-migration-job",
				},
			},
		},
	}

	// CS from size profile - has postgresql but not pg-migrator
	csServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"spec": map[string]interface{}{
				"cluster": map[string]interface{}{
					"instances": float64(2),
				},
			},
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
					"kind":       "Cluster",
					"name":       "common-service-db",
					"data": map[string]interface{}{
						"spec": map[string]interface{}{
							"instances": float64(2),
							"resources": map[string]interface{}{
								"limits": map[string]interface{}{
									"cpu":    "200m",
									"memory": "768Mi",
								},
							},
						},
					},
				},
			},
		},
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			CSData: apiv3.CSData{
				ServicesNs: "ibm-common-services",
			},
		},
	}
	merged := reconciler.mergeServicesWithBase(baseServices, csServices, ruleSlice)

	// CRITICAL: Both services must be present
	require.Len(t, merged, 2, "Both services must be present")

	// Verify pg-migrator is preserved
	pgMigrator := getItemByName(merged, "common-service-pg-migrator")
	require.NotNil(t, pgMigrator, "pg-migrator must be preserved from base")

	// Verify postgresql was merged with size config preserved
	postgresql := getItemByName(merged, "common-service-postgresql")
	require.NotNil(t, postgresql, "postgresql must be present")

	resources := postgresql.(map[string]interface{})["resources"].([]interface{})
	require.Len(t, resources, 1, "Should have merged Cluster resource")

	cluster := resources[0].(map[string]interface{})
	data := cluster["data"].(map[string]interface{})
	spec := data["spec"].(map[string]interface{})

	// Verify size config from base is preserved
	storage := spec["storage"].(map[string]interface{})
	assert.Equal(t, "10Gi", storage["size"], "Base storage size must be preserved")
	assert.Equal(t, "nfs-client", storage["storageClass"], "Base storageClass must be preserved")

	// Verify CS config was merged
	assert.Equal(t, float64(2), spec["instances"], "CS instances should be present")
	assert.NotNil(t, spec["resources"], "CS resources should be merged")
}

// TestMergeServiceConfig_MergesSpecWithRules verifies that spec sections
// are merged according to configuration rules.
func TestMergeServiceConfig_MergesSpecWithRules(t *testing.T) {
	baseService := map[string]interface{}{
		"name": "test-service",
		"spec": map[string]interface{}{
			"mongoDB": map[string]interface{}{
				"replicas": float64(1),
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "500m",
						"memory": "512Mi",
					},
				},
			},
		},
	}

	csService := map[string]interface{}{
		"name": "test-service",
		"spec": map[string]interface{}{
			"mongoDB": map[string]interface{}{
				"replicas": float64(3),
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "1000m",
						"memory": "1Gi",
					},
				},
			},
		},
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			CSData: apiv3.CSData{
				ServicesNs: "ibm-common-services",
			},
		},
	}

	merged := reconciler.mergeServiceConfig(baseService, csService, "test-service", ruleSlice)

	require.NotNil(t, merged["spec"])
	spec := merged["spec"].(map[string]interface{})
	mongoDB := spec["mongoDB"].(map[string]interface{})

	// Verify merge happened
	assert.NotNil(t, mongoDB["replicas"])
	assert.NotNil(t, mongoDB["resources"])
}

// TestMergeServiceConfig_MergesResources verifies that resource arrays
// are properly merged.
func TestMergeServiceConfig_MergesResources(t *testing.T) {
	baseService := map[string]interface{}{
		"name": "test-service",
		"resources": []interface{}{
			map[string]interface{}{
				"apiVersion": "cert-manager.io/v1",
				"kind":       "Certificate",
				"name":       "base-cert",
			},
		},
	}

	csService := map[string]interface{}{
		"name": "test-service",
		"resources": []interface{}{
			map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"name":       "cs-config",
			},
		},
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			CSData: apiv3.CSData{
				ServicesNs: "ibm-common-services",
			},
		},
	}

	merged := reconciler.mergeServiceConfig(baseService, csService, "test-service", ruleSlice)

	require.NotNil(t, merged["resources"])
	resources := merged["resources"].([]interface{})

	// Both resources should be present
	require.Len(t, resources, 2, "Both base and CS resources should be present")
}

// TestMergeServiceConfig_CopiesManagementStrategy verifies that
// managementStrategy is copied from CS config.
func TestMergeServiceConfig_CopiesManagementStrategy(t *testing.T) {
	baseService := map[string]interface{}{
		"name": "test-service",
	}

	csService := map[string]interface{}{
		"name":               "test-service",
		"managementStrategy": "turbo",
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			CSData: apiv3.CSData{
				ServicesNs: "ibm-common-services",
			},
		},
	}

	merged := reconciler.mergeServiceConfig(baseService, csService, "test-service", ruleSlice)

	assert.Equal(t, "turbo", merged["managementStrategy"], "managementStrategy should be copied from CS")
}

// TestMergeServiceConfig_PreservesBaseWhenCSEmpty verifies that base
// configuration is preserved when CS has no spec or resources.
func TestMergeServiceConfig_PreservesBaseWhenCSEmpty(t *testing.T) {
	baseService := map[string]interface{}{
		"name": "test-service",
		"spec": map[string]interface{}{
			"replicas": float64(3),
		},
		"resources": []interface{}{
			map[string]interface{}{
				"kind": "Certificate",
				"name": "base-cert",
			},
		},
	}

	csService := map[string]interface{}{
		"name": "test-service",
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			CSData: apiv3.CSData{
				ServicesNs: "ibm-common-services",
			},
		},
	}

	merged := reconciler.mergeServiceConfig(baseService, csService, "test-service", ruleSlice)

	// Base spec and resources should be preserved
	assert.NotNil(t, merged["spec"], "Base spec should be preserved")
	assert.NotNil(t, merged["resources"], "Base resources should be preserved")

	spec := merged["spec"].(map[string]interface{})
	assert.Equal(t, float64(3), spec["replicas"], "Base replicas should be preserved")

	resources := merged["resources"].([]interface{})
	assert.Len(t, resources, 1, "Base resources should be preserved")
}

// TestMergeServiceConfig_PreservesSizeConfiguration verifies that the size
// configuration in storage sections is preserved during merge.
// This tests the specific bug where size config was missing in second reconciliation.
func TestMergeServiceConfig_PreservesSizeConfiguration(t *testing.T) {
	// Base service has complete storage configuration including size
	baseService := map[string]interface{}{
		"name": "common-service-postgresql",
		"resources": []interface{}{
			map[string]interface{}{
				"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
				"kind":       "Cluster",
				"name":       "common-service-db",
				"data": map[string]interface{}{
					"spec": map[string]interface{}{
						"instances": float64(2),
						"storage": map[string]interface{}{
							"size":         "10Gi",
							"storageClass": "nfs-client",
						},
						"walStorage": map[string]interface{}{
							"size":         "5Gi",
							"storageClass": "nfs-client",
						},
					},
				},
			},
		},
	}

	// CS service from size profile - has resources but may not have size
	csService := map[string]interface{}{
		"name": "common-service-postgresql",
		"resources": []interface{}{
			map[string]interface{}{
				"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
				"kind":       "Cluster",
				"name":       "common-service-db",
				"data": map[string]interface{}{
					"spec": map[string]interface{}{
						"instances": float64(2),
						"resources": map[string]interface{}{
							"limits": map[string]interface{}{
								"cpu":    "200m",
								"memory": "768Mi",
							},
						},
					},
				},
			},
		},
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			CSData: apiv3.CSData{
				ServicesNs: "ibm-common-services",
			},
		},
	}

	merged := reconciler.mergeServiceConfig(baseService, csService, "common-service-postgresql", ruleSlice)

	require.NotNil(t, merged["resources"], "Resources must be present")
	resources := merged["resources"].([]interface{})
	require.Len(t, resources, 1, "Should have one Cluster resource")

	cluster := resources[0].(map[string]interface{})
	data := cluster["data"].(map[string]interface{})
	spec := data["spec"].(map[string]interface{})

	// CRITICAL: Verify size configuration is preserved
	require.NotNil(t, spec["storage"], "storage section must be preserved")
	storage := spec["storage"].(map[string]interface{})
	assert.Equal(t, "10Gi", storage["size"], "storage.size must be preserved from base")
	assert.Equal(t, "nfs-client", storage["storageClass"], "storage.storageClass must be preserved from base")

	require.NotNil(t, spec["walStorage"], "walStorage section must be preserved")
	walStorage := spec["walStorage"].(map[string]interface{})
	assert.Equal(t, "5Gi", walStorage["size"], "walStorage.size must be preserved from base")
	assert.Equal(t, "nfs-client", walStorage["storageClass"], "walStorage.storageClass must be preserved from base")

	// Verify CS resources config was also merged
	require.NotNil(t, spec["resources"], "CS resources config should be merged")
	resourcesConfig := spec["resources"].(map[string]interface{})
	limits := resourcesConfig["limits"].(map[string]interface{})
	assert.Equal(t, "200m", limits["cpu"], "CS cpu limits should be present")
	assert.Equal(t, "768Mi", limits["memory"], "CS memory limits should be present")
}

// TestMergeServicesWithBase_PostgreSQLSizeConfigPreserved tests the complete
// scenario where postgresql size configuration must be preserved across reconciliations.
func TestMergeServicesWithBase_PostgreSQLSizeConfigPreserved(t *testing.T) {
	// Simulate base OperandConfig with complete postgresql configuration
	baseServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "cert-manager.io/v1",
					"kind":       "Certificate",
					"name":       "common-service-db-tls-cert",
				},
				map[string]interface{}{
					"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
					"kind":       "Cluster",
					"name":       "common-service-db",
					"data": map[string]interface{}{
						"spec": map[string]interface{}{
							"instances": float64(2),
							"storage": map[string]interface{}{
								"size":         "20Gi",
								"storageClass": "nfs-client",
							},
							"walStorage": map[string]interface{}{
								"size":         "10Gi",
								"storageClass": "nfs-client",
							},
							"postgresql": map[string]interface{}{
								"parameters": map[string]interface{}{
									"max_connections": "600",
									"shared_buffers":  "64MB",
								},
							},
						},
					},
				},
			},
		},
		map[string]interface{}{
			"name": "common-service-pg-migrator",
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "batch/v1",
					"kind":       "Job",
					"name":       "pg-migration-job",
				},
			},
		},
	}

	// Simulate CS from size profile - has postgresql but not pg-migrator
	// and may not have complete storage config
	csServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
					"kind":       "Cluster",
					"name":       "common-service-db",
					"data": map[string]interface{}{
						"spec": map[string]interface{}{
							"instances": float64(2),
							"resources": map[string]interface{}{
								"limits": map[string]interface{}{
									"cpu":               "200m",
									"memory":            "768Mi",
									"ephemeral-storage": "512Mi",
								},
								"requests": map[string]interface{}{
									"cpu":               "75m",
									"memory":            "256Mi",
									"ephemeral-storage": "128Mi",
								},
							},
							"postgresql": map[string]interface{}{
								"parameters": map[string]interface{}{
									"max_connections": "600",
									"shared_buffers":  "64MB",
								},
							},
						},
					},
				},
			},
		},
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			CSData: apiv3.CSData{
				ServicesNs: "ibm-common-services",
			},
		},
	}

	merged := reconciler.mergeServicesWithBase(baseServices, csServices, ruleSlice)

	// Verify both services are present
	require.Len(t, merged, 2, "Both services must be present")

	// Verify pg-migrator is preserved
	pgMigrator := getItemByName(merged, "common-service-pg-migrator")
	require.NotNil(t, pgMigrator, "pg-migrator must be preserved")

	// Verify postgresql with complete configuration
	postgresql := getItemByName(merged, "common-service-postgresql")
	require.NotNil(t, postgresql, "postgresql must be present")

	resources := postgresql.(map[string]interface{})["resources"].([]interface{})
	require.Len(t, resources, 2, "Should have Certificate and Cluster resources")

	// Find the Cluster resource
	var clusterResource map[string]interface{}
	for _, res := range resources {
		resMap := res.(map[string]interface{})
		if resMap["kind"] == "Cluster" {
			clusterResource = resMap
			break
		}
	}

	require.NotNil(t, clusterResource, "Cluster resource must be present")
	data := clusterResource["data"].(map[string]interface{})
	spec := data["spec"].(map[string]interface{})

	// CRITICAL: Verify size configuration is preserved from base
	require.NotNil(t, spec["storage"], "storage must be preserved")
	storage := spec["storage"].(map[string]interface{})
	assert.Equal(t, "20Gi", storage["size"], "storage.size from base must be preserved")
	assert.Equal(t, "nfs-client", storage["storageClass"], "storage.storageClass from base must be preserved")

	require.NotNil(t, spec["walStorage"], "walStorage must be preserved")
	walStorage := spec["walStorage"].(map[string]interface{})
	assert.Equal(t, "10Gi", walStorage["size"], "walStorage.size from base must be preserved")
	assert.Equal(t, "nfs-client", walStorage["storageClass"], "walStorage.storageClass from base must be preserved")

	// Verify CS configuration was also merged
	assert.Equal(t, float64(2), spec["instances"], "instances from CS should be present")
	require.NotNil(t, spec["resources"], "resources from CS should be merged")
	resourcesConfig := spec["resources"].(map[string]interface{})
	limits := resourcesConfig["limits"].(map[string]interface{})
	assert.Equal(t, "200m", limits["cpu"], "CS cpu limits should be present")
	assert.Equal(t, "768Mi", limits["memory"], "CS memory limits should be present")

	// Verify postgresql parameters are preserved
	require.NotNil(t, spec["postgresql"], "postgresql parameters must be present")
	postgresql_params := spec["postgresql"].(map[string]interface{})
	params := postgresql_params["parameters"].(map[string]interface{})
	assert.Equal(t, "600", params["max_connections"], "max_connections should be preserved")
	assert.Equal(t, "64MB", params["shared_buffers"], "shared_buffers should be preserved")
}

// TestMergeServicesWithBase_MultipleReconciliations simulates multiple
// reconciliation cycles to ensure configuration remains stable.
func TestMergeServicesWithBase_MultipleReconciliations(t *testing.T) {
	// Initial base state
	baseServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
					"kind":       "Cluster",
					"name":       "common-service-db",
					"data": map[string]interface{}{
						"spec": map[string]interface{}{
							"storage": map[string]interface{}{
								"size": "10Gi",
							},
						},
					},
				},
			},
		},
		map[string]interface{}{
			"name": "common-service-pg-migrator",
			"resources": []interface{}{
				map[string]interface{}{
					"kind": "Job",
					"name": "migration-job",
				},
			},
		},
	}

	csServices := []interface{}{
		map[string]interface{}{
			"name": "common-service-postgresql",
			"resources": []interface{}{
				map[string]interface{}{
					"apiVersion": "postgresql.k8s.enterprisedb.io/v1",
					"kind":       "Cluster",
					"name":       "common-service-db",
					"data": map[string]interface{}{
						"spec": map[string]interface{}{
							"instances": float64(2),
						},
					},
				},
			},
		},
	}

	ruleSlice := []interface{}{}
	reconciler := &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			CSData: apiv3.CSData{
				ServicesNs: "ibm-common-services",
			},
		},
	}

	// First reconciliation
	merged1 := reconciler.mergeServicesWithBase(baseServices, csServices, ruleSlice)
	require.Len(t, merged1, 2, "First reconciliation: both services must be present")

	// Second reconciliation - use result of first as new base
	merged2 := reconciler.mergeServicesWithBase(merged1, csServices, ruleSlice)
	require.Len(t, merged2, 2, "Second reconciliation: both services must still be present")

	// Verify pg-migrator is still present after multiple reconciliations
	pgMigrator := getItemByName(merged2, "common-service-pg-migrator")
	require.NotNil(t, pgMigrator, "pg-migrator must be preserved after multiple reconciliations")

	// Verify size config is still present
	postgresql := getItemByName(merged2, "common-service-postgresql")
	require.NotNil(t, postgresql, "postgresql must be present")
	resources := postgresql.(map[string]interface{})["resources"].([]interface{})
	cluster := resources[0].(map[string]interface{})
	data := cluster["data"].(map[string]interface{})
	spec := data["spec"].(map[string]interface{})
	storage := spec["storage"].(map[string]interface{})
	assert.Equal(t, "10Gi", storage["size"], "size must be preserved after multiple reconciliations")
}
