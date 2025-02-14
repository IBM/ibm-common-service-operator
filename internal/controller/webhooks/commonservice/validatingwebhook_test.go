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

package commonservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestHugePageSettingDenied(t *testing.T) {

	r := &Defaulter{}
	cs := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"hugepages": map[string]interface{}{
					"enable": true,
				},
			},
		},
	}

	// Test case: Valid hugepages sizes and allocations
	cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{})["hugepages-1Gi"] = "2Gi"
	cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{})["hugepages-2Mi"] = "4Mi"
	isDenied, err := r.HugePageSettingDenied(cs)
	assert.False(t, isDenied)
	assert.Nil(t, err)

	// Test case: Invalid hugepages size format
	cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{})["invalid-1Gi"] = "2Gi"
	isDenied, err = r.HugePageSettingDenied(cs)
	assert.True(t, isDenied)
	assert.Contains(t, err.Error(), "invalid hugepages size on prefix")
	// Delete invalid size
	delete(cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{}), "invalid-1Gi")

	// Test case: Invalid hugepages size quantity
	cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{})["hugepages-invalid"] = "invalid-quantity"
	isDenied, err = r.HugePageSettingDenied(cs)
	assert.True(t, isDenied)
	assert.Contains(t, err.Error(), "invalid hugepages size on Quantity")
	// Delete invalid quantity
	delete(cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{}), "hugepages-invalid")

	// Test case: Invalid hugepages allocation format
	cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{})["hugepages-1Gi"] = "2Gi"
	cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{})["hugepages-2Mi"] = "invalid-allocation"
	isDenied, err = r.HugePageSettingDenied(cs)
	assert.True(t, isDenied)
	assert.Contains(t, err.Error(), "invalid hugepages allocation")
	// Delete invalid allocation
	delete(cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{}), "hugepages-2Mi")

	// Test case: No hugepages enabled
	cs.Object["spec"].(map[string]interface{})["hugepages"].(map[string]interface{})["enable"] = false
	isDenied, err = r.HugePageSettingDenied(cs)
	assert.False(t, isDenied)
	assert.Nil(t, err)
}
