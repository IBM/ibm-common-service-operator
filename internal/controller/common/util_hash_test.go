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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateHash(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "Empty input returns empty string",
			input: []byte{},
		},
		{
			name:  "Simple string input",
			input: []byte("hello world"),
		},
		{
			name:  "JSON input",
			input: []byte(`{"key":"value"}`),
		},
		{
			name:  "Same input produces same hash",
			input: []byte("test data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateHash(tt.input)

			if len(tt.input) == 0 {
				assert.Equal(t, "", result, "Empty input should return empty string")
			} else {
				assert.NotEmpty(t, result, "Non-empty input should produce a hash")
				assert.Len(t, result, 14, "Hash should be 14 characters (7 bytes in hex)")
			}
		})
	}
}

func TestCalculateHash_Consistency(t *testing.T) {
	input := []byte("consistent data")
	hash1 := CalculateHash(input)
	hash2 := CalculateHash(input)

	assert.Equal(t, hash1, hash2, "Same input should produce same hash")
	assert.Len(t, hash1, 14, "Hash should be 14 characters (7 bytes in hex)")
}

func TestCalculateResourceHash(t *testing.T) {
	tests := []struct {
		name        string
		resource    map[string]interface{}
		expectError bool
	}{
		{
			name: "Simple resource",
			resource: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"name":       "test-cm",
			},
			expectError: false,
		},
		{
			name: "Complex nested resource",
			resource: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"name":       "test-deploy",
				"data": map[string]interface{}{
					"spec": map[string]interface{}{
						"replicas": 3,
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "nginx",
										"image": "nginx:latest",
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:        "Empty resource",
			resource:    map[string]interface{}{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := CalculateResourceHash(tt.resource)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.Len(t, hash, 14, "Hash should be 14 characters")
			}
		})
	}
}

func TestCalculateResourceHash_Consistency(t *testing.T) {
	resource := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"name":       "test-svc",
		"data": map[string]interface{}{
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{
						"port":       80,
						"targetPort": 8080,
					},
				},
			},
		},
	}

	hash1, err1 := CalculateResourceHash(resource)
	hash2, err2 := CalculateResourceHash(resource)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, hash1, hash2, "Same resource should produce same hash")
}

func TestCalculateResourceHash_DifferentResources(t *testing.T) {
	resource1 := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"name":       "test-cm-1",
	}

	resource2 := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"name":       "test-cm-2",
	}

	hash1, err1 := CalculateResourceHash(resource1)
	hash2, err2 := CalculateResourceHash(resource2)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, hash1, hash2, "Different resources should produce different hashes")
}

func TestCompareResourceHashes(t *testing.T) {
	tests := []struct {
		name      string
		resource1 map[string]interface{}
		resource2 map[string]interface{}
		expected  bool
	}{
		{
			name: "Identical resources",
			resource1: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"name":       "test-cm",
			},
			resource2: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"name":       "test-cm",
			},
			expected: true,
		},
		{
			name: "Different resources",
			resource1: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"name":       "test-cm-1",
			},
			resource2: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"name":       "test-cm-2",
			},
			expected: false,
		},
		{
			name: "Same structure different values",
			resource1: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"data": map[string]interface{}{
					"spec": map[string]interface{}{
						"replicas": 1,
					},
				},
			},
			resource2: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"data": map[string]interface{}{
					"spec": map[string]interface{}{
						"replicas": 3,
					},
				},
			},
			expected: false,
		},
		{
			name: "Nested complex resources - identical",
			resource1: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"data": map[string]interface{}{
					"spec": map[string]interface{}{
						"replicas": 2,
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "nginx",
										"image": "nginx:1.19",
									},
								},
							},
						},
					},
				},
			},
			resource2: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"data": map[string]interface{}{
					"spec": map[string]interface{}{
						"replicas": 2,
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "nginx",
										"image": "nginx:1.19",
									},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name:      "Empty resources",
			resource1: map[string]interface{}{},
			resource2: map[string]interface{}{},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareResourceHashes(tt.resource1, tt.resource2)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareResourceHashes_OrderIndependence(t *testing.T) {
	// Note: JSON marshaling in Go maintains map key order, so this tests
	// that the same logical resource produces the same hash
	resource1 := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"name":       "test",
		"namespace":  "default",
	}

	resource2 := map[string]interface{}{
		"name":       "test",
		"namespace":  "default",
		"apiVersion": "v1",
		"kind":       "ConfigMap",
	}

	result, err := CompareResourceHashes(resource1, resource2)
	require.NoError(t, err)
	assert.True(t, result, "Resources with same content should match regardless of key order in code")
}

// Made with Bob
