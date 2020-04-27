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

package bootstrap

const namespace = `
apiVersion: v1
kind: Namespace
metadata:
  name: ibm-common-services
`

const subscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: operand-deployment-lifecycle-manager-app
  namespace: openshift-operators
  annotations:
    version: "1"
spec:
  channel: dev
  installPlanApproval: Automatic
  name: operand-deployment-lifecycle-manager-app
  source: opencloud-operators
  sourceNamespace: openshift-marketplace
`

const operandRegistry = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: ibm-common-services
  annotations:
    version: "1"
spec:
  operators:
  - name: ibm-metering-operator
    namespace: ibm-common-services
    channel: stable-v1
    packageName: ibm-metering-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: The service used to meter workloads in a kubernetes cluster
  - name: ibm-licensing-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-licensing-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: The service used to management the license in a kubernetes cluster
  - name: ibm-mongodb-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-mongodb-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: The service used to create mongodb in a kubernetes cluster
  - name: ibm-cert-manager-operator
    namespace: ibm-common-services
    channel: stable-v1
    packageName: ibm-cert-manager-operator
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of cert-manager service.
  - name: ibm-iam-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-iam-operator
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of iam service.
  - name: ibm-healthcheck-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-healthcheck-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of health check service.
  - name: ibm-commonui-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-commonui-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: The service that services the login page, common header, LDAP, and Team resources pages
  - name: ibm-management-ingress-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-management-ingress-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of management ingress service.
  - name: ibm-ingress-nginx-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-ingress-nginx-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of ingress nginx service.
  - name: ibm-auditlogging-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-auditlogging-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of auditlogging service.
  - name: ibm-catalog-ui-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-catalog-ui-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of catalog UI service.
  - name: ibm-platform-api-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-platform-api-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of Platform API service.
  - name: ibm-helm-api-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-helm-api-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of Helm API service.
  - name: ibm-helm-repo-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-helm-repo-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator for managing deployment of Helm repository service.
  - name: ibm-monitoring-exporters-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-monitoring-exporters-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator to provision node-exporter, kube-state-metrics and collectd exporter with tls enabled.
  - name: ibm-monitoring-prometheusext-operator
    namespace: ibm-common-services
    channel: dev
    packageName: ibm-monitoring-prometheusext-operator-app
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
    description: Operator to deploy Prometheus and Alertmanager instances with RBAC enabled. It will also enable Multicloud monitoring.
  - channel: dev
    description: Operator to deploy Grafana instances with RBAC enabled.
    name: ibm-monitoring-grafana-operator
    namespace: ibm-common-services
    packageName: ibm-monitoring-grafana-operator-app
    scope: private
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
  - channel: dev
    description: Operator that installs and manages Elastic Stack logging service instances.
    name: ibm-elastic-stack-operator
    namespace: ibm-common-services
    packageName: ibm-elastic-stack-operator-app
    scope: private
    sourceName: opencloud-operators
    sourceNamespace: openshift-marketplace
`

const operandConfig = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandConfig
metadata:
  name: common-service
  namespace: ibm-common-services
  annotations:
    version: "1"
spec:
  services:
  - name: ibm-metering-operator
    spec:
      metering: {}
      meteringUI: {}
  - name: ibm-licensing-operator
    spec:
      IBMLicensing: {}
      OperandBindInfo: {}
      OperandRequest: {}
  - name: ibm-mongodb-operator
    spec:
      mongoDB: {}
  - name: ibm-cert-manager-operator
    spec:
      certManager: {}
      issuer: {}
      certificate: {}
      clusterIssuer: {}
  - name: ibm-iam-operator
    spec:
      authentication: {}
      oidcclientwatcher: {}
      pap: {}
      policycontroller: {}
      policydecision: {}
      secretwatcher: {}
      securityonboarding: {}
  - name: ibm-healthcheck-operator
    spec:
      healthService: {}
  - name: ibm-commonui-operator
    spec:
      commonWebUI: {}
      legacyHeader: {}
  - name: ibm-management-ingress-operator
    spec:
      managementIngress: {}
  - name: ibm-ingress-nginx-operator
    spec:
      nginxIngress: {}
  - name: ibm-auditlogging-operator
    spec:
      auditLogging: {}
  - name: ibm-catalog-ui-operator
    spec:
      catalogUI: {}
  - name: ibm-platform-api-operator
    spec:
      platformApi: {}
  - name: ibm-helm-api-operator
    spec:
      helmApi: {}
  - name: ibm-helm-repo-operator
    spec:
      helmRepo: {}
  - name: ibm-monitoring-exporters-operator
    spec:
      exporter: {}
  - name: ibm-monitoring-prometheusext-operator
    spec:
      prometheusExt: {}
  - name: ibm-monitoring-grafana-operator
    spec:
      grafana: {}
  - name: ibm-elastic-stack-operator
    spec:
      elasticStack: {}
`
