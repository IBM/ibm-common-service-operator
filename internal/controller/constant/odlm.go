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
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	utilyaml "github.com/ghodss/yaml"

	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
)

var (
	CSV4OperandRegistry     string
	CSV4SaasOperandRegistry string
	CSV4OperandConfig       string
	CSV4SaasOperandConfig   string
)

// ServiceNames defines the list of service names used in the OperandConfig template.
var ServiceNames = map[string][]string{
	"PostgreSQL": {
		"cloud-native-postgresql",
	},
	// Add more service categories as needed
}

const (
	ExcludedCatalog         = "certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog"
	StatusMonitoredServices = "ibm-idp-config-ui-operator,ibm-mongodb-operator,ibm-im-operator"
)

const (
	MongoDBOpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: {{ .ExcludedCatalog }}
    status-monitored-services: {{ .StatusMonitoredServices }}
spec:
  operators:
  - name: ibm-im-mongodb-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: v4.0
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-mongodb-operator-v4.1
    namespace: "{{ .CPFSNs }}"
    channel: v4.1
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-mongodb-operator-v4.2
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
`

	IMOpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: {{ .ExcludedCatalog }}
    status-monitored-services: {{ .StatusMonitoredServices }}
spec:
  operators:
  - name: ibm-im-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: v4.0
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.1
    namespace: "{{ .CPFSNs }}"
    channel: v4.1
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.2
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.3
    namespace: "{{ .CPFSNs }}"
    channel: v4.3
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.4
    namespace: "{{ .CPFSNs }}"
    channel: v4.4
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.5
    namespace: "{{ .CPFSNs }}"
    channel: v4.5
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.6
    namespace: "{{ .CPFSNs }}"
    channel: v4.6
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.7
    namespace: "{{ .CPFSNs }}"
    channel: v4.7
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.8
    namespace: "{{ .CPFSNs }}"
    channel: v4.8
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.9
    namespace: "{{ .CPFSNs }}"
    channel: v4.9
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.10
    namespace: "{{ .CPFSNs }}"
    channel: v4.10
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.11
    namespace: "{{ .CPFSNs }}"
    channel: v4.11
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.12
    namespace: "{{ .CPFSNs }}"
    channel: v4.12
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.13
    namespace: "{{ .CPFSNs }}"
    channel: v4.13
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
`

	IdpConfigUIOpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: {{ .ExcludedCatalog }}
    status-monitored-services: {{ .StatusMonitoredServices }}
spec:
  operators:
  - name: ibm-idp-config-ui-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: v4.0
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.1
    namespace: "{{ .CPFSNs }}"
    channel: v4.1
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.2
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.3
    namespace: "{{ .CPFSNs }}"
    channel: v4.3
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.4
    namespace: "{{ .CPFSNs }}"
    channel: v4.4
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.5
    namespace: "{{ .CPFSNs }}"
    channel: v4.5
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.6
    namespace: "{{ .CPFSNs }}"
    channel: v4.6
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.7
    namespace: "{{ .CPFSNs }}"
    channel: v4.7
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.8
    namespace: "{{ .CPFSNs }}"
    channel: v4.8
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.9
    namespace: "{{ .CPFSNs }}"
    channel: v4.9
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator-v4.10
    namespace: "{{ .CPFSNs }}"
    channel: v4.10
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
`

	PlatformUIOpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: {{ .ExcludedCatalog }}
    status-monitored-services: {{ .StatusMonitoredServices }}
spec:
  operators:
  - name: ibm-platformui-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: v4.0
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator-v4.1
    namespace: "{{ .CPFSNs }}"
    channel: v4.1
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator-v4.2
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator-v4.3
    namespace: "{{ .CPFSNs }}"
    channel: v4.3
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator-v4.4
    namespace: "{{ .CPFSNs }}"
    channel: v4.4
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator-v6.0
    namespace: "{{ .CPFSNs }}"
    channel: v6.0
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator-v6.1
    namespace: "{{ .CPFSNs }}"
    channel: v6.1
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator-v6.2
    namespace: "{{ .CPFSNs }}"
    channel: v6.2
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
`
)

const (
	KeyCloakOpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: {{ .ExcludedCatalog }}
    status-monitored-services: {{ .StatusMonitoredServices }}
spec:
  operators:
  - channel: ""
    fallbackChannels: []
    installPlanApproval: {{ .ApprovalMode }}
    name: keycloak-operator
    namespace: "{{ .ServicesNs }}"
    packageName: rhbk-operator
    scope: public
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: stable-v26
    installPlanApproval: {{ .ApprovalMode }}
    name: keycloak-operator-v26
    namespace: "{{ .ServicesNs }}"
    packageName: rhbk-operator
    scope: public
    configName: keycloak-operator
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: stable
    fallbackChannels:
      - stable-v1.25
      - stable-v1.22
    installPlanApproval: {{ .ApprovalMode }}
    name: edb-keycloak
    namespace: "{{ .CPFSNs }}"
    packageName: cloud-native-postgresql
    scope: public
    operatorConfig: cloud-native-postgresql-operator-config
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
`
)

const (
	CommonServicePGOpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: {{ .ExcludedCatalog }}
    status-monitored-services: {{ .StatusMonitoredServices }}
spec:
  operators:
  - channel: stable-v1.25
    fallbackChannels:
      - stable-v1.22
      - stable
    installPlanApproval: {{ .ApprovalMode }}
    name: common-service-postgresql
    namespace: "{{ .CPFSNs }}"
    packageName: cloud-native-postgresql
    scope: public
    operatorConfig: cloud-native-postgresql-operator-config
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
`
)

