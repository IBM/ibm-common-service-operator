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
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.20@sha256:4ccd786884e39641a74096d783742c219fb277caafe80b667a80396c3a9cf6c3
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.17@sha256:99589ee11b50af8ed6355a5f9c2272ce1fc7f9a13e80fda8eda5883a332782a3
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.12@sha256:0341c499ae144ee81dc2b92e8b99ee313ff6759374b7c1a6469156c20f522921
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.8@sha256:6d903fd4bd0ef3ef361d22b209ba7b686601b49d6f8d4e29b5d030da8d949dd5
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.4@sha256:c330a321438943fdd264b292b68fdfb8be89c3bde93a7bd987bd92c9dd2512b0
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:443b51f8b10acc85bdefde7193e0f45b1c423fa7fbdcaa28342805815c43db3d
`
