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

package constant

const CSV3OperandConfig = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: {{ .MasterNs }}
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: ibm-licensing-operator
    spec:
      IBMLicensing:
        datasource: datacollector
      operandBindInfo: {}
  - name: ibm-mongodb-operator
    spec:
      mongoDB: {}
      operandRequest: {}
  - name: ibm-cert-manager-operator
    spec:
      certManager: {}
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
      operandBindInfo:
        bindings:
          protected-zen-serviceid:
            secret: zen-serviceid-apikey-secret
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
      operandBindInfo: {}
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
  - name: ibm-monitoring-exporters-operator
    spec:
      exporter: {}
      operandRequest: {}
  - name: ibm-monitoring-prometheusext-operator
    spec:
      prometheusExt: {}
      operandRequest: {}
  - name: ibm-monitoring-grafana-operator
    spec:
      grafana: {}
      operandRequest: {}
  - name: user-data-services-operator
    spec:
      AnalyticsProxy: {}
      operandBindInfo: {}
      operandRequest: {}
`

const CSV3OperandRegistry = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: {{ .MasterNs }}
  annotations:
    version: {{ .Version }}
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: {{ .ControlNs }}
    channel: {{ .Channel }}
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-mongodb-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-cert-manager-operator
    namespace: {{ .ControlNs }}
    channel: {{ .Channel }}
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-iam-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-healthcheck-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-healthcheck-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-commonui-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-management-ingress-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-ingress-nginx-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-auditlogging-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-auditlogging-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-platform-api-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-platform-api-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-monitoring-exporters-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-monitoring-exporters-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-monitoring-prometheusext-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-monitoring-prometheusext-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - channel: {{ .Channel }}
    name: ibm-monitoring-grafana-operator
    namespace: {{ .MasterNs }}
    packageName: ibm-monitoring-grafana-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - channel: {{ .Channel }}
    name: ibm-events-operator
    namespace: {{ .MasterNs }}
    packageName: ibm-events-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - channel: stable
    name: redhat-marketplace-operator
    namespace: openshift-redhat-marketplace
    packageName: redhat-marketplace-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: certified-operators
    sourceNamespace: {{ .CatalogSourceNs }}
  - channel: {{ .Channel }}
    name: ibm-zen-operator
    namespace: {{ .MasterNs }}
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - channel: v1.1
    name: ibm-db2u-operator
    namespace: {{ .MasterNs }}
    packageName: db2u-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: {{ .Channel }}
    name: user-data-services-operator
    namespace: {{ .MasterNs }}
    packageName: user-data-services-operator-certified
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: stable
    name: cloud-native-postgresql
    namespace: {{ .MasterNs }}
    packageName: cloud-native-postgresql
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
`

const CSV3SaasOperandConfig = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: {{ .MasterNs }}
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: ibm-licensing-operator
    spec:
      IBMLicensing:
        datasource: datacollector
        routeEnabled: false
        logLevel: VERBOSE
      operandBindInfo: {}
  - name: ibm-mongodb-operator
    spec:
      mongoDB: {}
      operandRequest: {}
  - name: ibm-cert-manager-operator
    spec:
      certManager: {}
  - name: ibm-iam-operator
    spec:
      authentication:
        config:
          ibmCloudSaas: true
      oidcclientwatcher: {}
      pap: {}
      policycontroller: {}
      policydecision: {}
      secretwatcher: {}
      securityonboarding: {}
      operandBindInfo:
        bindings:
          protected-zen-serviceid:
            secret: zen-serviceid-apikey-secret
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
      operandBindInfo: {}
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
  - name: ibm-monitoring-exporters-operator
    spec:
      exporter: {}
      operandRequest: {}
  - name: ibm-monitoring-prometheusext-operator
    spec:
      prometheusExt: {}
      operandRequest: {}
  - name: ibm-monitoring-grafana-operator
    spec:
      grafana: {}
      operandRequest: {}
`

const CSV3SaasOperandRegistry = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: {{ .MasterNs }}
  annotations:
    version: {{ .Version }}
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: {{ .ControlNs }}
    channel: {{ .Channel }}
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-mongodb-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-cert-manager-operator
    namespace: {{ .ControlNs }}
    channel: {{ .Channel }}
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-iam-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-management-ingress-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - name: ibm-ingress-nginx-operator
    namespace: {{ .MasterNs }}
    channel: {{ .Channel }}
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - channel: {{ .Channel }}
    name: ibm-events-operator
    namespace: {{ .MasterNs }}
    packageName: ibm-events-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
  - channel: {{ .Channel }}
    name: ibm-zen-operator
    namespace: {{ .MasterNs }}
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: {{ .CatalogSourceNs }}
`

const ODLMClusterSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: operand-deployment-lifecycle-manager-app
  namespace: {{ .MasterNs }}
spec:
  channel: {{ .Channel }}
  installPlanApproval: Automatic
  name: ibm-odlm
  source: {{ .CatalogSourceName }}
  sourceNamespace: {{ .CatalogSourceNs }}
`

const ODLMNamespacedSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: operand-deployment-lifecycle-manager-app
  namespace: {{ .MasterNs }}
spec:
  channel: {{ .Channel }}
  installPlanApproval: Automatic
  name: ibm-odlm
  source: {{ .CatalogSourceName }}
  sourceNamespace: {{ .CatalogSourceNs }}
  config:
    env:
    - name: INSTALL_SCOPE
      value: namespaced
    - name: ISOLATED_MODE
      value: "{{ .IsolatedModeEnable }}"
`
