//
// Copyright 2020 IBM Corporation
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

package rules

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog"
)

func resourceStringComparison(resourceA, resourceB string) (string, error) {
	quantityA, err := resource.ParseQuantity(resourceA)
	if err != nil {
		return "", err
	}
	quantityB, err := resource.ParseQuantity(resourceB)
	if err != nil {
		return "", err
	}
	if quantityA.Cmp(quantityB) > 0 {
		return resourceA, nil
	}
	return resourceB, nil
}

func ResourceComparison(resourceA, resourceB interface{}) interface{} {

	if reflect.TypeOf(resourceA).Kind() != reflect.TypeOf(resourceB).Kind() {
		return resourceA
	}

	switch resourceA.(type) {
	case string:
		result, err := resourceStringComparison(resourceA.(string), resourceB.(string))
		if err != nil {
			klog.Error(err)
		}
		return result
	case int64:
		if resourceA.(int64) > resourceB.(int64) {
			return resourceA
		}
		return resourceB
	default:
		return resourceA
	}
}
