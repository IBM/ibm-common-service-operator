//
// Copyright 2024 IBM Corporation
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

import "fmt"

var PostGresOperatorConfig string

// Populate PostGresOperatorConfig at package initialization
func init() {
	services := []string{
		"edb-keycloak",
		"cloud-native-postgresql",
		"common-service-postgresql",
		"cloud-native-postgresql-v1.22",
		"cloud-native-postgresql-v1.25",
	}

	servicesConfig := ""
	for _, service := range services {
		servicesConfig += fmt.Sprintf(postgresServiceTemplate, service)
	}

	PostGresOperatorConfig = `apiVersion: operator.ibm.com/v1alpha1
kind: OperatorConfig
metadata:
  name: cloud-native-postgresql-operator-config
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
    operator.ibm.com/experimental: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:` + servicesConfig
}

const postgresServiceTemplate = `
    - name: %s
      replicas: placeholder-size
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/arch
                operator: In
                values:
                - amd64
                - ppc64le
                - s390x
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 90
            podAffinityTerm:
              topologyKey: topology.kubernetes.io/zone
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values:
                  - cloud-native-postgresql
          - weight: 50
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values:
                  - cloud-native-postgresql
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: cloud-native-postgresql
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/region
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: cloud-native-postgresql`
