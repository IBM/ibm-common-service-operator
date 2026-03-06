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
