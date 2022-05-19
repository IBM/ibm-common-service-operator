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

// Kind
const SecretshareKind = "SecretShare"

// ApiVersion
const SecretshareAPIVersion = "ibmcpcs.ibm.com/v1"

// Secretshare Operator CR
const SecretshareCR = `
apiVersion: ibmcpcs.ibm.com/v1
kind: SecretShare
metadata:
  name: common-services
  namespace: placeholder
spec:
  # Secrets to share for adopter compatibility to Common Services 3.2.4
  secretshares:
  - secretname: oauth-client-secret
    sharewith:
    - namespace: services
  - secretname: ibmcloud-cluster-ca-cert
    sharewith:
    - namespace: kube-public
  - secretname: icp-serviceid-apikey-secret
    sharewith:
    - namespace: kube-system
  - secretname: platform-oidc-credentials
    sharewith:
    - namespace: kube-system
  - secretname: icp-mongodb-admin
    sharewith:
    - namespace: kube-system
  - secretname: icp-mongodb-client-cert
    sharewith:
    - namespace: kube-system
  - secretname: cs-ca-certificate-secret
    sharewith:
    - namespace: kube-system
  # ConfigMaps to share for adopter compatibility to Common Services 3.2.4
  configmapshares:
  - configmapname: platform-auth-idp
    sharewith:
    - namespace: kube-system
  - configmapname: oauth-client-map
    sharewith:
    - namespace: services
  - configmapname: ibmcloud-cluster-info
    sharewith:
    - namespace: kube-public
`

// Secretshare Operator RBAC
const SecretshareRBAC = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: secretshare
  namespace: placeholder
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    version: "3.19.0"
  creationTimestamp: null
  name: secretshare
rules:
# create namespace it it doesn't exist
- verbs:
    - create
    - get
    - list
    - watch
  apiGroups:
    - ''
  resources:
    - namespaces
# copy secret and configmap to other namespaces
- verbs:
    - create
    - delete
    - get
    - list
    - patch
    - update
    - watch
  apiGroups:
    - ''
  resources:
    - events
    - configmaps
    - secrets
    - pods
# manage its own CR
- verbs:
    - create
    - delete
    - get
    - list
    - patch
    - update
    - watch
  apiGroups:
    - ibmcpcs.ibm.com
  resources:
    - secretshares
    - secretshares/status
# check if subscription is created
- verbs:
    - get
    - list
    - watch
  apiGroups:
    - operators.coreos.com
  resources:
    - subscriptions
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: secretshare-placeholder
subjects:
- kind: ServiceAccount
  name: secretshare
  namespace: placeholder
roleRef:
  kind: ClusterRole
  name: secretshare
  apiGroup: rbac.authorization.k8s.io
`

// Secretshare Operator CRD
const SecretshareCRD = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.4.0
    version: "3.19.0"
  name: secretshares.ibmcpcs.ibm.com
spec:
  group: ibmcpcs.ibm.com
  names:
    kind: SecretShare
    listKind: SecretShareList
    plural: secretshares
    singular: secretshare
  scope: Namespaced
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        description: SecretShare is the Schema for the secretshares API
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: SecretShareSpec defines the desired state of SecretShare
            properties:
              configmapshares:
                description: Configmapshares defines a list of configmap sharing information
                items:
                  description: Configmapshare identifies a Configmap required to be shared to another namespace
                  properties:
                    configmapname:
                      description: Configmapname is the name of the configmap waiting for sharing
                      type: string
                    sharewith:
                      description: Sharewith is a list of the target namespace for sharing
                      items:
                        description: TargetNamespace identifies the namespace the secret/configmap will be shared to
                        properties:
                          namespace:
                            description: Namespace is the target namespace of the secret or configmap
                            type: string
                        required:
                        - namespace
                        type: object
                      type: array
                  required:
                  - configmapname
                  - sharewith
                  type: object
                type: array
              secretshares:
                description: Secretshares defines a list of secret sharing information
                items:
                  description: Secretshare identifies a secret required to be shared to another namespace
                  properties:
                    secretname:
                      description: Secretname is the name of the secret waiting for sharing
                      type: string
                    sharewith:
                      description: Sharewith is a list of the target namespace for sharing
                      items:
                        description: TargetNamespace identifies the namespace the secret/configmap will be shared to
                        properties:
                          namespace:
                            description: Namespace is the target namespace of the secret or configmap
                            type: string
                        required:
                        - namespace
                        type: object
                      type: array
                  required:
                  - secretname
                  - sharewith
                  type: object
                type: array
            type: object
          status:
            description: SecretShareStatus defines the observed status of SecretShare
            properties:
              members:
                description: Members represnets the current operand status of the set
                properties:
                  configmapMembers:
                    additionalProperties:
                      description: MemberPhase identifies the status of the
                      type: string
                    description: ConfigmapMembers represnets the current operand status of the set
                    type: object
                  secretMembers:
                    additionalProperties:
                      description: MemberPhase identifies the status of the
                      type: string
                    description: SecretMembers represnets the current operand status of the set
                    type: object
                type: object
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
`

const CsSecretshareOperator = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: secretshare
  namespace: placeholder
  annotations:
    version: "3.19.0"
spec:
  replicas: 1
  selector:
    matchLabels:
      name: secretshare
  template:
    metadata:
      annotations:
        productID: 068a62892a1e4db39641342e592daa25
        productMetric: FREE
        productName: IBM Cloud Platform Common Services
      labels:
        name: secretshare
    spec:
      serviceAccountName: secretshare
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
        - /manager
        image: IBM_SECRETSHARE_OPERATOR_IMAGE
        imagePullPolicy: Always
        name: ibm-secretshare-operator
        env:
        - name: OPERATOR_NAME
          value: "secretshare"
        - name: OPERATOR_NAMESPACE
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 200m
            memory: 200Mi
      terminationGracePeriodSeconds: 10
`
