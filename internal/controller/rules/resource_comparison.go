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

package rules

import (
	"fmt"
	"reflect"
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog"
)

var (
	profileSize = map[string]int{
		"small":  1,
		"medium": 2,
		"large":  3,
	}
)

// normalizeResourceQuantity converts non-standard resource quantity formats
// to formats that resource.ParseQuantity can parse
func normalizeResourceQuantity(quantity string) string {
	// In Kubernetes, memory units are like "M" or "Mi", not "MB"
	if len(quantity) > 2 {
		if quantity[len(quantity)-2:] == "kB" {
			return quantity[:len(quantity)-2] + "k"
		}
		if quantity[len(quantity)-2:] == "MB" {
			return quantity[:len(quantity)-2] + "M"
		}
		if quantity[len(quantity)-2:] == "GB" {
			return quantity[:len(quantity)-2] + "G"
		}
		if quantity[len(quantity)-2:] == "TB" {
			return quantity[:len(quantity)-2] + "T"
		}
	}
	return quantity
}

func resourceStringComparison(resourceA, resourceB string) (string, string, error) {
	if sizeA, ok := profileSize[resourceA]; ok {
		if sizeB, ok := profileSize[resourceB]; ok {
			if sizeA > sizeB {
				return resourceA, resourceB, nil
			}
			return resourceB, resourceA, nil
		}
		err := fmt.Errorf("failed to compare resources %s and %s", resourceA, resourceB)
		return "", "", err
	}

	// Normalize the resource quantities to handle formats like "96MB" -> "96M"
	normalizedA := normalizeResourceQuantity(resourceA)
	normalizedB := normalizeResourceQuantity(resourceB)

	quantityA, errA := resource.ParseQuantity(normalizedA)
	quantityB, errB := resource.ParseQuantity(normalizedB)

	// If only one failed to parse, prefer the valid one
	if errA != nil && errB == nil {
		// A is invalid, B is valid - prefer B
		klog.Warningf("Resource '%s' is invalid (%v), using valid resource '%s' instead", resourceA, errA, resourceB)
		return resourceB, resourceA, nil
	}
	if errA == nil && errB != nil {
		// A is valid, B is invalid - prefer A
		klog.Warningf("Resource '%s' is invalid (%v), using valid resource '%s' instead", resourceB, errB, resourceA)
		return resourceA, resourceB, nil
	}

	// Both failed to parse
	if errA != nil && errB != nil {
		return "", "", fmt.Errorf("both resources are invalid: %v, %v", errA, errB)
	}

	// Both are valid - compare them
	if quantityA.Cmp(quantityB) > 0 {
		return resourceA, resourceB, nil
	}
	return resourceB, resourceA, nil
}

