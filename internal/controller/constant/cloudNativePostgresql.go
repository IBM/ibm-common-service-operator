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
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.22@sha256:d581cc2f72293f58facec185900dba417be03a02a170b0f6b41139e4586f9e3a
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.19@sha256:96d2abcf56efa08c6c079bd7ae88e89c3248892ce36a5b3b9c904d61cc6c640c
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.14@sha256:00ddf0d6ad061f9e58290610478d99ab76443a3eeda0a799ec21d74006735071
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.10@sha256:c65e9156f20ded832c3941b8c5d085c45cbe976593ca75ca3736af37acc7b76f
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.6@sha256:06af5e116a837794ebb08fce779a52e40e9ea733ca434f8562659a69e2e2f1a6
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:2238520c04ade21cd2e22b947cd94a8f3b5e6506f918ec83af3cb93fd65a4249
`
