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

const CrossSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ibm-crossplane-operator-app
  namespace: {{ .MasterNs }}
  annotations:
    version: {{ .Version }}
spec:
  channel: {{ .Channel }}
  installPlanApproval: Automatic
  name: ibm-crossplane-operator-app
  source: {{ .CatalogSourceName }}
  sourceNamespace: {{ .CatalogSourceNs }}
`

// sample subscription
// apiVersion: operators.coreos.com/v1alpha1
// kind: Subscription
// metadata:
//   name: ibm-crossplane-operator-app
//   namespace: ibm-common-services
// spec:
//   channel: v3
//   installPlanApproval: Automatic
//   name: ibm-crossplane-operator-app
//   source: opencloud-operators
//   sourceNamespace: openshift-marketplace
//   startingCSV: ibm-crossplane-operator.v1.0.0

// CrossConfigMap is the init configmap
const CrossConfigMap = `
apiVersion: v1
data:
  namespaces: placeholder
kind: ConfigMap
metadata:
  name: crossplane
  namespace: placeholder
`
