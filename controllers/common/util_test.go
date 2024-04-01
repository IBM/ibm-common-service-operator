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
)

func TestSanitizeData(t *testing.T) {
	// Test case 1: valueType is "string" and isEmpty is true
	data1 := map[string]interface{}{
		"key1": "value1",
		"key2": "",
		"key3": 123,
	}
	expectedResult1 := map[string]interface{}{
		"key1": "value1",
		"key2": "",
	}
	result1 := SanitizeData(data1, "string", true)
	assert.Equal(t, expectedResult1, result1)

	// Test case 2: valueType is "string" and isEmpty is false
	data2 := map[string]interface{}{
		"key1": "value1",
		"key2": "",
		"key3": 123,
	}
	expectedResult2 := map[string]interface{}{
		"key1": "value1",
	}
	result2 := SanitizeData(data2, "string", false)
	assert.Equal(t, expectedResult2, result2)

	// Test case 3: valueType is "bool"
	data3 := map[string]interface{}{
		"key1": "value1",
		"key2": true,
		"key3": 123,
	}
	expectedResult3 := map[string]interface{}{
		"key2": true,
	}
	result3 := SanitizeData(data3, "bool", false)
	assert.Equal(t, expectedResult3, result3)

	// Test case 4: valueType is not "string" or "bool"
	data4 := map[string]interface{}{
		"key1": "value1",
		"key2": true,
		"key3": 123,
	}
	expectedResult4 := map[string]interface{}{}
	result4 := SanitizeData(data4, "other", false)
	assert.Equal(t, expectedResult4, result4)
}
