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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
)

var _ = Describe("Render Template", func() {
	Describe("unmarshalHugePages", func() {
		It("should unmarshal the input map into a HugePages struct", func() {
			hugePages := map[string]interface{}{
				"enable":        true,
				"hugepages-2Gi": "",
				"hugepages-2Mi": "1Gi",
			}

			hugePagesStruct, err := UnmarshalHugePages(hugePages)
			Expect(err).To(BeNil())
			Expect(hugePagesStruct).To(Equal(&apiv3.HugePages{
				HugePagesSizes: map[string]string{
					"hugepages-2Gi": "",
					"hugepages-2Mi": "1Gi",
				},
			}))
		})

		It("should unmarshal the input map into a HugePages struct and sanitize the invalid value", func() {
			hugePages := map[string]interface{}{
				"enable":        true,
				"replica":       1, // invalid value
				"hugepages-2Gi": "",
				"hugepages-2Mi": "1Gi",
			}

			hugePagesStruct, err := UnmarshalHugePages(hugePages)
			Expect(err).To(BeNil())
			Expect(hugePagesStruct).To(Equal(&apiv3.HugePages{
				Enable: true,
				HugePagesSizes: map[string]string{
					"hugepages-2Gi": "",
					"hugepages-2Mi": "1Gi",
				},
			}))
		})
	})
})

func TestRenderTemplate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Render Template Suite")
}
