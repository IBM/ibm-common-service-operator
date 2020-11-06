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
  - secretname: icp-metering-api-secret
    sharewith:
    - namespace: kube-system
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
    version: "2"
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
  name: secretshare-ibm-common-services
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
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: secretshares.ibmcpcs.ibm.com
spec:
  group: ibmcpcs.ibm.com
  names:
    kind: SecretShare
    listKind: SecretShareList
    plural: secretshares
    singular: secretshare
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      type: object
      x-kubernetes-preserve-unknown-fields: true
  versions:
  - name: v1
    served: true
    storage: true
`
