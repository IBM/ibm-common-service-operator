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
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.2@sha256:dfd176f664352298c481b40197ecb08678705e876cee44bc8a6d4d5998ee84e9
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.6@sha256:e43e67652b6b5c2faf3d59f9108a7d2ba7b7b1029f6d14915ec68ef362bab616
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.11@sha256:ba747b4a9666d66383e52009cd66b572c8fc22620a6e608f62e1552f9e979b5e
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.14@sha256:2cb7e3e7447bc16cb12c09ae84fe7a2a1f16ab6ed43cbf92313d45fc1628c17a
  ibm-postgresql-12-operand-image: icr.io/cpopen/edb/postgresql:12.18@sha256:5d743d6e5d2f840ff79efcb8b5aed26bc573f12e5f9a2776db71c21bb445925f
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:c9660f09a003178b13830fc260519c8d2e054ee8312e3cd1273d88d8d1acb759
`
