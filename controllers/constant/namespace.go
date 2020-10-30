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

// NamespaceScope Operator CR
const NamespaceScopeCR = `
apiVersion: operator.ibm.com/v1
kind: NamespaceScope
metadata:
  name: common-service
  namespace: placeholder
spec:
  namespaceMembers:
  - placeholder

  restartLabels:
    intent: projected
`

// NamespaceScope Operator RBAC
const NamespaceScopeRBAC = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ibm-namespace-scope-operator
  namespace: placeholder
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  creationTimestamp: null
  name: ibm-namespace-scope-operator
  namespace: placeholder
rules:
- apiGroups:
  - "*"
  resources:
  - "*"
  verbs:
  - "*"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ibm-namespace-scope-operator
  namespace: placeholder
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: ibm-namespace-scope-operator
subjects:
- kind: ServiceAccount
  name: ibm-namespace-scope-operator
`

// NamespaceScope Operator CRD
const NamespaceScopeCRD = `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.3.0
  creationTimestamp: null
  name: namespacescopes.operator.ibm.com
spec:
  group: operator.ibm.com
  names:
    kind: NamespaceScope
    listKind: NamespaceScopeList
    plural: namespacescopes
    singular: namespacescope
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      description: NamespaceScope is the Schema for the namespacescopes API
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          description: NamespaceScopeSpec defines the desired state of NamespaceScope
          properties:
            configmapName:
              description: ConfigMap name that will contain the list of namespaces
                to be watched
              type: string
            manualManagement:
              description: Set the following to true to manaually manage permissions
                for the NamespaceScope operator to extend control over other namespaces
                The operator may fail when trying to extend permissions to other namespaces,
                but the cluster administrator can correct this using the authorize-namespace
                command.
              type: boolean
            namespaceMembers:
              description: Namespaces that are part of this scope
              items:
                type: string
              type: array
            restartLabels:
              additionalProperties:
                type: string
              description: Restart pods with the following labels when the namspace
                list changes
              type: object
          type: object
        status:
          description: NamespaceScopeStatus defines the observed state of NamespaceScope
          type: object
      type: object
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
`
