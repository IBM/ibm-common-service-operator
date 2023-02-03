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

const (
	ICPPKOperator  = "ibm-crossplane-provider-kubernetes-operator-app"
	ICPPICOperator = "ibm-crossplane-provider-ibm-cloud-operator-app"
	ICPOperator    = "ibm-crossplane-operator-app"
)

const CrossSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: {{ .ICPOperator }}
  namespace: "{{ .ControlNs }}"
spec:
  channel: {{ .Channel }}
  installPlanApproval: {{ .ApprovalMode }}
  name: {{ .ICPOperator }}
  source: {{ .CatalogSourceName }}
  sourceNamespace: "{{ .CatalogSourceNs }}"
`

const CrossConfiguration = `
apiVersion: pkg.ibm.crossplane.io/v1
kind: Configuration
metadata:
  name: ibm-crossplane-bedrock-shim-config
  labels:
    ibm-crossplane-provider: {{ .CrossplaneProvider }}
spec:
  ignoreCrossplaneConstraints: false
  package: ibm-crossplane-bedrock-shim-config
  packagePullPolicy: Never
  revisionActivationPolicy: Automatic
  revisionHistoryLimit: 1
  skipDependencyResolution: false
`

const CrossLock = `
apiVersion: pkg.ibm.crossplane.io/v1beta1
kind: Lock
metadata:
  name: lock
`

const CrossKubernetesProviderSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: {{ .ICPPKOperator }}
  namespace: "{{ .ControlNs }}"
spec:
  channel: {{ .Channel }}
  installPlanApproval: {{ .ApprovalMode }}
  name: {{ .ICPPKOperator }}
  source: {{ .CatalogSourceName }}
  sourceNamespace: "{{ .CatalogSourceNs }}"
`

const CrossKubernetesProviderConfig = `
apiVersion: kubernetes.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  finalizers:
    - in-use.crossplane.io
  name: kubernetes-provider
spec:
  credentials:
    source: InjectedIdentity
`

const CrossIBMCloudProviderSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: {{ .ICPPICOperator }}
  namespace: "{{ .ControlNs }}"
spec:
  channel: {{ .Channel }}
  installPlanApproval: {{ .ApprovalMode }}
  name: {{ .ICPPICOperator }}
  source: {{ .CatalogSourceName }}
  sourceNamespace: "{{ .CatalogSourceNs }}"
`

const CrossIBMCloudProviderConfig = `
apiVersion: ibmcloud.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: ibm-crossplane-provider-ibm-cloud
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: "{{ .ControlNs }}"
      name: provider-ibm-cloud-secret
      key: credentials
  region: us-south
`

const CFCrossplaneConfigMap = `
apiVersion: v1
data:
  removeFinalizersFromObjects: REMOVAL
kind: ConfigMap
metadata:
  name: cf-crossplane
  namespace: "placeholder"
`
