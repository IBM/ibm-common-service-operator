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
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.4@sha256:d87a804d9bb7558d124bd83a83c261a074c26bccc3fe6ef095cdbe22be29a456
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.8@sha256:8f602b668e1174357332374094a93534a2ab132e954badcb82b331c2e04b65da
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.15@sha256:e027ee5a9aaebd369196c119481a14eba961119a3c3b1748aac06936bcb3afe1
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.16@sha256:9fc9cc5dd91c38797397a02736996857d0182e4858ee77aeeb21f855121c9347
  ibm-postgresql-12-operand-image: icr.io/cpopen/edb/postgresql:12.20@sha256:1a5ec719c2f7da6d98374cfa43a03f96a56195416ff0c7443ee255dc92d8a82b
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:8112cfa96daac82de5a4fdede8fcaecfc41908e7ee22686e0f8818c875784a00
`
