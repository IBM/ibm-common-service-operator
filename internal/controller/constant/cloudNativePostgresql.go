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
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.22@sha256:6d74c43c7f6d016c28484d252607feeb2b1c5a2a0539db60bc685718d55cc770
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.17@sha256:1f31a3ab18f4f9d0ef852f4dd22300746150912e5a4731ed07195f9c79674e40
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.13@sha256:c632b5ef78ab6686939e8f76543c08dac8cb77e7b30578b21eb0a0c8d3b0b020
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.9@sha256:69e19c5ee470943d281445c670dac682e4963362ca9e8638b2623cbf27124e33
  ibm-postgresql-18-operand-image: icr.io/cpopen/edb/postgresql:18.3@sha256:6a2e4c1108532a28d89b83e33d8206e543b63372e31f6831b997740de36d3b79
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:fd61ba81bd15d53e79ed8bc7467acf4e92b813dc8dac916e30231468efca2afe
`
