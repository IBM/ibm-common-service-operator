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
	"text/template"

	utilyaml "github.com/ghodss/yaml"

	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
)

var (
	CSV3OperandRegistry     string
	CSV3SaasOperandRegistry string
	CSV3OperandConfig       string
	CSV3SaasOperandConfig   string
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - channel: stable-v22
    installPlanApproval: {{ .ApprovalMode }}
    name: keycloak-operator
    namespace: "{{ .ServicesNs }}"
    packageName: rhbk-operator
    scope: public
  - channel: stable
    installPlanApproval: {{ .ApprovalMode }}
    name: edb-keycloak
    namespace: "{{ .CPFSNs }}"
    packageName: cloud-native-postgresql
    scope: public
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
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v4.1
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v4.2
    spec:
      operandBindInfo: {}
  - name: ibm-platformui-operator-v4.3
    spec:
      operandBindInfo: {}
`
)

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
            description: Binding information that should be accessible to Keycloak adopters
            operand: keycloak-operator
            registry: common-service
            registryNamespace: {{ .ServicesNs }}
        force: true
        kind: OperandBindInfo
        name: keycloak-bindinfo
      - apiVersion: cert-manager.io/v1
        kind: Certificate
        force: true
        name: cs-keycloak-tls-cert
        data:
          spec:
            commonName: cs-keycloak-service
            dnsNames:
                - cs-keycloak-service
                - cs-keycloak-service.{{ .ServicesNs }}
                - cs-keycloak-service.{{ .ServicesNs }}.svc
                - cs-keycloak-service.{{ .ServicesNs }}.svc.cluster.local
            issuerRef:
                kind: Issuer
                name: cs-ca-issuer
            secretName: cs-keycloak-tls-secret
      - apiVersion: v1
        kind: ConfigMap
        force: true
        name: keycloak-bindinfo
        namespace: {{ .OperatorNs }}
        data:
          data:
            keycloak-bindinfo.yaml: |
              route:
                name: keycloak
                configmap: keycloak-bindinfo-cs-keycloak-route
                data:
                  HOSTNAME: https://+.spec.host
                  TERMINATION: .spec.tls.termination
                  BACKEND_SERVICE: .spec.to.name
              service:
                name: cs-keycloak-service
                configmap: keycloak-bindinfo-cs-keycloak-service
                data:
                  PORT: .spec.ports[0].port
                  CLUSTER_IP: .spec.clusterIP
                  SERVICE_NAME: .metadata.name
                  SERVICE_NAMESPACE: .metadata.namespace
                  SERVICE_ENDPOINT: https://+.metadata.name+.+.metadata.namespace+.+svc:+.spec.ports[0].port
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
              name: cs-keycloak-service
            wildcardPolicy: None
        force: true
        kind: Route
        name: keycloak
      - apiVersion: v1
        kind: ConfigMap
        name: keycloak-bindinfo-script
        namespace: {{ .OperatorNs }}
        data:  
          data:
            keycloak-bindinfo-script.sh: |
              #!/bin/bash
          
              KIND_LIST="route,service"
              config_yaml=$(cat /bindinfo-data/keycloak-bindinfo.yaml)
              resource_namespace=$(oc get commonservice common-service -n $OPERATOR_NAMESPACE -o jsonpath='{.spec.servicesNamespace}')
          
              function parse_schema() {
                  local kind="$1"
                  local resource_name="$2"
                  local resource_namespace="$3"
                  local schema="$4"
          
                  result=""
                  IFS='+'
          
                  read -ra items <<< "$schema"
          
                  for item in "${items[@]}"; do
                      
                      # if the item start with dot, and length is greater than 1, then it is a jsonpath
                      if [[ "$item" == .* ]] && [[ ${#item} -gt 1 ]]; then
                          echo $item is jsonpath
                          value=""
                          parse_jsonpath $kind $resource_name $resource_namespace $item value
                      else
                          echo $item is not jsonpath
                          value=$item
                      fi
                      result+=$value
                  done
          
                  eval "$5=$result"
          
              }
          
              function parse_jsonpath() {
                  local kind="$1"
                  local resource_name="$2"
                  local resource_namespace="$3"
                  local jsonpath="$4"
                  
                  value=$(oc get $kind "$resource_name" -n "$resource_namespace" -o yaml | yq eval "$jsonpath" -)
          
                  eval "$5=$value"
              }
          
              function copy_secret() {
                  local secret_name="$1"
                  local source_namespace="$2"
                  local target_namespace="$3"
                  local new_secret_name="$4"
          
                  oc get secret "$secret_name" -n "$source_namespace" -o yaml | sed "s/$secret_name/$new_secret_name/g" | oc apply -n "$target_namespace" -f -
                  echo "Secret $secret_name copied from namespace $source_namespace to namespace $target_namespace"
              }
          
              function create_configmap() {
                  local kind="$1"
                  local resource_name="$2"
                  local resource_namespace="$3"
                  local configmap_name="$4"
                  local config_namespace="$5"
                  local data_fields="$6"
          
                  cat <<EOF >/tmp/configmap.yaml
              apiVersion: v1
              kind: ConfigMap
              metadata:
                  name: $configmap_name
                  namespace: $config_namespace
              data:
              EOF
          
                  while read -r field; do
                      key=$(echo "$field" | awk -F ': ' '{print $1}')
                      complete_path=$(echo "$field" | awk -F ': ' '{print $2}')
                      echo $complete_path
                      value=""
                      parse_schema "$kind" "$resource_name" "$resource_namespace" "$complete_path" value
                      echo "  $key: '$value'" >> /tmp/configmap.yaml
                  done <<< "$data_fields"
          
                  oc apply -f /tmp/configmap.yaml
          
                  echo "ConfigMap $resource_name created in namespace $config_namespace"
              }
          
              while true; do
                  if [ -z "$WATCH_NAMESPACE" ] || [ "$WATCH_NAMESPACE" == "''" ]; then
                      config_namespace_list=$(oc get OperandRequest --all-namespaces -o custom-columns='NAMESPACE:.metadata.namespace' --no-headers | tr '\n' ',' | sed 's/,$//')
                  else
                      config_namespace_list=$WATCH_NAMESPACE
                  fi
                  while IFS=',' read -ra kind; do
                      for i in "${kind[@]}"; do
                          echo $i
                          resource_name=$(echo "$config_yaml" | yq eval ".$i.name" -)
                          configmap_name=$(echo "$config_yaml" | yq eval ".$i.configmap" -)
                          data_fields=$(echo "$config_yaml" | yq eval ".$i.data" -)
          
                          # check if this resource already exists
                          oc get $i "$resource_name" -n "$resource_namespace" >/dev/null 2>&1
                          if [ $? -ne 0 ]; then
                              echo "Resource $i/$resource_name does not exist in namespace $resource_namespace, skipping..."
                              continue
                          fi
                          
                          while IFS=',' read -ra config_namespace; do
                              for j in "${config_namespace[@]}"; do
                                  create_configmap "$i" "$resource_name" "$resource_namespace" "$configmap_name" "$j" "$data_fields"
                              done
                          done <<< "$config_namespace_list"
                          
                      done
                  done <<< "$KIND_LIST"
                  sleep 120
              done
      - apiVersion: apps/v1
        kind: Deployment
        name: keycloak-bindinfo-deployment
        namespace: {{ .OperatorNs }}
        force: false
        data:
          spec:
            replicas: 1
            selector:
              matchLabels:
                app: keycloak-bindinfo
            template:
              metadata:
                labels:
                  app: keycloak-bindinfo
              spec:
                containers:
                - name: keycloak-bindinfo-container
                  image: icr.io/cpopen/cpfs/cpfs-utils:latest
                  command: ["/bin/bash"]
                  args: ["-c", "/bindinfo-script/keycloak-bindinfo-script.sh"]
                  volumeMounts:
                  - name: keycloak-bindinfo-data
                    mountPath: /bindinfo-data
                  - name: keycloak-bindinfo-script
                    mountPath: /bindinfo-script
                  env:
                    - name: OPERATOR_NAMESPACE
                      valueFrom:
                        fieldRef:
                          apiVersion: v1
                          fieldPath: metadata.namespace
                    - name: WATCH_NAMESPACE
                      valueFrom:
                          configMapKeyRef:
                            name: namespace-scope
                            key: namespaces
                            optional: true
                  securityContext:
                    runAsNonRoot: true
                    seccompProfile:
                      type: RuntimeDefault
                    capabilities:
                      drop:
                        - ALL
                    allowPrivilegeEscalation: false
                volumes:
                - name: keycloak-bindinfo-data
                  configMap:
                    name: keycloak-bindinfo
                    items:
                    - key: keycloak-bindinfo.yaml
                      path: keycloak-bindinfo.yaml
                - name: keycloak-bindinfo-script
                  configMap:
                    name: keycloak-bindinfo-script
                    defaultMode: 0777
                    items:
                    - key: keycloak-bindinfo-script.sh
                      path: keycloak-bindinfo-script.sh
                securityContext:
                  runAsNonRoot: true
                  seccompProfile:
                    type: RuntimeDefault
                serviceAccountName: operand-deployment-lifecycle-manager
      - apiVersion: k8s.keycloak.org/v2alpha1
        data:
          spec:
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
                  objectRef:
                    apiVersion: route.openshift.io/v1
                    kind: Route
                    name: keycloak
                    path: spec.host
                  required: true
            http:
              tlsSecret: cs-keycloak-tls-secret
            ingress:
              enabled: false
            instances: 1
            unsupported:
              podTemplate:
                spec:
                  containers:
                    - resources:
                        limits:
                          cpu: 1000m
                          memory: 1Gi
                        requests:
                          cpu: 1000m
                          memory: 1Gi
        force: true
        kind: Keycloak
        name: cs-keycloak
      - apiVersion: k8s.keycloak.org/v2alpha1
        kind: KeycloakRealmImport
        name: cs-cloudpak-realm
        force: false
        data:
          spec:
            keycloakCRName: cs-keycloak
            realm:
              displayName: cloudpak
              enabled: true
              id: cloudpak
              realm: cloudpak
  - name: edb-keycloak
    resources:
      - apiVersion: batch/v1
        kind: Job
        force: true
        name: create-postgres-license-config
        namespace: "{{ .OperatorNs }}"
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
                        defaultValue: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:05f30f2117ff6e0e853487f17785024f6bb226f3631425eaf1498b9d3b753345
                        configMapKeyRef:
                          name: cloud-native-postgresql-image-list
                          key: edb-postgres-license-provider-image
                          namespace: {{ .OperatorNs }}
                  name: edb-license
                  resources:
                    limits:
                      cpu: 200m
                      memory: 768Mi
                    requests:
                      cpu: 200m
                      memory: 512Mi
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
                  image:
                    templatingValueFrom:
                      default:
                        required: true
                        defaultValue: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:05f30f2117ff6e0e853487f17785024f6bb226f3631425eaf1498b9d3b753345
                        configMapKeyRef:
                          name: cloud-native-postgresql-image-list
                          key: edb-postgres-license-provider-image
                          namespace: {{ .OperatorNs }}
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
            - watch
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
            bootstrap:
              initdb:
                database: keycloak
                owner: app
            imageName:
              templatingValueFrom:
                default:
                  required: true
                  defaultValue: icr.io/cpopen/edb/postgresql:14.9@sha256:90136074adcbafb5033668b07fe1efea9addf0168fa83b0c8a6984536fc22264
                  configMapKeyRef:
                    name: cloud-native-postgresql-image-list
                    key: ibm-postgresql-14-operand-image
                    namespace: {{ .OperatorNs }}
                configMapKeyRef:
                    name: edb-keycloak-operand-image
                    key: ibm-cpp-config
            imagePullSecrets:
              - name: ibm-entitlement-key
            instances: 1
            resources:
              limits:
                cpu: 1000m
                memory: 1Gi
              requests:
                cpu: 1000m
                memory: 1Gi
            logLevel: info
            primaryUpdateStrategy: unsupervised
            storage:
              size: 1Gi
            walStorage:
              size: 1Gi
        force: true
        kind: Cluster
        name: keycloak-edb-cluster
`
)

