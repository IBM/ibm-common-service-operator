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

const StorageClassTemplate = `
- name: ibm-im-mongodb-operator
  spec:
    mongoDB:
      storageClass: placeholder
- name: ibm-im-mongodb-operator-v4.0
  spec:
    mongoDB:
      storageClass: placeholder
- name: ibm-im-mongodb-operator-v4.1
  spec:
    mongoDB:
      storageClass: placeholder
- name: ibm-im-mongodb-operator-v4.2
  spec:
    mongoDB:
      storageClass: placeholder
- name: ibm-healthcheck-operator
  spec:
    mustgatherService:
      persistentVolumeClaim:
        storageClassName: placeholder
- name: edb-keycloak
  resources:
    - apiVersion: postgresql.k8s.enterprisedb.io/v1
      data:
        spec:
          storage:
            storageClass: placeholder
      kind: Cluster
      name: keycloak-edb-cluster
`
