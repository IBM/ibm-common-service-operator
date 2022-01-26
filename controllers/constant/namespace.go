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
---
apiVersion: operator.ibm.com/v1
kind: NamespaceScope
metadata:
  name: nss-managedby-odlm
  namespace: placeholder
spec:
  namespaceMembers:
  - placeholder
`

// NamespaceScopeConfigMap is the init configmap
const NamespaceScopeConfigMap = `
apiVersion: v1
data:
  namespaces: placeholder
kind: ConfigMap
metadata:
  name: namespace-scope
  namespace: placeholder
`
