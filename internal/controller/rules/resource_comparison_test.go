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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resource Comparison", func() {

	Context("Compare CPU", func() {
		It("Should 1 be larger than 10m", func() {
			A := "1"
			B := "10m"
			expectedResult := "1"

			result, _, err := resourceStringComparison(A, B)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).Should(Equal(expectedResult))
		})
		It("Should 100m be larger than 10m", func() {
			A := "100m"
			B := "10m"
			expectedResult := "100m"

			result, _, err := resourceStringComparison(A, B)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).Should(Equal(expectedResult))
		})
	})

	Context("Compare Memory", func() {
		It("Should 1Gi be larger than 10Mi", func() {
			A := "1Gi"
			B := "10Mi"
			expectedResult := "1Gi"

			result, _, err := resourceStringComparison(A, B)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).Should(Equal(expectedResult))
		})
		It("Should 100Mi be larger than 10Mi", func() {
			A := "100Mi"
			B := "10Mi"
			expectedResult := "100Mi"

			result, _, err := resourceStringComparison(A, B)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).Should(Equal(expectedResult))
		})
	})

	Context("Compare Mixed Types", func() {
		It("Should compare string vs int64 using k8s resource quantities", func() {
			A := "1000m"  // 1000 millicores from size profile
			B := int64(2) // 2 cores from user CR = 2000m

			large, small := ResourceComparison(A, B)

			// 2 cores (2000m) > 1000m, so B should be larger
			Expect(large).Should(Equal(B))
			Expect(small).Should(Equal(A))
		})

		It("Should compare int64 vs string using k8s resource quantities", func() {
			A := int64(1) // 1 core from user CR = 1000m
			B := "2000m"  // 2000 millicores from size profile

			large, small := ResourceComparison(A, B)

			// 2000m > 1 core (1000m), so B should be larger
			Expect(large).Should(Equal(B))
			Expect(small).Should(Equal(A))
		})

		It("Should handle memory comparison with mixed types", func() {
			A := "512Mi"  // string from size profile
			B := int64(1) // int from user CR (treated as 1 byte)

			large, small := ResourceComparison(A, B)

			// 512Mi >> 1 byte, so A should be larger
			Expect(large).Should(Equal(A))
			Expect(small).Should(Equal(B))
		})
	})

	Context("Handle Invalid Resource Quantities", func() {
		It("Should prefer valid value when one is invalid string", func() {
			A := "test"       // Invalid resource quantity
			B := float64(1.5) // Valid float64

			large, small := ResourceComparison(A, B)

			// Should prefer B (valid value) over A (invalid)
			Expect(large).Should(Equal(B))
			Expect(small).Should(Equal(A))
		})

		It("Should prefer valid string when other is invalid", func() {
			A := "1000m"   // Valid
			B := "invalid" // Invalid

			large, small := ResourceComparison(A, B)

			// Should prefer A (valid value)
			Expect(large).Should(Equal(A))
			Expect(small).Should(Equal(B))
		})

		It("Should prefer resourceB (user value) when both are invalid", func() {
			A := "invalid1"
			B := "invalid2"

			large, small := ResourceComparison(A, B)

			// Should prefer B (user's value) as fallback
			Expect(large).Should(Equal(B))
			Expect(small).Should(Equal(A))
		})
	})
})
