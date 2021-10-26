//
// Copyright 2021 IBM Corporation
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

const NSRestrictedSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ibm-namespace-scope-operator-restricted
  namespace: {{ .MasterNs }}
spec:
  channel: {{ .Channel }}
  installPlanApproval: Automatic
  name: ibm-namespace-scope-operator-restricted
  source: {{ .CatalogSourceName }}
  sourceNamespace: {{ .CatalogSourceNs }}
`

const NSSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ibm-namespace-scope-operator
  namespace: {{ .MasterNs }}
spec:
  channel: {{ .Channel }}
  installPlanApproval: Automatic
  name: ibm-namespace-scope-operator
  source: {{ .CatalogSourceName }}
  sourceNamespace: {{ .CatalogSourceNs }}
`

// NamespaceScope Operator CR
const NamespaceScopeCR = `
apiVersion: operator.ibm.com/v1
kind: NamespaceScope
metadata:
  name: common-service
  namespace: {{ .MasterNs }}
spec:
  csvInjector:
    enable: true
  namespaceMembers:
  - {{ .MasterNs }}
  - openshift-redhat-marketplace
---
apiVersion: operator.ibm.com/v1
kind: NamespaceScope
metadata:
  name: nss-odlm-scope
  namespace: {{ .MasterNs }}
spec:
  namespaceMembers:
  - {{ .MasterNs }}
  configmapName: odlm-scope
  restartLabels:
    intent: projected-odlm
`

// NamespaceScope Operator CR Managed By ODLM
const NamespaceScopeCRManagedbyODLM = `
apiVersion: operator.ibm.com/v1
kind: NamespaceScope
metadata:
  name: nss-managedby-odlm
  namespace: {{ .MasterNs }}
spec:
  namespaceMembers:
  - {{ .MasterNs }}
---
apiVersion: operator.ibm.com/v1
kind: NamespaceScope
metadata:
  name: odlm-scope-managedby-odlm
  namespace: {{ .MasterNs }}
spec:
  namespaceMembers:
  - {{ .MasterNs }}
  configmapName: odlm-scope
  restartLabels:
    intent: projected-odlm
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
---
apiVersion: v1
data:
  namespaces: placeholder
kind: ConfigMap
metadata:
  name: odlm-scope
  namespace: placeholder
`
