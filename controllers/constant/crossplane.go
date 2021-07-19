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
spec:
  channel: {{ .Channel }}
  installPlanApproval: Automatic
  name: ibm-crossplane-operator-app
  source: {{ .CatalogSourceName }}
  sourceNamespace: {{ .CatalogSourceNs }}
`

const CrossplaneCR = `
apiVersion: pkg.crossplane.io/v1
kind: Configuration
metadata:
  annotations:
  name: ibm-crossplane-bedrock-shim-config
  labels:
    app.kubernetes.io/instance: ibm-crossplane
    app.kubernetes.io/managed-by: ibm-crossplane-operator
    app.kubernetes.io/name: ibm-crossplane-operator
spec:
  ignoreCrossplaneConstraints: false
  package: 'quay.io/opencloudio/ibm-crossplane-bedrock-shim-config:1.0.0'
  packagePullPolicy: Always
  revisionActivationPolicy: Automatic
  revisionHistoryLimit: 1
  skipDependencyResolution: false
`
