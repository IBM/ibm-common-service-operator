//
// Copyright 2021 IBM Corporation
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
})
