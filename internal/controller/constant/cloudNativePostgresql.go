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
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.22@sha256:841db53fc45ebcb3c0dbbb0f68c2fdfe80cda1c7a5871ac1956fc80e18198e27
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.19@sha256:bce78b457ffeb4080aaac01dee712aa0e852ab9a3b4b42414634310b85019d8e
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.14@sha256:44167b92101095992f3c1e1d7727db7e9e624fc5ed81c13a2d5efb1f1901c70d
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.10@sha256:0ec7d9cc2211ce38d9881bda9b0a1370f0420d54ff18f90495530c7cf02c0301
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.6@sha256:6985297657d7b7b916427f7a0da9684cf7b269cb9015dfdacbaa7f68659e7ae4
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:2f921d1c39f72f183ca285151acf5d0dc6bd9e71a24acd1db17e6b05baf4935c
`
