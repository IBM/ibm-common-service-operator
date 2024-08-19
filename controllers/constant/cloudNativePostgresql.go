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
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.3@sha256:0fd61248ef26dc90b72f7bd7df1c094c6ba8b216fb398f8878765fd425b286e9
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.7@sha256:0328e8cbf635a0da828fb70300bfe10ac22e1686261f71a75bfc32f8505c7dfe
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.12@sha256:2ccccce28ed1cdb15b21f7fcb083570e3dbc98159f064ae84c4c9a9d4b9e4d53
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.15@sha256:063879a85c8ea1cf38df2043a00e9db490ed7660a652885c634a7f4e7c39d0be
  ibm-postgresql-12-operand-image: icr.io/cpopen/edb/postgresql:12.19@sha256:ed4da158d8551759d3f5994237b2c7f3ac7a1c8d01510e04d1a394a04279811a
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:6b5c69987f8967f5d0256a38e8759dad15480cf3c0eada9eb5fc71c51ed1cee9
`
