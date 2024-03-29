package controllers

import (
	"testing"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Render Template", func() {
	Describe("unmarshalHugePages", func() {
		It("should unmarshal the input map into a HugePages struct", func() {
			hugePages := map[string]interface{}{
				"enable":        true,
				"hugepages-2Gi": "",
				"hugepages-2Mi": "1Gi",
			}

			hugePagesStruct, err := unmarshalHugePages(hugePages)
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

			hugePagesStruct, err := unmarshalHugePages(hugePages)
			Expect(err).To(BeNil())
			Expect(hugePagesStruct).To(Equal(&apiv3.HugePages{
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
