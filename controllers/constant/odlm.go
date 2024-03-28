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
	CSV4OperandRegistry     string
	CSV4SaasOperandRegistry string
	CSV4OperandConfig       string
	CSV4SaasOperandConfig   string
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-im-mongodb-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: v4.0
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-im-mongodb-operator-v4.1
    namespace: "{{ .CPFSNs }}"
    channel: v4.1
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-im-mongodb-operator-v4.2
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-im-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: v4.0
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-im-operator-v4.1
    namespace: "{{ .CPFSNs }}"
    channel: v4.1
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-im-operator-v4.2
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-im-operator-v4.3
    namespace: "{{ .CPFSNs }}"
    channel: v4.3
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-im-operator-v4.4
    namespace: "{{ .CPFSNs }}"
    channel: v4.4
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-im-operator-v4.5
    namespace: "{{ .CPFSNs }}"
    channel: v4.5
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-idp-config-ui-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: v4.0
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-idp-config-ui-operator-v4.1
    namespace: "{{ .CPFSNs }}"
    channel: v4.1
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-idp-config-ui-operator-v4.2
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-idp-config-ui-operator-v4.3
    namespace: "{{ .CPFSNs }}"
    channel: v4.3
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-idp-config-ui-operator-v4.4
    namespace: "{{ .CPFSNs }}"
    channel: v4.4
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-platformui-operator-v4.0
    namespace: "{{ .CPFSNs }}"
    channel: v4.0
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-platformui-operator-v4.1
    namespace: "{{ .CPFSNs }}"
    channel: v4.1
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-platformui-operator-v4.2
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-platformui-operator-v4.3
    namespace: "{{ .CPFSNs }}"
    channel: v4.3
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-platformui-operator-v4.4
    namespace: "{{ .CPFSNs }}"
    channel: v4.4
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - channel: stable
    installPlanApproval: {{ .ApprovalMode }}
    name: common-service-postgresql
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
  - name: ibm-platformui-operator-v4.4
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
              public-cs-keycloak-route:
                configmap: cs-keycloak-route
              public-cs-keycloak-service:
                configmap: cs-keycloak-service
            description: Binding information that should be accessible to Keycloak adopters
            operand: keycloak-operator
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
              "/opt/keycloak/bin/kc.sh" "$@" --spi-truststore-file-file=${TRUSTSTORE_DIR}/keycloak-truststore.jks --spi-truststore-file-password=changeit --spi-truststore-file-hostname-verification-policy=WILDCARD
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
            features:
              enabled:
                - token-exchange
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
                    path: .spec.host
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
                    - command:
                        - /bin/sh
                        - /mnt/startup/cs-keycloak-entrypoint.sh
                      resources:
                        limits:
                          cpu: 1000m
                          memory: 1Gi
                          ephemeral-storage: 512Mi
                        requests:
                          cpu: 1000m
                          memory: 1Gi
                          ephemeral-storage: 256Mi
                      volumeMounts:
                        - mountPath: /mnt/truststore
                          name: truststore-volume
                        - mountPath: /mnt/startup
                          name: startup-volume
                        - mountPath: /mnt/trust-ca
                          name: trust-ca-volume
                        - mountPath: /opt/keycloak/providers
                          name: cs-keycloak-theme
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
      - apiVersion: operator.ibm.com/v1alpha1
        data:
          spec:
            requests:
              - operands:
                  - name: cloud-native-postgresql
                registry: common-service
                registryNamespace: {{ .ServicesNs }}
        force: true
        kind: OperandRequest
        name: postgresql-operator-request
      - apiVersion: postgresql.k8s.enterprisedb.io/v1
        data:
          spec:
            inheritedMetadata:
              annotations:
                backup.velero.io/backup-volumes: pgdata,pg-wal
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
                    name: ibm-cpp-config
                    key: edb-keycloak-operand-image
            imagePullSecrets:
              - name: ibm-entitlement-key
            instances: 1
            resources:
              limits:
                cpu: 200m
                memory: 512Mi
              requests:
                cpu: 200m
                memory: 512Mi
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
          k8s.enterprisedb.io/addons: ["velero"]
          k8s.enterprisedb.io/snapshotAllowColdBackupOnPrimary: enabled
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
                  - name: cloud-native-postgresql
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
              protected-cloudpak-db:
                secret: common-service-db-app
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
        data:
          spec:
            bootstrap:
              initdb:
                database: cloudpak
                owner: cpadmin
                dataChecksums: true
                postInitApplicationSQL:
                  - CREATE USER im_user
                  - CREATE DATABASE im OWNER im_user
                  - GRANT ALL PRIVILEGES ON DATABASE im TO im_user
                  - CREATE USER zen_user
                  - CREATE DATABASE zen OWNER zen_user
                  - GRANT ALL PRIVILEGES ON DATABASE zen TO zen_user
            affinity:
              topologyKey: topology.kubernetes.io/zone
            imageName:
              templatingValueFrom:
                default:
                  required: true
                  configMapKeyRef:
                    name: cloud-native-postgresql-image-list
                    key: ibm-postgresql-16-operand-image
                    namespace: {{ .OperatorNs }}
            imagePullSecrets:
              - name: ibm-entitlement-key
            instances: 1
            replicationSlots:
              highAvailability:
                enabled: true
            certificates:
              clientCASecret: cs-ca-certificate-secret
              replicationTLSSecret: common-service-db-replica-tls-secret
              serverCASecret: cs-ca-certificate-secret
              serverTLSSecret: common-service-db-tls-secret
            resources:
              limits:
                cpu: 200m
                memory: 512Mi
              requests:
                cpu: 200m
                memory: 512Mi
            primaryUpdateStrategy: unsupervised
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
                max_connections: "600"  
              pg_hba:
                - hostssl cloudpak cpadmin all cert
                - hostssl im im_user all cert
                - hostssl zen zen_user all cert
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-mongodb-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-cert-manager-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-iam-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-healthcheck-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-healthcheck-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-commonui-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-management-ingress-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-ingress-nginx-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-auditlogging-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-auditlogging-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-platform-api-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-platform-api-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - channel: v3.23
    name: ibm-monitoring-grafana-operator
    namespace: "{{ .ServicesNs }}"
    packageName: ibm-monitoring-grafana-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - channel: v3.23
    name: ibm-zen-operator
    namespace: "{{ .ServicesNs }}"
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - channel: v3.23
    name: ibm-zen-cpp-operator
    namespace: "{{ .CPFSNs }}"
    packageName: zen-cpp-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-im-operator
    namespace: "{{ .CPFSNs }}"
    channel: v4.5
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-im-mongodb-operator
    namespace: "{{ .CPFSNs }}"
    channel: v4.2
    installMode: no-op
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
  - channel: v3
    name: ibm-events-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-events-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-platformui-operator
    namespace: "{{ .CPFSNs }}"
    channel: v4.4
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
  - name: ibm-idp-config-ui-operator
    namespace: "{{ .CPFSNs }}"
    channel: v4.4
    packageName: ibm-commonui-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
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
  - channel: v1.1
    name: ibm-elasticsearch-operator
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-elasticsearch-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode}}
  - channel: v2.0
    name: ibm-opencontent-flink
    namespace: "{{ .CPFSNs }}"
    packageName: ibm-opencontent-flink
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
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
    excluded-catalogsource: certified-operators,community-operators,redhat-marketplace,ibm-cp-automation-foundation-catalog,operatorhubio-catalog
spec:
  operators:
  - name: ibm-licensing-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-licensing-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-mongodb-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-mongodb-operator-app
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-cert-manager-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-cert-manager-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-iam-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-iam-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-management-ingress-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-management-ingress-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - name: ibm-ingress-nginx-operator
    namespace: "{{ .ServicesNs }}"
    channel: v3.23
    packageName: ibm-ingress-nginx-operator-app
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
  - channel: v3.23
    name: ibm-zen-operator
    namespace: "{{ .ServicesNs }}"
    packageName: ibm-zen-operator
    scope: public
    installPlanApproval: {{ .ApprovalMode }}
    installMode: no-op
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
  channel: v4.3
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
