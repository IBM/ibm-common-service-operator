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
  ibm-postgresql-13-operand-image: icr.io/cpopen/edb/postgresql:13.21@sha256:6b71789c967b03d0b13358d93a635a4e9019d73683a8bf70c67fa406f52da07c
  ibm-postgresql-14-operand-image: icr.io/cpopen/edb/postgresql:14.18@sha256:8ceef1ac05972ab29b026fab3fab741a3c905f17773feee2b34f513e444f0fda
  ibm-postgresql-15-operand-image: icr.io/cpopen/edb/postgresql:15.13@sha256:47f886eefdda790410a896e557751e5934ddf6d13572b18a555b27295697b112
  ibm-postgresql-16-operand-image: icr.io/cpopen/edb/postgresql:16.9@sha256:e9e408a13bd103fb46536f896c51e7a27cba38d9b224231b20328a28efe1e616
  ibm-postgresql-17-operand-image: icr.io/cpopen/edb/postgresql:17.5@sha256:b72de7ac39caef09580e5fe55e8e0620d43fb1a00cd1e7f08a53f71c1d6631c5
  edb-postgres-license-provider-image: cp.icr.io/cp/cpd/edb-postgres-license-provider@sha256:095aae63ced410e033d8f7f3d27bd14dc5b93a10068ef5a389f181115c5d07c2
`
