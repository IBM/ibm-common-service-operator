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

// Package names for PostgreSQL operators
const (
	CloudNativePostgreSQLPackage = "cloud-native-postgresql"
	IBMPGOperatorPackage         = "ibm-pg-operator"
)

// OperatorConfig names for PostgreSQL operators
const (
	CloudNativePostgreSQLOperatorConfigName = "cloud-native-postgresql-operator-config"
	IBMPGOperatorConfigName                 = "ibm-pg-operator-config"
)

// EDBOperatorServices lists all EDB PostgreSQL operator services.
// This is the single source of truth for EDB operator service names used across:
// - OperatorConfig templates (init function)
// - Operator grouping logic (operatorconfig.go)
// - Size profiles (size/*.go)
var EDBOperatorServices = []string{
	"edb-keycloak",
	"cloud-native-postgresql",
	"common-service-postgresql",
	"cloud-native-postgresql-v1.22",
	"cloud-native-postgresql-v1.25",
	"cloud-native-postgresql-v1.28",
}

// IBMPGOperatorServices lists all IBM PG operator services.
// This is the single source of truth for IBM PG operator service names used across:
// - OperatorConfig templates (init function)
// - Operator grouping logic (operatorconfig.go)
// - Size profiles (size/*.go)
var IBMPGOperatorServices = []string{
	"ibm-pg-operator-v28",
	"common-service-cnpg",
	"common-service-pg-migrator",
}

// PostGresOperatorConfig contains the OperatorConfig template for EDB PostgreSQL operators.
// This is used for the cloud-native-postgresql package which includes EDB-based PostgreSQL services.
var PostGresOperatorConfig string

// IBMPGOperatorConfig contains the OperatorConfig template for IBM PostgreSQL operators.
// This is used for the ibm-pg-operator package which includes IBM's PostgreSQL implementation.
// The IBM PG operator is the successor to EDB and provides enhanced features and support.
var IBMPGOperatorConfig string

// Populate PostGresOperatorConfig and IBMPGOperatorConfig at package initialization.
// These templates define HA topology constraints including:
// - Node affinity for multi-architecture support (amd64, ppc64le, s390x)
// - Pod anti-affinity to spread replicas across zones and hosts
// - Topology spread constraints for zone and region distribution
func init() {
	// Build EDB PostgreSQL operator config from the service list
	servicesConfig := ""
	for _, service := range EDBOperatorServices {
		servicesConfig += fmt.Sprintf(postgresServiceTemplate, service)
	}

	PostGresOperatorConfig = `apiVersion: operator.ibm.com/v1alpha1
kind: OperatorConfig
metadata:
  name: ` + CloudNativePostgreSQLOperatorConfigName + `
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
    operator.ibm.com/experimental: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:` + servicesConfig

	// Build IBM PG operator config from the service list
	ibmPGServicesConfig := ""
	for _, service := range IBMPGOperatorServices {
		ibmPGServicesConfig += fmt.Sprintf(ibmPGServiceTemplate, service)
	}

	IBMPGOperatorConfig = `apiVersion: operator.ibm.com/v1alpha1
kind: OperatorConfig
metadata:
  name: ` + IBMPGOperatorConfigName + `
  namespace: "{{ .ServicesNs }}"
  labels:
    operator.ibm.com/managedByCsOperator: "true"
    operator.ibm.com/experimental: "true"
  annotations:
    version: {{ .Version }}
spec:
  services:` + ibmPGServicesConfig
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

const ibmPGServiceTemplate = `
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
                  - ibm-pg-operator
          - weight: 50
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values:
                  - ibm-pg-operator
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: ibm-pg-operator
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/region
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: ibm-pg-operator`
