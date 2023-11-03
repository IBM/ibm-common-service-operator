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

const EDBImageConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cloud-native-postgresql-image-list
  namespace: "{{ .CPFSNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
  annotations:
    version: {{ .Version }}
data:
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.4@sha256:7ae65a0e9f172a6882b29a629d2f727aa9b6114c82eb116df5a934bf2797c17c
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.9@sha256:90136074adcbafb5033668b07fe1efea9addf0168fa83b0c8a6984536fc22264
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.12@sha256:7f8188c114ea8d1a2f706e7e7672e3462a50f96e7eec4da137e9996c4e40b674
  ibm-postgresql-12-operand-image: icr.io/cpopen/edb/postgresql:12.16@sha256:ff99a217198e58f89f847ddeea1ddc153593123f93b88fb0ece39d3c1f59cb6e
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:05f30f2117ff6e0e853487f17785024f6bb226f3631425eaf1498b9d3b753345
`
