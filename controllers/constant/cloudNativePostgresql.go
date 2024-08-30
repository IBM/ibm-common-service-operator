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
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.4@sha256:40f729ded0e488271950daf010c55963e5745c97a5d674dead80e7d2c4d24aeb
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.8@sha256:63c1bfc431fba3eba7a2e803d5d24c48425dbe7d1e9b1dec9832b30717ce8753
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.13@sha256:11317eb23ce45d74e35fd2471d96a32e4a28c29a0367734eb61f5a3d6aff2cff
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.16@sha256:ffb17e14727dbea67533decb12eed011a6138e191c947eda0c136443b823b867
  ibm-postgresql-12-operand-image: icr.io/cpopen/edb/postgresql:12.20@sha256:a4960e2d7350ab6beccd37e5e977ce3e3e06145216fc3c07f6df866fa9927b2f
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:6b5c69987f8967f5d0256a38e8759dad15480cf3c0eada9eb5fc71c51ed1cee9
`
