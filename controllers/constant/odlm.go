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
  - name: ibm-apicatalog
    spec:
      apicatalogmanager: {}
      operandBindInfo: {}
      operandRequest: {}
  - name: cloud-native-postgresql
    resources:
      - apiVersion: batch/v1
        kind: Job
        name: create-postgres-license-config
        data:
          spec:
            activeDeadlineSeconds: 600
            backoffLimit: 5
            template:
              metadata:
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
                  image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:607231e7337608278220853bd0bb04002488a6e74d4418752b1720133daef036
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
                - command:
                  - bash
                  - '-c'
                  - >-
                    kubectl delete pods -l app.kubernetes.io/name=cloud-native-postgresql
                  image: >-
                    cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:607231e7337608278220853bd0bb04002488a6e74d4418752b1720133daef036
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
      - apiVersion: rbac.authorization.k8s.io/v1
        kind: Role
        name: edb-license-role
        data:
          rules:
          - apiGroups:
            - ""
            resources:
            - pods
            - secrets
            verbs:
            - create
            - update
            - patch
            - get
            - list
            - delete
      - apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        name: edb-license-rolebinding
        data:
          subjects:
          - kind: ServiceAccount
            name: edb-license-sa
          roleRef:
            kind: Role
            name: edb-license-role
            apiGroup: rbac.authorization.k8s.io
  - name: ibm-bts-operator
    spec:
      operandRequest: {}
  - name: ibm-zen-operator
    resources:
      - apiVersion: batch/v1
        data:
          spec:
            activeDeadlineSeconds: 600
            backoffLimit: 5
            template:
              metadata: null
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
                containers:
                  - command:
                      - bash
                      - '-c'
                      - bash /setup/pre-zen.sh
                    env:
                      - name: common_services_namespace
                        valueFrom:
                          fieldRef:
                            fieldPath: metadata.namespace
                    image: {{ .ZenOperatorImage }}
                    name: pre-zen-job
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
                restartPolicy: OnFailure
                securityContext:
                  runAsNonRoot: true
                serviceAccount: operand-deployment-lifecycle-manager
                serviceAccountName: operand-deployment-lifecycle-manager
                terminationGracePeriodSeconds: 30
        force: true
        kind: Job
        name: pre-zen-operand-config-job 
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
  - channel: v2.0
    name: ibm-db2u-operator
    namespace: {{ .MasterNs }}
    packageName: db2u-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: stable
    name: cloud-native-postgresql
    namespace: {{ .MasterNs }}
    packageName: cloud-native-postgresql
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: {{ .Channel }}
    name: ibm-apicatalog
    namespace: {{ .MasterNs }}
    packageName: ibm-apicatalog
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: alpha
    name: ibm-user-data-services-operator
    namespace: {{ .MasterNs }}
    packageName: ibm-user-data-services-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: {{ .Channel }}
    name: ibm-zen-cpp-operator
    namespace: {{ .MasterNs }}
    packageName: zen-cpp-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: {{ .Channel }}
    name: ibm-bts-operator
    namespace: {{ .MasterNs }}
    packageName: ibm-bts-operator
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
  - name: ibm-monitoring-grafana-operator
    spec:
      grafana: {}
      operandRequest: {}
  - name: ibm-bts-operator
    spec:
      operandRequest: {}
      - name: ibm-zen-operator
      resources:
        - apiVersion: batch/v1
          data:
            spec:
              activeDeadlineSeconds: 600
              backoffLimit: 5
              template:
                metadata: null
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
                  containers:
                    - command:
                        - bash
                        - '-c'
                        - bash /setup/pre-zen.sh
                      env:
                        - name: common_services_namespace
                          valueFrom:
                            fieldRef:
                              fieldPath: metadata.namespace
                      image: {{ .ZenOperatorImage }}
                      name: pre-zen-job
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
                  restartPolicy: OnFailure
                  securityContext:
                    runAsNonRoot: true
                  serviceAccount: operand-deployment-lifecycle-manager
                  serviceAccountName: operand-deployment-lifecycle-manager
                  terminationGracePeriodSeconds: 30
          force: true
          kind: Job
          name: pre-zen-operand-config-job 
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
  - channel: {{ .Channel }}
    name: ibm-bts-operator
    namespace: {{ .MasterNs }}
    packageName: ibm-bts-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
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
