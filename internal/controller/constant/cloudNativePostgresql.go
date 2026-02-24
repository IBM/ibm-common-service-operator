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
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.23@sha256:353d13d874d60f7dff5dba2b080e8b127d54f3d71f9c5421f3895d737a604d41
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.20@sha256:5b9e17fa0f210e78622e965d4da7f79c20b99cf8db507451e5453dd4c3dab8b7
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.15@sha256:425de739fb88db833881eb0c1c9db7408059bec5b5f45830c92e5de4b4ebbddc
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.11@sha256:b711c28ad678c53fae114a3b80b801979dd30cf4b7f6676db0db5c9987e377d3
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.7@sha256:4becaff092407bf0c4a55b921c1ff8a9f2e4cb2b53c339eea41269ee38a21f12
  ibm-postgresql-18-operand-image: icr.io/cpopen/edb/postgresql:18.1@sha256:022b2d0beb9ae959ddd7c06038a4369f935fc07edc9cd288bc019d81b6ef1a49
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:bd60720f4a317cae809f1bc345e1582b7f9caadf28c0aede9b370c64cf521bf4
`
