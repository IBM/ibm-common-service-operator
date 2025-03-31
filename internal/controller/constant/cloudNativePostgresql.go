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
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.4@sha256:5f31883f560796827f8c64d43a06acb214e031d8931de63a0ccd56c3b27a9f36
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.8@sha256:43599de779d84195ffc7558541883545068aec300a2b745303327ea8b7b5aaf4
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.12@sha256:bfb664c8d6720e3ca19d698141a4188769e122d0535498cd522966195697dce0
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.17@sha256:f2a7f7cb13b7582dc629f2d484687573f88cc255f8af7849da630c7b4cfde4d0
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.20@sha256:adfa36eda97f9fbcefa23dd0e39dc4386380f33dbd26a5c2960e08854dc1af8d
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:443b51f8b10acc85bdefde7193e0f45b1c423fa7fbdcaa28342805815c43db3d
`
