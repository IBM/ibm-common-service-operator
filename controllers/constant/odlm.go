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
  namespace: placeholder
  annotations:
	version: "3.8.0"
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
	  authentication: {}
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

const CSV3OperandRegistry = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: placeholder
  annotations:
	version: "3.8.0"
spec:
  operators:
  - name: ibm-licensing-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-licensing-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-mongodb-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-mongodb-operator-app
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-cert-manager-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-cert-manager-operator
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-iam-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-iam-operator
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-healthcheck-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-healthcheck-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-commonui-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-commonui-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-management-ingress-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-management-ingress-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-ingress-nginx-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-ingress-nginx-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-auditlogging-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-auditlogging-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-platform-api-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-platform-api-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-monitoring-exporters-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-monitoring-exporters-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-monitoring-prometheusext-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-monitoring-prometheusext-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - channel: beta
	name: ibm-monitoring-grafana-operator
	namespace: placeholder
	packageName: ibm-monitoring-grafana-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - channel: beta
	name: ibm-events-operator
	namespace: placeholder
	packageName: ibm-events-operator
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - channel: stable
	name: redhat-marketplace-operator
	namespace: openshift-redhat-marketplace
	packageName: redhat-marketplace-operator
	scope: public
	sourceName: certified-operators
	sourceNamespace: openshift-marketplace
  - channel: beta
	name: ibm-zen-operator
	namespace: placeholder
	packageName: ibm-zen-operator
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - channel: v1.0
	name: ibm-db2u-operator
	namespace: placeholder
	packageName: db2u-operator
	scope: public
	sourceName: ibm-operator-catalog
	sourceNamespace: openshift-marketplace
`

const CSV3SaasOperandConfig = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: placeholder
  annotations:
	version: "3.8.0"
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
	  authentication: {}
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
  namespace: placeholder
  annotations:
	version: "3.8.0"
spec:
  operators:
  - name: ibm-licensing-operator
	namespace: controlns
	channel: beta
	packageName: ibm-licensing-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-mongodb-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-mongodb-operator-app
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-cert-manager-operator
	namespace: controlns
	channel: beta
	packageName: ibm-cert-manager-operator
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-iam-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-iam-operator
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-management-ingress-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-management-ingress-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - name: ibm-ingress-nginx-operator
	namespace: placeholder
	channel: beta
	packageName: ibm-ingress-nginx-operator-app
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - channel: beta
	name: ibm-events-operator
	namespace: placeholder
	packageName: ibm-events-operator
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
  - channel: beta
	name: ibm-zen-operator
	namespace: placeholder
	packageName: ibm-zen-operator
	scope: public
	sourceName: opencloud-operators
	sourceNamespace: openshift-marketplace
`

const ODLMClusterSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: operand-deployment-lifecycle-manager-app
  namespace: placeholder
  annotations:
	version: "3.8.0"
spec:
  channel: beta
  installPlanApproval: Automatic
  name: ibm-odlm
  source: opencloud-operators
  sourceNamespace: openshift-marketplace
`

const ODLMNamespacedSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: operand-deployment-lifecycle-manager-app
  namespace: placeholder
  annotations:
	version: "3.8.0"
spec:
  channel: beta
  installPlanApproval: Automatic
  name: ibm-odlm
  source: opencloud-operators
  sourceNamespace: openshift-marketplace
  config:
	env:
	- name: INSTALL_SCOPE
	  value: namespaced
	- name: ODLM_SCOPE
	  value: "odlmScopesholder"
`