func ResourceComparison(resourceA, resourceB interface{}) (interface{}, interface{}) {

	klog.V(3).Infof("Kind of A %s", reflect.TypeOf(resourceA).Kind())
	klog.V(3).Infof("Kind of B %s", reflect.TypeOf(resourceB).Kind())

	switch resourceA.(type) {
	case string:
		// Check if resourceB is also a string to avoid panic
		if resourceBStr, ok := resourceB.(string); ok {
			large, small, err := resourceStringComparison(resourceA.(string), resourceBStr)
			if err != nil {
				klog.Warningf("Failed to compare string resources: %v, defaulting to user value", err)
				// Prefer resourceB (user's value) when comparison fails
				return resourceB, resourceA
			}
			return large, small
		}

		// Type mismatch - convert resourceB to string and use k8s resource comparison
		// e.g., int64(2) becomes "2" which equals "2000m" in CPU terms
		resourceBStr := fmt.Sprintf("%v", resourceB)

		// Try to parse both as k8s resource quantities
		normalizedA := normalizeResourceQuantity(resourceA.(string))
		normalizedB := normalizeResourceQuantity(resourceBStr)

		quantityA, errA := resource.ParseQuantity(normalizedA)
		quantityB, errB := resource.ParseQuantity(normalizedB)

		if errA == nil && errB == nil {
			// Both parsed successfully - compare as quantities
			if quantityA.Cmp(quantityB) > 0 {
				return resourceA, resourceB
			}
			return resourceB, resourceA
		}

		// If only one failed to parse, prefer the valid one
		if errA != nil && errB == nil {
			klog.Warningf("Resource '%v' is invalid (%v), using valid resource '%v' instead", resourceA, errA, resourceB)
			return resourceB, resourceA
		}
		if errA == nil && errB != nil {
			klog.Warningf("Resource '%v' is invalid (%v), using valid resource '%v' instead", resourceB, errB, resourceA)
			return resourceA, resourceB
		}

		// Both failed to parse - try string comparison
		large, small, err := resourceStringComparison(resourceA.(string), resourceBStr)
		if err != nil {
			// String comparison also failed - prefer resourceB (user's value)
			klog.Warningf("Failed to compare resources: %v, defaulting to user value", err)
			return resourceB, resourceA
		}
		return large, small
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		strA := fmt.Sprintf("%v", resourceA)
		strB := fmt.Sprintf("%v", resourceB)

		// Try k8s resource quantity comparison first
		normalizedA := normalizeResourceQuantity(strA)
		normalizedB := normalizeResourceQuantity(strB)

		quantityA, errA := resource.ParseQuantity(normalizedA)
		quantityB, errB := resource.ParseQuantity(normalizedB)

		if errA == nil && errB == nil {
			// Both parsed successfully - compare as k8s quantities
			if quantityA.Cmp(quantityB) > 0 {
				return resourceA, resourceB
			}
			return resourceB, resourceA
		}

		// Fallback to numeric comparison
		floatA, _ := strconv.ParseFloat(strA, 64)
		floatB, _ := strconv.ParseFloat(strB, 64)
		if floatA > floatB {
			return resourceA, resourceB
		}
		return resourceB, resourceA
	case bool:
		boolA := resourceA.(bool)
		boolB := resourceB.(bool)
		var boolMap = map[bool]int{
			false: 1,
			true:  0,
		}
		if boolMap[boolA] > boolMap[boolB] {
			return resourceA, resourceB
		}
		return resourceB, resourceA
	default:
		// result won't change for other types
		return resourceA, resourceA
	}
}

func ResourceEqualComparison(resourceA interface{}, resourceB interface{}) bool {

	if resourceA != nil && resourceB != nil {
		klog.V(3).Infof("Kind of A %s", reflect.TypeOf(resourceA).Kind())
		klog.V(3).Infof("Kind of B %s", reflect.TypeOf(resourceB).Kind())

		isEqual := true
		switch resourceA := resourceA.(type) {
		case []interface{}:
			if resourceB, ok := resourceB.([]interface{}); ok {
				if len(resourceA) != len(resourceB) {
					isEqual = false
				} else {
					// TODO: need to find a better way to compare when the order of slice is not fixed
					for index := range resourceA {
						if !ResourceEqualComparison(resourceA[index], resourceB[index]) {
							return false
						}
					}
				}
			}
			return isEqual
		case map[string]interface{}:
			if _, ok := resourceB.(map[string]interface{}); ok { //Check that the changed map value is also a map[string]interface
				resourceARef := resourceA
				resourceBRef := resourceB.(map[string]interface{})
				for newKey := range resourceARef {
					isEqual = ResourceEqualComparison(resourceARef[newKey], resourceBRef[newKey])
					if !isEqual {
						break
					}
				}
			}
			return isEqual
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			strA := fmt.Sprintf("%v", resourceA)
			strB := fmt.Sprintf("%v", resourceB)

			floatA, _ := strconv.ParseFloat(strA, 64)
			floatB, _ := strconv.ParseFloat(strB, 64)
			if floatA == floatB {
				isEqual = true
			} else {
				isEqual = false
			}
			return isEqual
		default:
			if resourceA == resourceB {
				isEqual = true
			} else {
				isEqual = false
			}
			return isEqual
		}
	}
	if resourceA == nil && resourceB == nil {
		return true
	}
	return false
}
