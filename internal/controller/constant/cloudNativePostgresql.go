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
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.21@sha256:4bd4706fd49cedcfa2c5325b0a92205fde2388bc4244c33efa66ab3cf7ab231f
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.18@sha256:09994dff6bb3f2f2b31badc2e3e98cef3c6cfdc8bc48a9dc467211d0e7802001
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.13@sha256:e70ab1db7cfda84eb538d4d439046155f9488cc73b2ed43daf164944e3e9286d
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.9@sha256:a9242382f2d398cfe95ced690c0d00082bffd86b903717369485b2c63b7e1e21
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.5@sha256:d44ae0cde4bf517ba62168259a0d610c6a3a02a86b46e60e1f897cb2dc851f2d
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:5dfd41bc9f85b14ff634b64699efd100baf0fd408aec20c52188c141a3a94aa2
`