const (
	CSV2OpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: "{{ .Version }}"
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-mongodb-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-cert-manager-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-iam-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-healthcheck-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-healthcheck-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-commonui-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-management-ingress-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-ingress-nginx-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-auditlogging-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-auditlogging-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-platform-api-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-platform-api-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - channel: v3.23
    name: ibm-monitoring-grafana-operator
    namespace: "{{ .ServicesNs }}"
    packageName: ibm-monitoring-grafana-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - channel: v3.23
    name: ibm-zen-operator
    namespace: "{{ .ServicesNs }}"
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - channel: v3.23
    name: ibm-zen-cpp-operator
    namespace: "{{ .CPFSNs }}"
    packageName: zen-cpp-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
`

	CSV3OpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-im-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-im-mongodb-operator
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
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
  - name: ibm-platformui-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
  - name: ibm-idp-config-ui-operator
    namespace: "{{ .CPFSNs }}"
    channel: {{ .Channel }}
    packageName: ibm-commonui-operator-app
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
)

const (
	CSV2SaasOpReg = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRegistry
metadata:
  name: common-service
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,redhat-operators,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-mongodb-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-cert-manager-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-iam-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-management-ingress-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - name: ibm-ingress-nginx-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  - channel: v3.23
    name: ibm-zen-operator
    namespace: "{{ .ServicesNs }}"
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    sourceName: {{ .CatalogSourceName }}
    sourceNamespace: "{{ .CatalogSourceNs }}"
    installMode: no-op
  `
)