const (
	MongoDBOpCon = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: ibm-im-mongodb-operator-v4.0
    spec:
      mongoDB: {}
      operandRequest: {}
  - name: ibm-im-mongodb-operator-v4.1
    spec:
      mongoDB: {}
      operandRequest: {}
  - name: ibm-im-mongodb-operator-v4.2
    spec:
      mongoDB: {}
      operandRequest: {}
`

	IMOpCon = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: ibm-im-operator-v4.0
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
      operandRequest:
        requests:
          - operands:
              - name: ibm-im-mongodb-operator-v4.0
              - name: ibm-idp-config-ui-operator-v4.0
            registry: common-service
  - name: ibm-im-operator-v4.1
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
      operandRequest:
        requests:
          - operands:
              - name: ibm-im-mongodb-operator-v4.1
              - name: ibm-idp-config-ui-operator-v4.1
            registry: common-service
  - name: ibm-im-operator-v4.2
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
      operandRequest:
        requests:
          - operands:
              - name: ibm-im-mongodb-operator-v4.2
              - name: ibm-idp-config-ui-operator-v4.2
            registry: common-service
  - name: ibm-im-operator-v4.3
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
      operandRequest:
        requests:
          - operands:
              - name: ibm-im-mongodb-operator-v4.2
              - name: ibm-idp-config-ui-operator-v4.3
            registry: common-service
  - name: ibm-im-operator-v4.4
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
      operandRequest:
        requests:
          - operands:
              - name: ibm-im-mongodb-operator-v4.2
              - name: ibm-idp-config-ui-operator-v4.3
            registry: common-service
  - name: ibm-im-operator-v4.5
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
  - name: ibm-im-operator-v4.6
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
  - name: ibm-im-operator-v4.7
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
  - name: ibm-im-operator-v4.8
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
  - name: ibm-im-operator-v4.9
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
  - name: ibm-im-operator-v4.10
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
  - name: ibm-im-operator-v4.11
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
  - name: ibm-im-operator-v4.12
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
  - name: ibm-im-operator-v4.13
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo: 
        operand: ibm-im-operator
`

	UserMgmtOpCon = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: ibm-user-management-operator
    resources:
      - apiVersion: operator.ibm.com/v1alpha1
        labels:
          app.kubernetes.io/created-by: ibm-user-management-operator
          app.kubernetes.io/instance: accountiam-sample
          app.kubernetes.io/managed-by: kustomize
          app.kubernetes.io/name: accountiam
          app.kubernetes.io/part-of: ibm-user-management-operator
        kind: AccountIAM
        name: accountiam-sample
      - apiVersion: operator.ibm.com/v1alpha1
        data:
          spec:
            bindings:
              public-account-iam-config-dev:
                configmap: account-iam-env-configmap-development
              public-bootstrap-creds:
                secret: user-mgmt-bootstrap
              public-mcsp-integration-details:
                secret: mcsp-im-integration-details
            description: Binding information that should be accessible to User Management adopters
            operand: ibm-user-management-operator
            registry: common-service
            registryNamespace: {{ .ServicesNs }}
        force: true
        kind: OperandBindInfo
        name: ibm-user-mgmt-bindinfo
`

	IdpConfigUIOpCon = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: ibm-idp-config-ui-operator-v4.0
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.1
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.2
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.3
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.4
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.5
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.6
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.7
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.8
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.9
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator-v4.10
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
`

	PlatformUIOpCon = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: ibm-platformui-operator-v4.0
    resources:
      - apiVersion: apps/v1
        force: true
        kind: Deployment
        labels:
          operator.ibm.com/opreq-control: 'true'
        name: meta-api-deploy
        namespace: "{{ .CPFSNs }}"
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v4.1
    resources:
      - apiVersion: apps/v1
        force: true
        kind: Deployment
        labels:
          operator.ibm.com/opreq-control: 'true'
        name: meta-api-deploy
        namespace: "{{ .CPFSNs }}"
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v4.2
    resources:
      - apiVersion: apps/v1
        force: true
        kind: Deployment
        labels:
          operator.ibm.com/opreq-control: 'true'
        name: meta-api-deploy
        namespace: "{{ .CPFSNs }}"
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v4.3
    resources:
      - apiVersion: apps/v1
        force: true
        kind: Deployment
        labels:
          operator.ibm.com/opreq-control: 'true'
        name: meta-api-deploy
        namespace: "{{ .CPFSNs }}"
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v4.4
    resources:
      - apiVersion: apps/v1
        force: true
        kind: Deployment
        labels:
          operator.ibm.com/opreq-control: 'true'
        name: meta-api-deploy
        namespace: "{{ .CPFSNs }}"
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v6.0
    resources:
      - apiVersion: apps/v1
        force: true
        kind: Deployment
        labels:
          operator.ibm.com/opreq-control: 'true'
        name: meta-api-deploy
        namespace: "{{ .CPFSNs }}"
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v6.1
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v6.2
    spec:
      operandBindInfo: {}
`
)

const EDBOpCon = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  {{- range .ServiceNames.PostgreSQL }}
  - name: {{ . }}
    resources:
      - apiVersion: batch/v1
        kind: Job
        name: create-postgres-license-config
        namespace: "{{ $.OperatorNs }}"
        labels:
          operator.ibm.com/opreq-control: 'true'
        data:
          spec:
            activeDeadlineSeconds: 600
            backoffLimit: 5
            template:
              metadata:
                annotations:
                  productID: 068a62892a1e4db39641342e592daa25
                  productMetric: FREE
                  productName: IBM Cloud Platform Common Services
              spec:
                imagePullSecrets:
                  - name: ibm-entitlement-key
                affinity:
                  nodeAffinity:
                    requiredDuringSchedulingIgnoredDuringExecution:
                      nodeSelectorTerms:
                      - matchExpressions:
                        - key: kubernetes.io/arch
                          operator: In
                          values:
                          - amd64
                          - ppc64le
                          - s390x
                initContainers:
                - command:
                  - bash
                  - -c
                  - |
                    cat << EOF | kubectl apply -f -
                    apiVersion: v1
                    kind: Secret
                    type: Opaque
                    metadata:
                      name: postgresql-operator-controller-manager-config
                    data:
                      EDB_LICENSE_KEY: $(base64 /license_keys/edb/EDB_LICENSE_KEY | tr -d '\n')
                    EOF
                  image:
                    templatingValueFrom:
                      default:
                        required: true
                        configMapKeyRef:
                          name: cloud-native-postgresql-image-list
                          key: edb-postgres-license-provider-image
                          namespace: {{ $.OperatorNs }}
                      configMapKeyRef:
                        name: cloud-native-postgresql-operand-images-config
                        key: edb-postgres-license-provider-image
                        namespace: {{ $.OperatorNs }}
                  name: edb-license
                  resources:
                    limits:
                      cpu: 500m
                      memory: 512Mi
                    requests:
                      cpu: 100m
                      memory: 50Mi
                  securityContext:
                    allowPrivilegeEscalation: false
                    capabilities:
                      drop:
                      - ALL
                    privileged: false
                    readOnlyRootFilesystem: false
                containers:
                - command: ["bash", "-c"]
                  args:
                  - |
                    kubectl delete pods -l app.kubernetes.io/name=cloud-native-postgresql
                    kubectl annotate secret postgresql-operator-controller-manager-config ibm-license-key-applied="EDB Database with IBM License Key"
                  image:
                    templatingValueFrom:
                      default:
                        required: true
                        configMapKeyRef:
                          name: cloud-native-postgresql-image-list
                          key: edb-postgres-license-provider-image
                          namespace: {{ $.OperatorNs }}
                      configMapKeyRef:
                        name: cloud-native-postgresql-operand-images-config
                        key: edb-postgres-license-provider-image
                        namespace: {{ $.OperatorNs }}
                  name: restart-edb-pod
                  resources:
                    limits:
                      cpu: 500m
                      memory: 512Mi
                    requests:
                      cpu: 100m
                      memory: 50Mi
                  securityContext:
                    allowPrivilegeEscalation: false
                    capabilities:
                      drop:
                      - ALL
                    privileged: false
                    readOnlyRootFilesystem: false
                hostIPC: false
                hostNetwork: false
                hostPID: false
                restartPolicy: OnFailure
                securityContext:
                  runAsNonRoot: true
                serviceAccountName: edb-license-sa
      - apiVersion: v1
        kind: ServiceAccount
        name: edb-license-sa
        namespace: "{{ $.OperatorNs }}"
      - apiVersion: rbac.authorization.k8s.io/v1
        kind: Role
        name: edb-license-role
        namespace: "{{ $.OperatorNs }}"
        data:
          rules:
          - apiGroups: [""]
            resources: ["pods", "secrets"]
            verbs: ["create", "update", "patch", "get", "list", "delete", "watch"]
      - apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        name: edb-license-rolebinding
        namespace: "{{ $.OperatorNs }}"
        data:
          subjects:
          - kind: ServiceAccount
            name: edb-license-sa
          roleRef:
            kind: Role
            name: edb-license-role
            apiGroup: rbac.authorization.k8s.io
  {{- end }}
`

const (
	KeyCloakOpCon = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: keycloak-operator
    resources:
      - apiVersion: operator.ibm.com/v1alpha1
        data:
          spec:
            requests:
              - operands:
                  - name: edb-keycloak
                registry: common-service
                registryNamespace: {{ .ServicesNs }}
        force: true
        kind: OperandRequest
        name: edb-keycloak-request
      - apiVersion: operator.ibm.com/v1alpha1
        data:
          spec:
            bindings:
              public-keycloak-tls-secret:
                secret: cs-keycloak-tls-secret
              public-cs-keycloak-route:
                configmap: cs-keycloak-route
              public-cs-keycloak-service:
                configmap: cs-keycloak-service
            description: Binding information that should be accessible to Keycloak adopters
            operand:
              templatingValueFrom:
                conditional:
                  expression:
                    lessThan:
                      left:
                        objectRef:
                          apiVersion: apps/v1
                          kind: Deployment
                          name: rhbk-operator
                          path: .metadata.labels.olm\.owner
                      right:
                        literal: rhbk-operator.v26.0.0
                  then:
                    literal: "keycloak-operator" 
                  else:
                    literal: "keycloak-operator-v26"
            registry: common-service
            registryNamespace: {{ .ServicesNs }}
        force: true
        kind: OperandBindInfo
        name: keycloak-bindinfo
      - apiVersion: v1
        kind: ConfigMap
        name: cs-keycloak-entrypoint
        data:
          data:
            cs-keycloak-entrypoint.sh: |
              #!/usr/bin/env bash
              CA_DIR=/mnt/trust-ca
              USERPROFILE_DIR=/mnt/user-profile
              TRUSTSTORE_DIR=/mnt/truststore
              echo "Building the truststore file ..."
              cp /etc/pki/java/cacerts ${TRUSTSTORE_DIR}/keycloak-truststore.jks
              chmod +w ${TRUSTSTORE_DIR}/keycloak-truststore.jks
              echo "Importing default service account certificates ..."
              index=0
              while read -r line; do
                if [ "$line" = "-----BEGIN CERTIFICATE-----" ]; then
                  echo "$line" > ${TRUSTSTORE_DIR}/temp_cert.pem
                elif [ "$line" = "-----END CERTIFICATE-----" ]; then
                  echo "$line" >> ${TRUSTSTORE_DIR}/temp_cert.pem
                  let "index++"
                  echo "Importing service account certificate entry number ${index} ..."
                  keytool -importcert -alias "serviceaccount-ca-crt_$index" -file ${TRUSTSTORE_DIR}/temp_cert.pem -keystore ${TRUSTSTORE_DIR}/keycloak-truststore.jks -storepass changeit -noprompt
                  rm -f ${TRUSTSTORE_DIR}/temp_cert.pem
                else
                  echo "$line" >> ${TRUSTSTORE_DIR}/temp_cert.pem
                fi
              done < /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
              for cert in $(ls ${CA_DIR}); do
                echo "Importing ${cert} into the truststore file ..."
                keytool -importcert -file ${CA_DIR}/${cert} -keystore ${TRUSTSTORE_DIR}/keycloak-truststore.jks -storepass changeit -alias ${cert} -noprompt
              done
              echo "Truststore file built, starting Keycloak ..."
              "/opt/keycloak/bin/kc.sh" "$@" --spi-truststore-file-file=${TRUSTSTORE_DIR}/keycloak-truststore.jks --spi-truststore-file-password=changeit --spi-truststore-file-hostname-verification-policy=WILDCARD --spi-user-profile-declarative-user-profile-config-file=${USERPROFILE_DIR}/cs-keycloak-user-profile.json
      - apiVersion: v1
        data:
          data:
            cs-keycloak-user-profile.json: |
              {
                "attributes": [
                  {
                    "name": "username",
                    "displayName": "${username}",
                    "validations": {
                      "length": {
                        "min": 3,
                        "max": 255
                      },
                      "username-prohibited-characters": {},
                      "up-username-not-idn-homograph": {}
                    },
                    "permissions": {
                      "view": [
                        "admin",
                        "user"
                      ],
                      "edit": [
                        "admin",
                        "user"
                      ]
                    },
                    "multivalued": false
                  },
                  {
                    "name": "email",
                    "displayName": "${email}",
                    "validations": {
                      "email": {},
                      "length": {
                        "max": 255
                      }
                    },
                    "annotations": {},
                    "permissions": {
                      "view": [
                        "admin",
                        "user"
                      ],
                      "edit": [
                        "admin",
                        "user"
                      ]
                    },
                    "multivalued": false
                  },
                  {
                    "name": "firstName",
                    "displayName": "${firstName}",
                    "validations": {
                      "length": {
                        "max": 255
                      },
                      "person-name-prohibited-characters": {}
                    },
                    "permissions": {
                      "view": [
                        "admin",
                        "user"
                      ],
                      "edit": [
                        "admin",
                        "user"
                      ]
                    },
                    "multivalued": false
                  },
                  {
                    "name": "lastName",
                    "displayName": "${lastName}",
                    "validations": {
                      "length": {
                        "max": 255
                      },
                      "person-name-prohibited-characters": {}
                    },
                    "permissions": {
                      "view": [
                        "admin",
                        "user"
                      ],
                      "edit": [
                        "admin",
                        "user"
                      ]
                    },
                    "multivalued": false
                  }
                ],
                "groups": [
                  {
                    "name": "user-metadata",
                    "displayHeader": "User metadata",
                    "displayDescription": "Attributes, which refer to user metadata"
                  }
                ]
              }
        force: true
        kind: ConfigMap
        name: cs-keycloak-user-profile
      - apiVersion: v1
        kind: ServiceAccount
        name: cs-keycloak-pre-upgrade-sa
      - apiVersion: rbac.authorization.k8s.io/v1
        kind: Role
        name: cs-keycloak-pre-upgrade-role
        data:
          rules:
          - apiGroups: [""]
            resources: ["configmaps", "secrets"]
            verbs: ["create", "update", "patch", "get", "list", "delete", "watch"]
      - apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        name: cs-keycloak-pre-upgrade-rolebinding
        data:
          subjects:
          - kind: ServiceAccount
            name: cs-keycloak-pre-upgrade-sa
          roleRef:
            kind: Role
            name: cs-keycloak-pre-upgrade-role
            apiGroup: rbac.authorization.k8s.io
      - apiVersion: v1
        kind: ConfigMap
        name: cs-keycloak-pre-upgrade
        data:
          data:
            cs-keycloak-pre-upgrade.sh: |
              #!/usr/bin/env bash
              # Check if the secret already exists
              if oc get secret cs-keycloak-ca-certs >/dev/null 2>&1; then
                echo "Secret cs-keycloak-ca-certs already exists. Skipping conversion."
                exit 0
              fi
              
              # Check if ConfigMap exists
              if ! oc get configmap cs-keycloak-ca-certs >/dev/null 2>&1; then
                echo "ConfigMap cs-keycloak-ca-certs not found. Nothing to conversion."
                exit 0
              fi
              
              # Extract certificate file names from ConfigMap
              CERT_FILES=$(oc get configmap cs-keycloak-ca-certs -o yaml | yq e '.data | keys | .[]' -)              
              
              # Create a temporary directory
              mkdir -p /tmp/certs
              # Extract certificates from ConfigMap and save them as files
              for CERT in $CERT_FILES; do
                oc get configmap cs-keycloak-ca-certs -o yaml | yq e ".data[\"$CERT\"]"> /tmp/certs/$CERT
              done
              
              # Create Secret from extracted certificates
              oc create secret generic cs-keycloak-ca-certs \
                $(for CERT in $CERT_FILES; do echo --from-file=/tmp/certs/$CERT; done)
              
              echo "Conversion complete. Secret created: cs-keycloak-ca-certs"
      - apiVersion: batch/v1
        kind: Job
        force: true
        name: cs-keycloak-pre-upgrade-job
        data:
          spec:
            template:
              spec:
                affinity:
                  nodeAffinity:
                    requiredDuringSchedulingIgnoredDuringExecution:
                      nodeSelectorTerms:
                      - matchExpressions:
                        - key: kubernetes.io/arch
                          operator: In
                          values:
                          - amd64
                          - ppc64le
                          - s390x
                restartPolicy: OnFailure
                serviceAccountName: cs-keycloak-pre-upgrade-sa
                containers:
                  - name: cs-keycloak-pre-upgrade-job
                    image: {{ .UtilsImage }}
                    command: ["/bin/sh", "/mnt/scripts/cs-keycloak-pre-upgrade.sh"]
                    volumeMounts:
                      - name: script-volume
                        mountPath: /mnt/scripts
                volumes:
                  - name: script-volume
                    configMap:
                      name: cs-keycloak-pre-upgrade
      - apiVersion: v1
        annotations:
          service.beta.openshift.io/serving-cert-secret-name: cpfs-opcon-cs-keycloak-tls-secret
        labels:
          app: keycloak
          app.kubernetes.io/instance: cs-keycloak
          app.kubernetes.io/managed-by: keycloak-operator
        data:
          spec:
            internalTrafficPolicy: Cluster
            ipFamilies:
              - IPv4
            ipFamilyPolicy: SingleStack
            ports:
              - name: https
                port: 8443
                protocol: TCP
                targetPort: 8443
            selector:
              app: keycloak
              app.kubernetes.io/instance: cs-keycloak
              app.kubernetes.io/managed-by: keycloak-operator
            sessionAffinity: None
            type: ClusterIP
        force: true
        kind: Service
        name: cpfs-opcon-cs-keycloak-service
      - apiVersion: v1
        labels:
          operator.ibm.com/opreq-control: 'true'
          operator.ibm.com/watched-by-cert-manager: ''
        data:
          stringData:
            ca.crt:
              templatingValueFrom:
                configMapKeyRef:
                  key: service-ca.crt
                  name: openshift-service-ca.crt
                required: true
            tls.crt:
              templatingValueFrom:
                required: true
                secretKeyRef:
                  key: tls.crt
                  name: cpfs-opcon-cs-keycloak-tls-secret
            tls.key:
              templatingValueFrom:
                required: true
                secretKeyRef:
                  key: tls.key
                  name: cpfs-opcon-cs-keycloak-tls-secret
          type: kubernetes.io/tls
        force: true
        kind: Secret
        name: cs-keycloak-tls-secret
      - apiVersion: route.openshift.io/v1
        data:
          spec:
            host:
              templatingValueFrom:
                configMapKeyRef:
                  key: keycloak_route_name
                  name: ibm-cpp-config
            port:
              targetPort: 8443
            tls:
              caCertificate:
                templatingValueFrom:
                  secretKeyRef:
                    key: ca.crt
                    name: keycloak-custom-tls-secret
              certificate:
                templatingValueFrom:
                  secretKeyRef:
                    key: tls.crt
                    name: keycloak-custom-tls-secret
              destinationCACertificate:
                templatingValueFrom:
                  required: true
                  secretKeyRef:
                    key: ca.crt
                    name: cs-keycloak-tls-secret
              key:
                templatingValueFrom:
                  secretKeyRef:
                    key: tls.key
                    name: keycloak-custom-tls-secret
              termination: reencrypt
            to:
              kind: Service
              name: cpfs-opcon-cs-keycloak-service
            wildcardPolicy: None
        force: true
        kind: Route
        name: keycloak
      - apiVersion: k8s.keycloak.org/v2alpha1
        data:
          spec:
            scheduling:
              affinity:
                nodeAffinity:
                  requiredDuringSchedulingIgnoredDuringExecution:
                    nodeSelectorTerms:
                    - matchExpressions:
                      - key: kubernetes.io/arch
                        operator: In
                        values:
                        - amd64
                        - ppc64le
                        - s390x
            truststores:
              my-truststore:
                secret:
                  name: cs-keycloak-ca-certs
                  optional: true
            proxy:
              headers: xforwarded
            features:
              enabled:
                - token-exchange
                - admin-fine-grained-authz
            db:
              host: keycloak-edb-cluster-rw
              passwordSecret:
                key: password
                name: keycloak-edb-cluster-app
              usernameSecret:
                key: username
                name: keycloak-edb-cluster-app
              vendor: postgres
            hostname:
              hostname:
                templatingValueFrom:
                  conditional:
                    expression:
                      greaterThan:
                        left:
                          objectRef:
                            apiVersion: apps/v1
                            kind: Deployment
                            name: rhbk-operator
                            path: .metadata.labels.olm\.owner
                        right:
                          literal: rhbk-operator.v26.0.0
                    then:
                      objectRef:
                        apiVersion: route.openshift.io/v1
                        kind: Route
                        name: keycloak
                        path: 'https://+.spec.host'
                      required: true
                    else:                        
                      objectRef:
                        apiVersion: route.openshift.io/v1
                        kind: Route
                        name: keycloak
                        path: .spec.host
                      required: true
              backchannelDynamic:
                templatingValueFrom:
                  conditional:
                    expression:
                      and:
                        - greaterThan:
                            left:
                              objectRef:
                                apiVersion: apps/v1
                                kind: Deployment
                                name: rhbk-operator
                                path: .metadata.labels.olm\.owner
                            right:
                              literal: rhbk-operator.v26.0.0
                        - equal:
                            left:
                              objectRef:
                                apiVersion: apiextensions.k8s.io/v1
                                kind: CustomResourceDefinition
                                name: keycloaks.k8s.keycloak.org
                                path: .spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.hostname.properties.backchannelDynamic.type
                            right:
                              literal: "boolean"
                    then:
                      boolean: true
            additionalOptions:
              templatingValueFrom:
                conditional:
                  expression:
                    and:
                      - greaterThan:
                          left:
                            objectRef:
                              apiVersion: apps/v1
                              kind: Deployment
                              name: rhbk-operator
                              path: .metadata.labels.olm\.owner
                          right:
                            literal: rhbk-operator.v26.0.0
                      - notEqual:
                          left:
                            objectRef:
                              apiVersion: apiextensions.k8s.io/v1
                              kind: CustomResourceDefinition
                              name: keycloaks.k8s.keycloak.org
                              path: .spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.hostname.properties.backchannelDynamic.type
                          right:
                            literal: "boolean"
                  then:
                    array:
                      - map:
                          name: spi-user-profile-declarative-user-profile-config-file
                          value: /mnt/user-profile/cs-keycloak-user-profile.json
                      - map:
                          name: hostname-backchannel-dynamic
                          value: "true"
                  else:
                    array:
                      - map:
                          name: spi-user-profile-declarative-user-profile-config-file
                          value: /mnt/user-profile/cs-keycloak-user-profile.json
            http:
              tlsSecret: cs-keycloak-tls-secret
            ingress:
              enabled: false
            unsupported:
              podTemplate:
                metadata:
                  annotations:
                    cloudpakThemesVersion:
                      templatingValueFrom:
                        objectRef:
                          apiVersion: v1
                          kind: ConfigMap
                          name: cs-keycloak-theme
                          path: .metadata.annotations.themesVersion
                        required: true
                spec:
                  containers:
                    - command: ["/bin/sh", "/mnt/startup/cs-keycloak-entrypoint.sh"]
                      volumeMounts:
                        - mountPath: /mnt/truststore
                          name: truststore-volume
                        - mountPath: /mnt/startup
                          name: startup-volume
                        - mountPath: /mnt/trust-ca
                          name: trust-ca-volume
                        - mountPath: /opt/keycloak/providers
                          name: cs-keycloak-theme
                        - mountPath: /mnt/user-profile
                          name: user-profile-volume
                  volumes:
                    - name: truststore-volume
                      emptyDir:
                        sizeLimit: 2Mi
                    - name: startup-volume
                      configMap:
                        name: cs-keycloak-entrypoint                      
                    - name: trust-ca-volume
                      configMap:
                        name: cs-keycloak-ca-certs
                        optional: true
                    - name: cs-keycloak-theme
                      configMap:
                        items:
                          - key: cloudpak-theme.jar
                            path: cloudpak-theme.jar
                        name: cs-keycloak-theme
                    - name: user-profile-volume
                      configMap: 
                        name: cs-keycloak-user-profile
                  affinity:
                    nodeAffinity:
                      requiredDuringSchedulingIgnoredDuringExecution:
                        nodeSelectorTerms:
                        - matchExpressions:
                          - key: kubernetes.io/arch
                            operator: In
                            values:
                            - amd64
                            - ppc64le
                            - s390x
        force: true
        kind: Keycloak
        name: cs-keycloak
        optionalFields:
          - path: .spec.unsupported.podTemplate.spec.containers[0].resources
            operation: remove
            matchExpressions:
              - objectRef:
                  name: keycloaks.k8s.keycloak.org
                  apiVersion: apiextensions.k8s.io/v1
                  kind: CustomResourceDefinition
                key: .spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.resources
                operator: Exists
          - path: .spec.resources
            operation: remove
            matchExpressions:
              - objectRef:
                  name: keycloaks.k8s.keycloak.org
                  apiVersion: apiextensions.k8s.io/v1
                  kind: CustomResourceDefinition
                key: .spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.resources
                operator: DoesNotExist
          - path: .spec.unsupported.podTemplate.spec.affinity
            operation: remove
            matchExpressions:
              - objectRef:
                  name: keycloaks.k8s.keycloak.org
                  apiVersion: apiextensions.k8s.io/v1
                  kind: CustomResourceDefinition
                key: .spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.scheduling
                operator: Exists
          - path: .spec.scheduling
            operation: remove
            matchExpressions:
              - objectRef:
                  name: keycloaks.k8s.keycloak.org
                  apiVersion: apiextensions.k8s.io/v1
                  kind: CustomResourceDefinition
                key: .spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.scheduling
                operator: DoesNotExist
          - path: .spec.unsupported.podTemplate.spec.containers[0].command
            operation: remove
            matchExpressions:
              - objectRef:
                  name: keycloaks.k8s.keycloak.org
                  apiVersion: apiextensions.k8s.io/v1
                  kind: CustomResourceDefinition
                key: .spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.truststores
                operator: Exists
          - path: .spec.truststores
            operation: remove
            matchExpressions:
              - objectRef:
                  name: keycloaks.k8s.keycloak.org
                  apiVersion: apiextensions.k8s.io/v1
                  kind: CustomResourceDefinition
                key: .spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.truststores
                operator: DoesNotExist
      - apiVersion: v1
        kind: ConfigMap
        force: true
        name: cs-keycloak-route
        data:
          data:
            HOSTNAME:
              templatingValueFrom:
                objectRef:
                  apiVersion: route.openshift.io/v1
                  kind: Route
                  name: keycloak
                  path: https://+.spec.host
                required: true
            TERMINATION:
              templatingValueFrom:
                objectRef:
                  apiVersion: route.openshift.io/v1
                  kind: Route
                  name: keycloak
                  path: .spec.tls.termination
                required: true
            BACKEND_SERVICE:
              templatingValueFrom:
                objectRef:
                  apiVersion: route.openshift.io/v1
                  kind: Route
                  name: keycloak
                  path: .spec.to.name
                required: true
      - apiVersion: v1
        kind: ConfigMap
        force: true
        name: cs-keycloak-service
        data:
          data:
            PORT:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: cpfs-opcon-cs-keycloak-service
                  path: .spec.ports[0].port
                required: true
            CLUSTER_IP:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: cpfs-opcon-cs-keycloak-service
                  path: .spec.clusterIP
                required: true
            SERVICE_NAME:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: cpfs-opcon-cs-keycloak-service
                  path: .metadata.name
                required: true
            SERVICE_NAMESPACE: {{ .ServicesNs }}
            SERVICE_ENDPOINT:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: cpfs-opcon-cs-keycloak-service
                  path: https://+.metadata.name+.+.metadata.namespace+.+svc:+.spec.ports[0].port
      - apiVersion: k8s.keycloak.org/v2alpha1
        kind: KeycloakRealmImport
        name: cs-cloudpak-realm
        force: true
        ownerReferences:
          - apiVersion: k8s.keycloak.org/v2alpha1
            kind: Keycloak
            name: cs-keycloak
            controller: false
        data:
          spec:
            keycloakCRName: cs-keycloak
            realm:
              displayName: IBM Cloud Pak
              displayNameHtml: "<div class=\"kc-logo-text\"><span>IBM Cloud Pak</span></div>"
              enabled: true
              id: cloudpak
              realm: cloudpak
              ssoSessionIdleTimeout: 43200
              ssoSessionMaxLifespan: 43200
              rememberMe: true
              passwordPolicy: "length(15) and notUsername(undefined) and notEmail(undefined)"
              loginTheme: cloudpak
              adminTheme: cloudpak
              accountTheme: cloudpak
              emailTheme: cloudpak
              internationalizationEnabled: true
              supportedLocales: [ "en", "de" , "es", "fr", "it", "ja", "ko", "pt_BR", "zh_CN", "zh_TW"]
  - name: edb-keycloak
    resources:
      - apiVersion: batch/v1
        kind: Job
        force: true
        name: create-postgres-license-config
        namespace: "{{ .OperatorNs }}"
        labels:
          operator.ibm.com/opreq-control: 'true'
        data:
          spec:
            activeDeadlineSeconds: 600
            backoffLimit: 5
            template:
              metadata:
                annotations:
                  productID: 068a62892a1e4db39641342e592daa25
                  productMetric: FREE
                  productName: IBM Cloud Platform Common Services
              spec:
                imagePullSecrets:
                  - name: ibm-entitlement-key
                affinity:
                  nodeAffinity:
                    requiredDuringSchedulingIgnoredDuringExecution:
                      nodeSelectorTerms:
                      - matchExpressions:
                        - key: kubernetes.io/arch
                          operator: In
                          values:
                          - amd64
                          - ppc64le
                          - s390x
                initContainers:
                - command:
                  - bash
                  - -c
                  - |
                    cat << EOF | kubectl apply -f -
                    apiVersion: v1
                    kind: Secret
                    type: Opaque
                    metadata:
                      name: postgresql-operator-controller-manager-config
                    data:
                      EDB_LICENSE_KEY: $(base64 /license_keys/edb/EDB_LICENSE_KEY | tr -d '\n')
                    EOF
                  image:
                    templatingValueFrom:
                      default:
                        required: true
                        configMapKeyRef:
                          name: cloud-native-postgresql-image-list
                          key: edb-postgres-license-provider-image
                          namespace: {{ .OperatorNs }}
                      configMapKeyRef:
                        name: cloud-native-postgresql-operand-images-config
                        key: edb-postgres-license-provider-image
                        namespace: {{ $.OperatorNs }}
                  name: edb-license
                  resources:
                    limits:
                      cpu: 500m
                      memory: 512Mi
                    requests:
                      cpu: 100m
                      memory: 50Mi
                  securityContext:
                    allowPrivilegeEscalation: false
                    capabilities:
                      drop:
                      - ALL
                    privileged: false
                    readOnlyRootFilesystem: false
                containers:
                - command: ["bash", "-c"]
                  args:
                  - |
                    kubectl delete pods -l app.kubernetes.io/name=cloud-native-postgresql
                    kubectl annotate secret postgresql-operator-controller-manager-config ibm-license-key-applied="EDB Database with IBM License Key"
                  image:
                    templatingValueFrom:
                      default:
                        required: true
                        configMapKeyRef:
                          name: cloud-native-postgresql-image-list
                          key: edb-postgres-license-provider-image
                          namespace: {{ .OperatorNs }}
                      configMapKeyRef:
                        name: cloud-native-postgresql-operand-images-config
                        key: edb-postgres-license-provider-image
                        namespace: {{ $.OperatorNs }}
                  name: restart-edb-pod
                  resources:
                    limits:
                      cpu: 500m
                      memory: 512Mi
                    requests:
                      cpu: 100m
                      memory: 50Mi
                  securityContext:
                    allowPrivilegeEscalation: false
                    capabilities:
                      drop:
                      - ALL
                    privileged: false
                    readOnlyRootFilesystem: false
                hostIPC: false
                hostNetwork: false
                hostPID: false
                restartPolicy: OnFailure
                securityContext:
                  runAsNonRoot: true
                serviceAccountName: edb-license-sa
      - apiVersion: v1
        kind: ServiceAccount
        name: edb-license-sa
        namespace: "{{ .OperatorNs }}"
      - apiVersion: rbac.authorization.k8s.io/v1
        kind: Role
        name: edb-license-role
        namespace: "{{ .OperatorNs }}"
        data:
          rules:
          - apiGroups: [""]
            resources: ["pods", "secrets"]
            verbs: ["create", "update", "patch", "get", "list", "delete", "watch"] 
      - apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        name: edb-license-rolebinding
        namespace: "{{ .OperatorNs }}"
        data:
          subjects:
          - kind: ServiceAccount
            name: edb-license-sa
          roleRef:
            kind: Role
            name: edb-license-role
            apiGroup: rbac.authorization.k8s.io
      - apiVersion: postgresql.k8s.enterprisedb.io/v1
        data:
          spec:
            inheritedMetadata:
              annotations:
                backup.velero.io/backup-volumes: pgdata,pg-wal
              labels:
                foundationservices.cloudpak.ibm.com: keycloak
            description:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Secret
                  name: postgresql-operator-controller-manager-config
                  path: .metadata.annotations.ibm-license-key-applied
                  namespace: {{ .OperatorNs }}
                required: true
            bootstrap:
              initdb:
                database: keycloak
                owner: app
            imageName:
              templatingValueFrom:
                default:
                  required: true
                  configMapKeyRef:
                    name: cloud-native-postgresql-image-list
                    key: ibm-postgresql-14-operand-image
                    namespace: {{ .OperatorNs }}
                configMapKeyRef:
                  name: cloud-native-postgresql-operand-images-config
                  key: ibm-postgresql-14-operand-image
                  namespace: {{ .OperatorNs }}
            imagePullSecrets:
              - name: ibm-entitlement-key
            logLevel: info
            primaryUpdateStrategy: unsupervised
            primaryUpdateMethod: switchover
            enableSuperuserAccess: true
            replicationSlots:
              highAvailability:
                enabled: false
            storage:
              size: 1Gi
            walStorage:
              size: 1Gi
        force: true
        annotations:
          k8s.enterprisedb.io/addons: '["velero"]'
          k8s.enterprisedb.io/snapshotAllowColdBackupOnPrimary: enabled
          productID: 068a62892a1e4db39641342e592daa25
          productMetric: FREE
          productName: IBM Cloud Platform Common Services
        labels:
          foundationservices.cloudpak.ibm.com: keycloak
        kind: Cluster
        name: keycloak-edb-cluster
`
)

const (
	CommonServicePGOpCon = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: common-service-postgresql
    resources:
      - apiVersion: operator.ibm.com/v1alpha1
        data:
          spec:
            requests:
              - operands:
                  - name: cloud-native-postgresql-v1.25
                registry: common-service
                registryNamespace: {{ .ServicesNs }}
        force: true
        kind: OperandRequest
        name: postgresql-operator-request
      - apiVersion: cert-manager.io/v1
        kind: Certificate
        name: common-service-db-replica-tls-cert
        labels:
            app.kubernetes.io/component: common-service-db-replica-tls-cert
            component: common-service-db-replica-tls-cert
        data:
          spec:
            commonName: streaming_replica
            duration: 2160h0m0s
            issuerRef:
              kind: Issuer
              name: cs-ca-issuer
            renewBefore: 720h0m0s
            secretName: common-service-db-replica-tls-secret
            secretTemplate:
              labels:
                k8s.enterprisedb.io/reload: ''
            usages:
              - client auth
      - apiVersion: cert-manager.io/v1
        kind: Certificate
        labels:
            app.kubernetes.io/component: common-service-db-tls-cert
            component: common-service-db-tls-cert
        name: common-service-db-tls-cert
        data:  
          spec:
            dnsNames:
              - common-service-db
              - common-service-db.{{ .ServicesNs }}
              - common-service-db.{{ .ServicesNs }}.svc
              - common-service-db-r
              - common-service-db-r.{{ .ServicesNs }}
              - common-service-db-r.{{ .ServicesNs }}.svc
              - common-service-db-ro
              - common-service-db-ro.{{ .ServicesNs }}
              - common-service-db-ro.{{ .ServicesNs }}.svc
              - common-service-db-rw
              - common-service-db-rw.{{ .ServicesNs }}
              - common-service-db-rw.{{ .ServicesNs }}.svc
            duration: 8760h0m0s
            issuerRef:
              kind: Issuer
              name: cs-ca-issuer
            renewBefore: 720h0m0s
            secretName: common-service-db-tls-secret
            secretTemplate:
              labels:
                k8s.enterprisedb.io/reload: ''
            usages:
              - server auth
      - apiVersion: cert-manager.io/v1
        kind: Certificate
        name: common-service-db-im-tls-cert
        data:
          spec:
            commonName: im_user
            duration: 2160h0m0s
            issuerRef:
              kind: Issuer
              name: cs-ca-issuer
            renewBefore: 720h0m0s
            secretName: common-service-db-im-tls-secret
            secretTemplate:
              labels:
                app.kubernetes.io/instance: common-service-db-im-tls-secret
                app.kubernetes.io/name: common-service-db-im-tls-secret
            usages:
              - client auth
      - apiVersion: cert-manager.io/v1
        kind: Certificate
        name: common-service-db-zen-tls-cert
        data:
          spec:
            commonName: zen_user
            duration: 2160h0m0s
            issuerRef:
              kind: Issuer
              name: cs-ca-issuer
            renewBefore: 720h0m0s
            secretName: common-service-db-zen-tls-secret
            secretTemplate:
              labels:
                app.kubernetes.io/instance: common-service-db-zen-tls-secret
                app.kubernetes.io/name: common-service-db-zen-tls-secret
            usages:
              - client auth
      - apiVersion: operator.ibm.com/v1alpha1
        data:
          spec:
            bindings:
              protected-zen-db:
                configmap: common-service-db-zen
                secret: common-service-db-zen-tls-secret
              protected-im-db:
                configmap: common-service-db-im
                secret: common-service-db-im-tls-secret
              private-superuser-db:
                secret: common-service-db-superuser
            description: Binding information that should be accessible to Common Service Postgresql Adopters
            operand: common-service-postgresql
            registry: common-service
            registryNamespace: {{ .ServicesNs }}
        force: true
        kind: OperandBindInfo
        name: common-service-postgresql-bindinfo
      - apiVersion: postgresql.k8s.enterprisedb.io/v1
        kind: Cluster
        name: common-service-db          
        force: true
        annotations:
          productID: 068a62892a1e4db39641342e592daa25
          productMetric: FREE
          productName: IBM Cloud Platform Common Services
        labels:
          foundationservices.cloudpak.ibm.com: cs-db
        data:
          spec:
            inheritedMetadata:
              labels:
                foundationservices.cloudpak.ibm.com: cs-db
            description:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Secret
                  name: postgresql-operator-controller-manager-config
                  path: .metadata.annotations.ibm-license-key-applied
                  namespace: {{ .OperatorNs }}
                required: true
            bootstrap:
              initdb:
                database: im
                owner: im_user
                dataChecksums: true
                postInitApplicationSQL:
                  - CREATE USER zen_user
                  - CREATE DATABASE zen OWNER zen_user
                  - GRANT ALL PRIVILEGES ON DATABASE zen TO zen_user
            affinity:
              nodeAffinity:
                requiredDuringSchedulingIgnoredDuringExecution:
                  nodeSelectorTerms:
                    - matchExpressions:
                        - key: kubernetes.io/arch
                          operator: In
                          values:
                            - amd64
                            - ppc64le
                            - s390x
              additionalPodAntiAffinity:
                preferredDuringSchedulingIgnoredDuringExecution:
                  - podAffinityTerm:
                      labelSelector:
                        matchExpressions:
                          - key: k8s.enterprisedb.io/cluster
                            operator: In
                            values:
                              - common-service-db
                      topologyKey: kubernetes.io/hostname
                    weight: 50
              podAntiAffinityType: preferred
              topologyKey: topology.kubernetes.io/zone
            topologySpreadConstraints:
            - maxSkew: 1
              topologyKey: topology.kubernetes.io/zone
              whenUnsatisfiable: ScheduleAnyway
              labelSelector:
                matchExpressions:
                  - key: k8s.enterprisedb.io/cluster
                    operator: In
                    values:
                      - common-service-db
            - maxSkew: 1
              topologyKey: topology.kubernetes.io/region
              whenUnsatisfiable: ScheduleAnyway
              labelSelector:
                matchExpressions:
                  - key: k8s.enterprisedb.io/cluster
                    operator: In
                    values:
                      - common-service-db
            imageName:
              templatingValueFrom:
                default:
                  required: true
                  configMapKeyRef:
                    name: cloud-native-postgresql-image-list
                    key: ibm-postgresql-16-operand-image
                    namespace: {{ .OperatorNs }}
                configMapKeyRef:
                  name: cloud-native-postgresql-operand-images-config
                  key: ibm-postgresql-16-operand-image
                  namespace: {{ .OperatorNs }}
            imagePullSecrets:
              - name: ibm-entitlement-key
            logLevel: info
            primaryUpdateStrategy: unsupervised
            primaryUpdateMethod: switchover
            enableSuperuserAccess: true
            replicationSlots:
              highAvailability:
                enabled: true
            certificates:
              clientCASecret: cs-ca-certificate-secret
              replicationTLSSecret: common-service-db-replica-tls-secret
              serverCASecret: cs-ca-certificate-secret
              serverTLSSecret: common-service-db-tls-secret
            startDelay: 120
            stopDelay: 90
            storage:
              resizeInUseVolumes: true
              size: 10Gi
            walStorage:
              resizeInUseVolumes: true
              size: 10Gi
            postgresql:
              parameters:
                track_activities: "on"
                track_counts: "on"
                track_io_timing: "on"
                pg_stat_statements.track: all
                pg_stat_statements.max: "10000"
                max_slot_wal_keep_size: "8GB"
              pg_hba:
                - hostssl im im_user all cert
                - hostssl zen zen_user all cert
                - host zen instana_user all scram-sha-256
                - host im instana_user all scram-sha-256
      - apiVersion: v1
        kind: ConfigMap
        force: true
        name: common-service-db-zen
        data:
          data:
            IS_EMBEDDED: 'true'
            DATABASE_PORT:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: common-service-db-rw
                  path: .spec.ports[0].port
                required: true
            DATABASE_R_ENDPOINT:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: common-service-db-r
                  path: .metadata.name+.+.metadata.namespace+.+svc
                required: true
            DATABASE_RW_ENDPOINT:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: common-service-db-rw
                  path: .metadata.name+.+.metadata.namespace+.+svc
                required: true
            DATABASE_NAME: zen
            DATABASE_USER: zen_user
            DATABASE_CA_CERT: ca.crt
            DATABASE_CLIENT_KEY: tls.key
            DATABASE_CLIENT_CERT: tls.crt
      - apiVersion: v1
        kind: ConfigMap
        force: true
        name: common-service-db-im
        data:
          data:
            IS_EMBEDDED: 'true'
            DATABASE_PORT:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: common-service-db-rw
                  path: .spec.ports[0].port
                required: true
            DATABASE_R_ENDPOINT:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: common-service-db-r
                  path: .metadata.name+.+.metadata.namespace+.+svc
                required: true
            DATABASE_RW_ENDPOINT:
              templatingValueFrom:
                objectRef:
                  apiVersion: v1
                  kind: Service
                  name: common-service-db-rw
                  path: .metadata.name+.+.metadata.namespace+.+svc
                required: true
            DATABASE_NAME: im
            DATABASE_USER: im_user
            DATABASE_CA_CERT: ca.crt
            DATABASE_CLIENT_KEY: tls.key
            DATABASE_CLIENT_CERT: tls.crt
`
)

const (
	CSV3OpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: "{{ .Version }}"
    excluded-catalogsource: {{ .ExcludedCatalog }}
    status-monitored-services: {{ .StatusMonitoredServices }}
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-mongodb-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-cert-manager-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-iam-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-healthcheck-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-healthcheck-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-commonui-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-management-ingress-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-ingress-nginx-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-auditlogging-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-auditlogging-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platform-api-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-platform-api-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3.23
    name: ibm-monitoring-grafana-operator
    namespace: "{{ .ServicesNs }}"
    packageName: ibm-monitoring-grafana-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3.23
    name: ibm-zen-operator
    namespace: "{{ .ServicesNs }}"
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3.23
    name: ibm-zen-cpp-operator
    namespace: "{{ .CPFSNs }}"
    packageName: zen-cpp-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
`

	CSV4OpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: {{ .ExcludedCatalog }}
    status-monitored-services: {{ .StatusMonitoredServices }}
spec:
  operators:
  - name: ibm-usage-metering-operator
    namespace: "{{ .CPFSNs }}"
    channel: v1.0
    packageName: ibm-usage-metering-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-user-management-operator
    namespace: "{{ .CPFSNs }}"
    channel: v1.0
    packageName: ibm-user-management-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-websphere-liberty
    namespace: "{{ .CPFSNs }}"
    channel: v1.3
    packageName: ibm-websphere-liberty
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-redis-cp-operator
    namespace: "{{ .CPFSNs }}"
    channel: v1.2
    packageName: ibm-redis-cp
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator
    namespace: "{{ .CPFSNs }}"
    channel: v4.13
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-mongodb-operator
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    installMode: no-op
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3
    name: ibm-events-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-events-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v5.1
    name: ibm-events-operator-v5.1
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-events-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v5.2
    name: ibm-events-operator-v5.2
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-events-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator
    namespace: "{{ .CPFSNs }}"
    channel: v6.2
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator
    namespace: "{{ .CPFSNs }}"
    channel: v4.10
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: stable
    name: internal-use-only-edb
    namespace: "{{ .CPFSNs }}"
    packageName: cloud-native-postgresql
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: stable
    name: cloud-native-postgresql
    namespace: "{{ .CPFSNs }}"
    packageName: cloud-native-postgresql
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    operatorConfig: cloud-native-postgresql-operator-config
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: stable-v1.22
    fallbackChannels:
      - stable
    name: cloud-native-postgresql-v1.22
    namespace: "{{ .CPFSNs }}"
    packageName: cloud-native-postgresql
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    operatorConfig: cloud-native-postgresql-operator-config
    configName: cloud-native-postgresql
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: stable-v1.25
    fallbackChannels:
      - stable-v1.22
      - stable
    name: cloud-native-postgresql-v1.25
    namespace: "{{ .CPFSNs }}"
    packageName: cloud-native-postgresql
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    operatorConfig: cloud-native-postgresql-operator-config
    configName: cloud-native-postgresql
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: alpha
    name: ibm-user-data-services-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-user-data-services-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3
    name: ibm-bts-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-bts-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3.34
    name: ibm-bts-operator-v3.34
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-bts-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3.35
    name: ibm-bts-operator-v3.35
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-bts-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v1.3
    name: ibm-automation-flink
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-automation-flink
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v1.3
    name: ibm-automation-elastic
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-automation-elastic
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v1.1
    name: ibm-elasticsearch-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-elasticsearch-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode}}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v2.0
    name: ibm-opencontent-flink
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-opencontent-flink
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v1.1
    name: ibm-opensearch-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-opensearch-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode}}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
`
)

const (
	CSV3SaasOpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: {{ .ExcludedCatalog }}
    status-monitored-services: {{ .StatusMonitoredServices }}
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-mongodb-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-cert-manager-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-iam-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-management-ingress-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-ingress-nginx-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3.23
    name: ibm-zen-operator
    namespace: "{{ .ServicesNs }}"
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  `
)

const CSV4OpCon = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: ibm-usage-metering-operator
    spec:
      ibmUsageMetering: {}
  - name: ibm-licensing-operator
    spec:
      operandBindInfo: {}
  - name: ibm-mongodb-operator
    spec:
      mongoDB: {}
      operandRequest: {}
  - name: ibm-im-mongodb-operator
    spec:
      mongoDB: {}
      operandRequest: {}
  - name: ibm-im-operator
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      operandBindInfo:  
        operand: ibm-im-operator
  - name: ibm-iam-operator
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      oidcclientwatcher: {}
      pap: {}
      policycontroller: {}
      policydecision: {}
      secretwatcher: {}
      securityonboarding: {}
      operandBindInfo: {}
      operandRequest: {}
  - name: ibm-healthcheck-operator
    spec:
      healthService: {}
      mustgatherService: {}
      mustgatherConfig: {}
  - name: ibm-commonui-operator
    spec:
      commonWebUI: {}
      switcheritem: {}
      operandRequest: {}
      navconfiguration: {}
  - name: ibm-idp-config-ui-operator
    spec:
      commonWebUI: {}
      switcheritem: {}
      navconfiguration: {}
  - name: ibm-cert-manager-operator
    spec:
      certManager: {}
  - name: ibm-management-ingress-operator
    spec:
      managementIngress: {}
      operandBindInfo: {}
      operandRequest: {}
  - name: ibm-ingress-nginx-operator
    spec:
      nginxIngress: {}
  - name: ibm-auditlogging-operator
    spec:
      auditLogging: {}
      operandBindInfo: {}
      operandRequest: {}
  - name: ibm-platform-api-operator
    spec:
      platformApi: {}
      operandRequest: {}
  - name: ibm-monitoring-grafana-operator
    spec:
      grafana: {}
      operandRequest: {}
  - name: ibm-user-data-services-operator
    spec:
      operandBindInfo: {}
      operandRequest: {}
  - name: ibm-bts-operator
    spec:
      operandRequest:
        requests:
          - operands:
              - name: ibm-im-operator
            registry: common-service
  - name: ibm-bts-operator-v3.34
    spec:
      operandRequest:
        requests:
          - operands:
              - name: ibm-im-operator
            registry: common-service
  - name: ibm-bts-operator-v3.35
    spec:
      operandRequest:
        requests:
          - operands:
              - name: ibm-im-operator
            registry: common-service
  - name: ibm-zen-operator
    resources:
      - apiVersion: apps/v1
        force: true
        kind: Deployment
        labels:
          operator.ibm.com/opreq-control: 'true'
        name: meta-api-deploy
        namespace: "{{ .ServicesNs }}"
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator
    spec:
      operandBindInfo: {}
`

const ODLMSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: operand-deployment-lifecycle-manager-app
  namespace: "{{ .CPFSNs }}"
spec:
  channel: v4.5
  installPlanApproval: {{ .ApprovalMode }}
  name: ibm-odlm
  source: {{ .ODLMCatalogSourceName }}
  sourceNamespace: "{{ .ODLMCatalogSourceNs }}"
`

// ConcatenateRegistries concatenate the two YAML strings and return the new YAML string
func ConcatenateRegistries(baseRegistryTemplate string, insertedRegistryTemplateList []string, data interface{}, cppdata map[string]string) (string, error) {
	baseRegistry := odlm.OperandRegistry{}
	var template []byte
	var err error

	// Unmarshal first OprandRegistry
	if template, err = applyTemplate(baseRegistryTemplate, data); err != nil {
		return "", err
	}
	if err := utilyaml.Unmarshal(template, &baseRegistry); err != nil {
		return "", fmt.Errorf("failed to fetch data of OprandRegistry %v: %v", baseRegistry, err)
	}

	var newOperators []odlm.Operator
	for _, registryTemplate := range insertedRegistryTemplateList {
		insertedRegistry := odlm.OperandRegistry{}

		if template, err = applyTemplate(registryTemplate, data); err != nil {
			return "", err
		}
		if err := utilyaml.Unmarshal(template, &insertedRegistry); err != nil {
			return "", fmt.Errorf("failed to fetch data of OprandRegistry %v/%v: %v", insertedRegistry.Namespace, insertedRegistry.Name, err)
		}

		newOperators = append(newOperators, insertedRegistry.Spec.Operators...)
	}
	// Add new operators to baseRegistry
	baseRegistry.Spec.Operators = append(baseRegistry.Spec.Operators, newOperators...)

	// Update default and fallback channels with ConfigMap data
	operatorNames := []string{"keycloak-operator"} // List of operators to process
	processdDynamicChannels(&baseRegistry, cppdata, operatorNames)

	opregBytes, err := utilyaml.Marshal(baseRegistry)
	if err != nil {
		return "", err
	}

	return string(opregBytes), nil
}

// ConcatenateConfigs concatenate the two YAML strings and return the new YAML string
func ConcatenateConfigs(baseConfigTemplate string, insertedConfigTemplateList []string, data interface{}) (string, error) {
	baseConfig := odlm.OperandConfig{}
	var template []byte
	var err error

	// unmarshal first OprandConfig
	if template, err = applyTemplate(baseConfigTemplate, data); err != nil {
		return "", err
	}
	if err := utilyaml.Unmarshal(template, &baseConfig); err != nil {
		return "", fmt.Errorf("failed to fetch data of OprandConfig %v: %v", baseConfig, err)
	}

	var newServices []odlm.ConfigService
	for _, configTemplate := range insertedConfigTemplateList {
		insertedConfig := odlm.OperandConfig{}
		if template, err = applyTemplate(configTemplate, data); err != nil {
			return "", err
		}
		if err := utilyaml.Unmarshal(template, &insertedConfig); err != nil {
			return "", fmt.Errorf("failed to fetch data of OprandConfig %v/%v: %v", insertedConfig.Namespace, insertedConfig.Name, err)
		}

		newServices = append(newServices, insertedConfig.Spec.Services...)
	}
	// Add new services to baseConfig
	baseConfig.Spec.Services = append(baseConfig.Spec.Services, newServices...)

	opconBytes, err := utilyaml.Marshal(baseConfig)
	if err != nil {
		return "", err
	}

	return string(opconBytes), nil
}

func applyTemplate(objectTemplate string, data interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	t := template.Must(template.New("newTemplate").Parse(objectTemplate))
	if err := t.Execute(&buffer, data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// processFallbackChannels updates operator entries with dynamic fallback channels based on ibm-cpp-config data
func processdDynamicChannels(registry *odlm.OperandRegistry, configMapData map[string]string, operatorNames []string) {
	for i, operator := range registry.Spec.Operators {
		// Check if this operator is in our list of operators to process
		for _, opName := range operatorNames {
			if operator.Name == opName {

				channelList, exists := DefaultChannels[operator.Name]
				if !exists || len(channelList) == 0 {
					continue
				}
				highestVersion := channelList[0]

				var currentChannel string
				if operator.Name == "keycloak-operator" {
					// For keycloak, check the keycloak_preferred_channel in ConfigMap or use a default
					keycloakVersion, exists := configMapData["keycloak_preferred_channel"]
					if exists {
						currentChannel = keycloakVersion
					} else {
						currentChannel = "stable-v24" // Default for keycloak
					}
				} else {
					continue
				}

				// If current channel is less than highest version in list
				if compareVersions(currentChannel, highestVersion) < 0 {
					registry.Spec.Operators[i].Channel = currentChannel
					// Get all versions less than current channel
					var fallbacks []string
					for _, channel := range channelList {
						if compareVersions(channel, currentChannel) < 0 {
							fallbacks = append(fallbacks, channel)
						}
					}

					// Sort fallbacks in descending order
					sort.Slice(fallbacks, func(i, j int) bool {
						return compareVersions(fallbacks[i], fallbacks[j]) > 0
					})

					// Update the operator's fallback channels
					registry.Spec.Operators[i].FallbackChannels = fallbacks
				} else {
					registry.Spec.Operators[i].Channel = highestVersion
					if len(channelList) > 1 {
						registry.Spec.Operators[i].FallbackChannels = channelList[1:]
					} else {
						registry.Spec.Operators[i].FallbackChannels = []string{}
					}
				}
				break
			}
		}
	}
}

// compareVersions compares version strings (e.g., "stable-v22" vs "stable-v24")
// Returns -1 if v1 < v2; 0 if v1 == v2; 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// Extract version numbers
	re := regexp.MustCompile(`v(\d+)`)
	v1Matches := re.FindStringSubmatch(v1)
	v2Matches := re.FindStringSubmatch(v2)

	if len(v1Matches) < 2 || len(v2Matches) < 2 {
		return strings.Compare(v1, v2)
	}

	v1Num, _ := strconv.Atoi(v1Matches[1])
	v2Num, _ := strconv.Atoi(v2Matches[1])

	if v1Num < v2Num {
		return -1
	} else if v1Num > v2Num {
		return 1
	}
	return 0
}
