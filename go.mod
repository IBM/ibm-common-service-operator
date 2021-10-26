module github.com/IBM/ibm-common-service-operator

go 1.15

require (
	github.com/IBM/controller-filtered-cache v0.2.1
	github.com/IBM/ibm-namespace-scope-operator v1.0.1
	github.com/IBM/ibm-secretshare-operator v1.9.0
	github.com/IBM/operand-deployment-lifecycle-manager v1.5.0
	github.com/deckarep/golang-set v1.7.1
	github.com/ghodss/yaml v1.0.0
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/operator-framework/api v0.3.20
	github.com/operator-framework/operator-lifecycle-manager v0.17.0
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.9
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.6.2
)

// fix vulnerability: CVE-2021-3121 in github.com/gogo/protobuf < v1.3.2
replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
