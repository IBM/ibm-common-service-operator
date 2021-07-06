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
apiVersion: operator.ibm.com/v1beta1
kind: Crossplane
metadata:
  namespace: {{ .MasterNs }}
  name: ibm-crossplane
  labels:
    app.kubernetes.io/instance: ibm-crossplane
    app.kubernetes.io/managed-by: ibm-crossplane-operator
    app.kubernetes.io/name: ibm-crossplane-operator
spec:
  configuration:
    packages:
      - 'quay.io/opencloudio/ibm-crossplane-bedrock-shim-config:1.0.0'
  replicas: 1
  resourcesCrossplane:
    limits:
      cpu: 100m
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 256Mi
`
