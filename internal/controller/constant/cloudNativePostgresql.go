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
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.23@sha256:063c8a38d80a2b34f06ccdf4e432424a2d1257cddbefb87794b39ae505f8c921
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.18@sha256:3d2f9ceb5f3414881d67a78f921303e47b2eb90601c0a938fea6d16611d26b0f
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.14@sha256:e32d271e1d9635b6d9b9b46d84f4984a28d3cc863d828a4fbb42ec082c5976bd
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.10@sha256:256e0a72bbcd490af6071e82d8fb1697a24835223cd186b2673b72f6d8add4d9
  ibm-postgresql-18-operand-image: icr.io/cpopen/edb/postgresql:18.4@sha256:8b046402ce495e262f6589b4ad16d960e190f151a42751461f5939b2275511db
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:b259810099ca7a4150f00ccaa870b02890d1d7f022f3ac330918d515968a033b
`
