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
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.23@sha256:f67f5db05ba9d830d999f99248e7978812d498bd5f51f6cff3347b3672dd0580
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.18@sha256:60809f1ebc9df3c3aaed5aa2b8b6eb11402192c6cb949b4fbb4f4d622f300c19
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.14@sha256:97610e3b18dcdb90eda197481c2fb6b51bf4e4f39bebb97c1597b268c9dfefa4
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.10@sha256:1f566bbdc6feb07e818a06f680aaedaefdbb31a770562c27a2fe67ac2c970c44
  ibm-postgresql-18-operand-image: icr.io/cpopen/edb/postgresql:18.4@sha256:7c2b8e6f1820fdb29523de45ba965d111fe0d959fc88e61dc88c6479ae67f880
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:b839fa082ce65b218734d34e716119bb0c57ad0a761448c8b01153456f30728a
`
