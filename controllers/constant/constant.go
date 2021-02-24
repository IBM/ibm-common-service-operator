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

package constant

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
	// CS main namespace
	MasterNamespace = "ibm-common-services"
	// Cluster Operator namespace
	ClusterOperatorNamespace = "openshift-operators"

	// Namespace Scope Operator resource name
	NsSubResourceName = "nsSubscription"
	// Namespace Scope Operator Restricted resource name
	NsRestrictedSubResourceName = "nsRestrictedSubscription"
	// Namespace Scope Operator sub name
	NsSubName = "ibm-namespace-scope-operator"
	// Namespace Scope Operator Restricted sub name
	NsRestrictedSubName = "ibm-namespace-scope-operator-restricted"
)

// CsOg is OperatorGroup constent for the common service operator
const CsOperatorGroup = `
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: ibm-common-services-operators
  namespace: placeholder
spec:
  targetNamespaces:
  - placeholder
`

// CsCR is the default common service operator CR
const CsCR = `
apiVersion: operator.ibm.com/v3
kind: CommonService
metadata:
  annotations:
    version: "-1"
  name: common-service
  namespace: placeholder
spec:
  size: medium
`

// CsNoSizeCR is the default common service operator CR for upgrade
const CsNoSizeCR = `
apiVersion: operator.ibm.com/v3
kind: CommonService
metadata:
  annotations:
    version: "-1"
  name: common-service
  namespace: placeholder
spec:
  size: as-is
`

// Cluster Admin RBAC
const ClusterAdminRBAC = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    version: "1"
  name: ibm-common-services-cluster-admin
rules:
- apiGroups:
  - "*"
  resources:
  - "*"
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ibm-common-services-cluster-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ibm-common-services-cluster-admin
subjects:
- kind: ServiceAccount
  name: operand-deployment-lifecycle-manager
  namespace: placeholder
`
