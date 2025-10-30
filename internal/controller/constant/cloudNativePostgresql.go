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
    operator.ibm.com/watched-by-odlm: "true"
  annotations:
    version: {{ .Version }}
data:
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.22@sha256:950bcd0cbafaad3d7286d08c26b7ff55b03b860ec4bb204ae9dbc8ea9178eb3b
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.19@sha256:70789d893cfec43d0daa2f1e519df7b5aaca5e20952a22c0ba743b06af1c8ce7
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.14@sha256:4fdcb1fad9ffa11d65a9f144a6280051fc9d3d92c36ee2e6d9af67458e39b39d
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.10@sha256:c25a5aa73590c3140d6982749e627d2b665567809b28fec23eee90b47438f9cd
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.6@sha256:b44d7d332140c5be49290c8cd62ad06451d4af439f44ed8edd0a11d5c009ce90
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:c2759e7125878a909f4ff69db60b5989757d829d313bef01041addf517ef3226
`
