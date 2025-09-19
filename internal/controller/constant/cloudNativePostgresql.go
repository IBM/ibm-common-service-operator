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
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.22@sha256:5bdcb3af39b9333e15ae344ef98bb48104e631b6687d30e3c94f0706e6a60518
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.19@sha256:be2c432bdbb2efe9dd88f13f09c2417576318e1f0133f7a8dea27a992e385762
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.14@sha256:0bf80362de71d1541d9c64ebe62061da3f2debdeb67b7aaf6c98b774274fa372
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.10@sha256:f7583d6896684cf13646624761322890d0b9602181bb9196634479db8b3bfbfd
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.6@sha256:78a5c61832fc29f7331d361de1db690403e6e248aa4d0937932d1a66b88d0a55
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:569fb57931219caa12ad81c2d1a2267e5dfbf9e45e010a2a33b40d1978f90d07
`