const CSV3OpCon = `
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
      operandRequest: 
        requests:
          - operands:
              - name: ibm-im-mongodb-operator
              - name: ibm-idp-config-ui-operator
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
  - name: ibm-idp-config-ui-operator
    spec:
      commonWebUI: {}
      switcheritem: {}
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
        namespace: "{{ .OperatorNs }}"
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
                        defaultValue: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:05f30f2117ff6e0e853487f17785024f6bb226f3631425eaf1498b9d3b753345
                        configMapKeyRef:
                          name: cloud-native-postgresql-image-list
                          key: edb-postgres-license-provider-image
                          namespace: {{ .OperatorNs }}
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
                  image:
                    templatingValueFrom:
                      default:
                        required: true
                        defaultValue: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:05f30f2117ff6e0e853487f17785024f6bb226f3631425eaf1498b9d3b753345
                        configMapKeyRef:
                          name: cloud-native-postgresql-image-list
                          key: edb-postgres-license-provider-image
                          namespace: {{ .OperatorNs }}
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
            - watch
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
  - name: ibm-bts-operator
    spec:
      operandRequest:
        requests:
          - operands:
              - name: ibm-im-operator
            registry: common-service
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
`

const ODLMSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: operand-deployment-lifecycle-manager-app
  namespace: "{{ .CPFSNs }}"
spec:
  channel: v4.2
  installPlanApproval: {{ .ApprovalMode }}
  name: ibm-odlm
  source: {{ .CatalogSourceName }}
  sourceNamespace: "{{ .CatalogSourceNs }}"
`

// ConcatenateRegistries concatenate the two YAML strings and return the new YAML string
func ConcatenateRegistries(baseRegistryTemplate string, insertedRegistryTemplateList []string, data interface{}) (string, error) {
	baseRegistry := odlm.OperandRegistry{}
	var template []byte
	var err error

	// unmarshal first OprandRegistry
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
	// add new operators to baseRegistry
	baseRegistry.Spec.Operators = append(baseRegistry.Spec.Operators, newOperators...)

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

	// unmarshal first OprandCongif
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
	// add new services to baseConfig
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
