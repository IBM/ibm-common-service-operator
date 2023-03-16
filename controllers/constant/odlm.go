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
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:
  - name: ibm-licensing-operator
    spec:
      operandBindInfo: {}
  - name: ibm-mongodb-operator
    spec:
      mongoDB: {}
      operandRequest: {}
  - name: ibm-im-operator
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      policydecision: {}
      operandBindInfo: {}
      operandRequest: 
        requests:
          - operands:
              - name: ibm-mongodb-operator
              - name: ibm-commonui-operator
            registry: common-service
  - name: ibm-im-operator-v4.0
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      policydecision: {}
      operandBindInfo: {}
      operandRequest:
        requests:
          - operands:
              - name: ibm-mongodb-operator
              - name: ibm-commonui-operator
            registry: common-service
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
                  image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:fd8339c382e1c5d69184d9c3f299a3da5c9a12a579e0db5e76e86d65be9190fd
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
                    cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:fd8339c382e1c5d69184d9c3f299a3da5c9a12a579e0db5e76e86d65be9190fd
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
    spec:
      operandBindInfo: {}
    resources:
      - apiVersion: batch/v1
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
        namespace: "{{ .OperatorNs }}"
  - name: ibm-platformui-operator
    spec:
      operandBindInfo: {}
    resources:
      - apiVersion: batch/v1
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
        namespace: "{{ .OperatorNs }}"
  - name: ibm-platformui-operator-v4.0
    spec:
      operandBindInfo: {}
    resources:
      - apiVersion: batch/v1
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
        namespace: "{{ .OperatorNs }}"
`

const CSV3OperandRegistry = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-mongodb-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-cert-manager-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-iam-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-im-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-healthcheck-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-healthcheck-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-commonui-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-management-ingress-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-ingress-nginx-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-auditlogging-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-auditlogging-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-platform-api-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-platform-api-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - channel: v3.23
    name: ibm-monitoring-grafana-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-monitoring-grafana-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - channel: v3
    name: ibm-events-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-events-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3.23
    name: ibm-zen-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-platformui-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: stable
    name: cloud-native-postgresql
    namespace: "{{ .CPFSNs }}"
    packageName: cloud-native-postgresql
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: alpha
    name: ibm-user-data-services-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-user-data-services-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: v3.23
    name: ibm-zen-cpp-operator
    namespace: "{{ .CPFSNs }}"
    packageName: zen-cpp-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: v3
    name: ibm-bts-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-bts-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: v1.3
    name: ibm-automation-flink
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-automation-flink
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: v1.3
    name: ibm-automation-elastic
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-automation-elastic
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
`

const CSV3SaasOperandConfig = `
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
  - name: ibm-licensing-operator
    spec:
      operandBindInfo: {}
  - name: ibm-mongodb-operator
    spec:
      mongoDB: {}
      operandRequest: {}
  - name: ibm-im-operator
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      policydecision: {}
      operandBindInfo:
        bindings: {}
      operandRequest:
        requests:
          - operands:
              - name: ibm-mongodb-operator
              - name: ibm-commonui-operator
            registry: common-service
  - name: ibm-im-operator-v4.0
    spec:
      authentication:
        config:
          onPremMultipleDeploy: {{ .OnPremMultiEnable }}
      policydecision: {}
      operandBindInfo: {}
      operandRequest:
        requests:
          - operands:
              - name: ibm-mongodb-operator
              - name: ibm-commonui-operator
            registry: common-service
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
    spec:
      operandBindInfo: {}
    resources:
      - apiVersion: batch/v1
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
        namespace: "{{ .OperatorNs }}"
  - name: ibm-platformui-operator
    spec:
      operandBindInfo: {}
    resources:
      - apiVersion: batch/v1
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
        namespace: "{{ .OperatorNs }}"
  - name: ibm-platformui-operator-v4.0
    spec:
      operandBindInfo: {}
    resources:
      - apiVersion: batch/v1
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
        namespace: "{{ .OperatorNs }}"
`

const CSV3SaasOperandRegistry = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-mongodb-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-cert-manager-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-iam-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-im-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-management-ingress-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-ingress-nginx-operator
    namespace: "{{ .CPFSNs }}"
    channel: v3.23
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - channel: v3
    name: ibm-events-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-events-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - channel: v3.23
    name: ibm-zen-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-platformui-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-platformui-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-zen-operator
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
  - channel: v1.3
    name: ibm-automation-flink
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-automation-flink
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - channel: v1.3
    name: ibm-automation-elastic
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-automation-elastic
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
`

const ODLMSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: operand-deployment-lifecycle-manager-app
  namespace: "{{ .CPFSNs }}"
spec:
  channel: {{ .Channel }}
  installPlanApproval: {{ .ApprovalMode }}
  name: ibm-odlm
  source: {{ .CatalogSourceName }}
  sourceNamespace: "{{ .CatalogSourceNs }}"
`
