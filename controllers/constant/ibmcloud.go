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

const IbmCloudSubscription = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ibmcloud-operator
  namespace: {{ .MasterNs }}
  annotations:
    version: {{ .Version }}
spec:
  channel: {{ .Channel }}
  installPlanApproval: Automatic
  name: ibmcloud-operator
  source: {{ .CatalogSourceName }}
  sourceNamespace: {{ .CatalogSourceNs }}
`

// apiVersion: operators.coreos.com/v1alpha1
// kind: Subscription
// metadata:
//   name: ibmcloud-operator
//   namespace: ibm-common-services
// spec:
//   channel: stable
//   installPlanApproval: Automatic
//   name: ibmcloud-operator
//   source: community-operators
//   sourceNamespace: openshift-marketplace
