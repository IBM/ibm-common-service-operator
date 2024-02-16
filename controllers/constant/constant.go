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

package constant

import (
	"time"
)

const (
	// OperatorNameEnvVar is the constant for env variable OPERATOR_NAME
	// which is the name of the current operator
	OperatorNameEnvVar = "OPERATOR_NAME"
	// OperatorNamespaceEnvVar is the constant for env variable OPERATOR_NAMESPACE
	// which is the namespace of the current operator
	OperatorNamespaceEnvVar = "OPERATOR_NAMESPACE"
	// UseExistingCluster is the constant for env variable USE_EXISTING_CLUSTER
	// it used to control unit test run into existing cluster or kubebuilder
	UseExistingCluster = "USE_EXISTING_CLUSTER"
	// Default CS CR
	MasterCR = "common-service"
	// CS main namespace
	MasterNamespace = "ibm-common-services"
	// CS kind
	KindCR = "CommonService"
	// CS api version
	APIVersion = "operator.ibm.com/v3"
	// Cluster Operator namespace
	ClusterOperatorNamespace = "openshift-operators"
	// CS map configMap
	CsMapConfigMap = "common-service-maps"
	// CS Saas configMap
	SaasConfigMap = "saas-config"
	// Namespace Scope Operator resource name
	NsSubResourceName = "nsSubscription"
	// Namespace Scope Operator sub name
	NsSubName = "ibm-namespace-scope-operator"
	// NamespaceScope CRs kind
	NssKindCR = "NamespaceScope"
	// NamespaceScope CRs api version
	NssAPIVersion = "operator.ibm.com/v1"
	// NamespaceScopeConfigmapName is the name of ConfigMap which stores the NamespaceScope Info
	NamespaceScopeConfigmapName = "namespace-scope"
	// NssCRFinalizer is the name for the finalizer to allow for deletion
	NssCRFinalizer = "finalizer.nss.operator.ibm.com"
	// Namespace Scope Operator Restricted resource name
	NsRestrictedSubResourceName = "nsRestrictedSubscription"
	// Namespace Scope Operator Restricted sub name
	NsRestrictedSubName = "ibm-namespace-scope-operator-restricted"
	//DefaultRequeueDuration is the default requeue time duration for request
	DefaultRequeueDuration = 20 * time.Second
	//CsMapsLabel is the label used to label the configmaps are managed by cs operator
	CsManagedLabel = "operator.ibm.com/managedByCsOperator"
	//CatalogsourceNs is the namespace of the catalogsource
	CatalogsourceNs = "openshift-marketplace"
	//CSCatalogsource is the name of the common service catalogsource
	CSCatalogsource = "opencloud-operators"
	//IBMCatalogsource is the names of the ibm catalogsource
	IBMCatalogsource = "ibm-operator-catalog"
	//Certified is the names of the ibm catalogsource
	CertifiedCatalogsource = "certified-operators"
	//Community is the names of the ibm catalogsource
	CommunityCatalogsource = "community-operators"
	//RedhatMarketplace is the names of the ibm catalogsource
	RedhatMarketplaceCatalogsource = "redhat-marketplace"
	//Redhat is the names of the ibm catalogsource
	RedhatCatalogsource = "redhat-operators"
	//IBMCSPackage is the package name of the ibm common service operator
	IBMCSPackage = "ibm-common-service-operator"
	//IBMODLMSPackage is the package name of the ODLM operator
	IBMODLMPackage = "operand-deployment-lifecycle-manager-app"
	//IBMNSSPackage is the package name of the namespace scope operator
	IBMNSSPackage = "ibm-namespace-scope-operator"
	// DevBuildImage is regular expression of the image address of internal dev build for testing
	DevBuildImage = `^hyc\-cloud\-private\-(.*)\-docker\-local\.artifactory\.swg\-devops\.com\/ibmcom\/ibm\-common\-service\-catalog\:(.*)`
	// BedrockCatalogsourcePriority is an annotation defined in the catalogsource
	BedrockCatalogsourcePriority = "bedrock_catalogsource_priority"
	// CSCACertificate is the name of cs-ca-certificate
	CSCACertificate = "cs-ca-certificate"
	// CertManagerSub is the name of ibm-cert-manager-operator subscription
	CertManagerSub = "ibm-cert-manager-operator"
	// CsClonedFromLabel is the label used to label the CommonService CR are cloned from the default CR in operatorNamespace
	CsClonedFromLabel = "operator.ibm.com/common-services.cloned-from"
	// IBMCPPCONFIG is the name of ibm-cpp-config ConfigMap
	IBMCPPCONFIG = "ibm-cpp-config"
)

// CsOg is OperatorGroup constent for the common service operator
const CsOperatorGroup = `
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: ibm-common-services-operators
  namespace: "placeholder"
spec:
  targetNamespaces:
  - "placeholder"
`

// CsCR is the default common service operator CR
const CsCR = `
apiVersion: operator.ibm.com/v3
kind: CommonService
metadata:
  annotations:
    version: "-1"
  name: common-service
  namespace: "{{ .OperatorNs }}"
spec:
  servicesNamespace:  "{{ .ServicesNs }}"
  operatorNamespace: "{{ .CPFSNs }}"
  size: starterset
  license:
    accept: false
`

// CsNoSizeCR is the default common service operator CR for upgrade
const CsNoSizeCR = `
apiVersion: operator.ibm.com/v3
kind: CommonService
metadata:
  annotations:
    version: "-1"
  name: common-service
  namespace: "placeholder"
spec:
  size: as-is
`

// CommonServiceMaps is the default common service maps ConfigMap
const CommonServiceMaps = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: common-service-maps
  namespace: kube-public
data:
  common-service-maps.yaml: |
    namespaceMapping:
    - requested-from-namespace:
      - "{{ .OperatorNs }}"
      map-to-common-service-namespace: "{{ .ServicesNs }}"
`
