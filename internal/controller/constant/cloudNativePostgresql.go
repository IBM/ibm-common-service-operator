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
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.22@sha256:8b1f10ce2ca260a381eac0677c70d74a7c733092e0f3625f9b12b425edfb3cc8
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.17@sha256:c4f7664b77ec21fb43e924006640da5d55ed39a596d25a90c8374410d362e673
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.13@sha256:a16a63c48cd117e592a3e2fd15f31802a0e1ee357c71630aaba38452fa2f8c04
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.9@sha256:d4fe1575db25f382608de20b552dd042f3bb4542387e6f82910dafd026d170e1
  ibm-postgresql-18-operand-image: icr.io/cpopen/edb/postgresql:18.3@sha256:5ac13a0948a0648306cf6792a882db2e1a24ee55bda3b6adf78a7aba35f5b7aa
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:f1af89c4aa6d9f8e842c1afeb9969e529f778368230ff1de8d7631adce318a36
`
