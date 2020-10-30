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

// Extra RBAC
const ExtraRBAC = `
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ibmcloud-cluster-info
  namespace: kube-public
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    resourceNames: ["ibmcloud-cluster-info"]
    verbs: ["get"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ibmcloud-cluster-info
  namespace: kube-public
subjects:
  - kind: Group
    apiGroup: rbac.authorization.k8s.io
    name: "system:authenticated"
  - kind: Group
    apiGroup: rbac.authorization.k8s.io
    name: "system:unauthenticated"
roleRef:
  kind: Role
  name: ibmcloud-cluster-info
  apiGroup: rbac.authorization.k8s.io

---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ibmcloud-cluster-ca-cert
  namespace: kube-public
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["ibmcloud-cluster-ca-cert"]
    verbs: ["get"]

---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ibmcloud-cluster-ca-cert
  namespace: kube-public
subjects:
  - kind: Group
    apiGroup: rbac.authorization.k8s.io
    name: "system:authenticated"
  - kind: Group
    apiGroup: rbac.authorization.k8s.io
    name: "system:unauthenticated"
roleRef:
  kind: Role
  name: ibmcloud-cluster-ca-cert
  apiGroup: rbac.authorization.k8s.io
`
