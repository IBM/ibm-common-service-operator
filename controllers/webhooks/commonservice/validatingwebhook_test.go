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
