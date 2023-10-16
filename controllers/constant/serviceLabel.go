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

// still need flink and elastic
const ServiceLabelTemplate = `
- name: ibm-iam-operator
  spec:
    authentication:
      labels:
	  	placeholder
- name: ibm-im-mongodb-operator
	spec:
		mongoDB:
			labels:
				placeholder
- name: ibm-im-operator
  spec:
    authentication:
      labels:
	  - name: placeholder1
	  	value: placeholder2
- name: ibm-im-operator-v4.0
  spec:
    authentication:
      labels:
	  	placeholder
- name: ibm-im-operator-v4.1
  spec:
    authentication:
      labels:
	  	placeholder
- name: ibm-im-operator-v4.2
  spec:
    authentication:
      labels:
	  	placeholder
- name: ibm-idp-config-ui-operator-v4.0
	spec:
		commonWebUI: 
			labels:
				placeholder
- name: ibm-idp-config-ui-operator-v4.1
	spec:
		commonWebUI: 
			labels:
				placeholder
- name: ibm-idp-config-ui-operator-v4.2
	spec:
		commonWebUI: 
			labels:
				placeholder
`

const LabelTemplate = `
- name: placeholder1
  value: placeholder2
`
