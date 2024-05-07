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
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.2@sha256:5390227d36006c0cd7ac4b957411707adb49ca40125156ceef44f8b593838e94
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.6@sha256:370e20dd2cb68b88bfd5a5ff5147d6a638e20952d4fda6215880a03dbfc77517
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.11@sha256:5f825e253f330006144895af8a8c8cd0cc91a95937652fb094b3244302aaa469
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.14@sha256:60328994ab265ca367b67259a1faacc349cf57f7bcccdb2c1494244a5663940d
  ibm-postgresql-12-operand-image: icr.io/cpopen/edb/postgresql:12.18@sha256:287c28ca4584e92bc1fae545c813c6e9a6723f978f0d3f8c2f29e40a0b15853f
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:c1670e7dd93c1e65a6659ece644e44aa5c2150809ac1089e2fd6be37dceae4ce
`
