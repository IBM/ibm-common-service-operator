//
// Copyright 2020 IBM Corporation
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

package size

const Medium = `
- name: ibm-cert-manager-operator
  spec:
    certManager:
      certManagerCAInjector:
        resources:
          limits:
            cpu: 35m
            memory: 520Mi
          requests:
            cpu: 30m
            memory: 350Mi
      certManagerController:
        resources:
          limits:
            cpu: 80m
            memory: 530Mi
          requests:
            cpu: 70m
            memory: 390Mi
      certManagerWebhook:
        resources:
          limits:
            cpu: 60m
            memory: 100Mi
          requests:
            cpu: 50m
            memory: 90Mi
      configMapWatcher:
        resources:
          limits:
            cpu: 10m
            memory: 60Mi
          requests:
            cpu: 10m
            memory: 60Mi
- name: ibm-mongodb-operator
  spec:
    mongoDB:
      replicas: 3
      resources:
        limits:
          cpu: 2000m
          memory: 2Gi
        requests:
          cpu: 2000m
          memory: 2Gi
- name: ibm-iam-operator
  spec:
    authentication:
      replicas: 2
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            cpu: 50m
            memory: 50Mi
      authService:
        resources:
          limits:
            cpu: 1000m
            memory: 745Mi
          requests:
            cpu: 370m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 1260Mi
          requests:
            cpu: 290m
            memory: 940Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 920Mi
          requests:
            cpu: 590m
            memory: 690Mi
    oidcclientwatcher:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 325Mi
        requests:
          cpu: 30m
          memory: 50Mi
    pap:
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            cpu: 50m
            memory: 50Mi
      papService:
        resources:
          limits:
            cpu: 1000m
            memory: 600Mi
          requests:
            cpu: 50m
            memory: 195Mi
      replicas: 2
    policycontroller:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 50Mi
        requests:
          cpu: 20m
          memory: 50Mi
    policydecision:
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            cpu: 20m
            memory: 50Mi
      resources:
        limits:
          cpu: 1000m
          memory: 160Mi
        requests:
          cpu: 110m
          memory: 160Mi
      replicas: 2
    secretwatcher:
      resources:
        limits:
          cpu: 1000m
          memory: 220Mi
        requests:
          cpu: 30m
          memory: 220Mi
      replicas: 1
    securityonboarding:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 50Mi
        requests:
          cpu: 20m
          memory: 50Mi
      iamOnboarding:
        resources:
          limits:
            cpu: 1000m
            memory: 1024Mi
          requests:
            cpu: 20m
            memory: 64Mi
- name: ibm-management-ingress-operator
  spec:
    managementIngress:
      replicas: 2
      resources:
        requests:
          cpu: 200m
          memory: 256Mi
        limits:
          cpu: 1000m
          memory: 760Mi
- name: ibm-ingress-nginx-operator
  spec:
    nginxIngress:
      ingress:
        replicas: 2
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 860Mi
      defaultBackend:
        replicas: 1
        resources:
          requests:
            cpu: 20m
            memory: 50Mi
          limits:
            cpu: 50m
            memory: 100Mi
      kubectl:
        resources:
          requests:
            memory: 150Mi
            cpu: 30m
          limits:
            memory: 150Mi
            cpu: 50m
- name: ibm-metering-operator
  spec:
    metering:
      dataManager:
        dm:
          resources:
            limits:
              cpu: 450m
              memory: 850Mi
            requests:
              cpu: 200m
              memory: 230Mi
      reader:
        rdr:
          resources:
            limits:
              cpu: 50m
              memory: 290Mi
            requests:
              cpu: 30m
              memory: 230Mi
    meteringReportServer:
      reportServer:
        resources:
          limits:
            cpu: 100m
            memory: 90Mi
          requests:
            cpu: 50m
            memory: 50Mi
    meteringUI:
      replicas: 1
      ui:
        resources:
          limits:
            cpu: 100m
            memory: 256Mi
          requests:
            cpu: 50m
            memory: 100Mi
- name: ibm-licensing-operator
  spec:
    IBMLicensing:
      resources:
        requests:
          cpu: 200m
          memory: 270Mi
        limits:
          cpu: 300m
          memory: 350Mi
    IBMLicenseServiceReporter:
      databaseContainer:
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 300m
            memory: 300Mi
      receiverContainer:
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 300m
            memory: 300Mi
- name: ibm-commonui-operator
  spec:
    commonWebUI:
      replicas: 2
      resources:
        requests:
          memory: 480Mi
          cpu: 450m
        limits:
          memory: 660Mi
          cpu: 1000m
      commonWebUIConfig:
        dashboardData:
          resources:
            limits:
              cpu: 3000m
              memory: 460Mi
            requests:
              cpu: 300m
              memory: 230Mi
- name: ibm-platform-api-operator
  spec:
    platformApi:
      auditService:
        resources:
          limits:
            cpu: 25m
            memory: 50Mi
          requests:
            cpu: 25m
            memory: 50Mi
      platformApi:
        resources:
          limits:
            cpu: 25m
            memory: 50Mi
          requests:
            cpu: 25m
            memory: 50Mi
      replicas: 1
- name: ibm-catalog-ui-operator
  spec:
    catalogUI:
      catalogui:
        resources:
          limits:
            cpu: 190m
            memory: 230Mi
          requests:
            cpu: 45m
            memory: 130Mi
      replicaCount: 1
- name: ibm-healthcheck-operator
  spec:
    healthService:
      memcached:
        replicas: 1
        resources:
          requests:
            memory: 50Mi
            cpu: 20m
          limits:
            memory: 100Mi
            cpu: 200m
      healthService:
        replicas: 1
        resources:
          requests:
            memory: 125Mi
            cpu: 20m
          limits:
            memory: 250Mi
            cpu: 200m
- name: ibm-auditlogging-operator
  spec:
    auditLogging:
      fluentd:
        resources:
          requests:
            cpu: 35m
            memory: 128Mi
          limits:
            cpu: 50m
            memory: 200Mi
- name: ibm-monitoring-exporters-operator
  spec:
    exporter:
      collectd:
        resource:
          requests:
            cpu: 30m
            memory: 50Mi
          limits:
            cpu: 30m
            memory: 50Mi
        routerResource:
          limits:
            cpu: 25m
            memory: 50Mi
          requests:
            cpu: 20m
            memory: 50Mi
      nodeExporter:
        resource:
          requests:
            cpu: 7m
            memory: 50Mi
          limits:
            cpu: 20m
            memory: 50Mi
        routerResource:
          requests:
            cpu: 50m
            memory: 128Mi
          limits:
            cpu: 100m
            memory: 256Mi
      kubeStateMetrics:
        resource:
          requests:
            cpu: 500m
            memory: 180Mi
          limits:
            cpu: 540m
            memory: 220Mi
        routerResource:
          limits:
            cpu: 25m
            memory: 50Mi
          requests:
            cpu: 20m
            memory: 50Mi
- name: ibm-monitoring-grafana-operator
  spec:
    grafana:
      grafanaConfig:
        resources:
          requests:
            cpu: 25m
            memory: 200Mi
          limits:
            cpu: 150m
            memory: 230Mi
      dashboardConfig:
        resources:
          requests:
            cpu: 25m
            memory: 140Mi
          limits:
            cpu: 70m
            memory: 170Mi
      routerConfig:
        resources:
          requests:
            cpu: 25m
            memory: 65Mi
          limits:
            cpu: 70m
            memory: 80Mi
- name: ibm-monitoring-prometheusext-operator
  spec:
    prometheusExt:
      prometheusConfig:
        routerResource:
          requests:
            cpu: 10m
            memory: 50Mi
          limits:
            cpu: 75m
            memory: 50Mi
        resource:
          requests:
            cpu: 160m
            memory: 4800Mi
          limits:
            cpu: 230m
            memory: 7550Mi
      alertManagerConfig:
        resource:
          requests:
            cpu: 30m
            memory: 50Mi
          limits:
            cpu: 30m
            memory: 50Mi
      mcmMonitor:
        resource:
          requests:
            cpu: 30m
            memory: 50Mi
          limits:
            cpu: 50m
            memory: 50Mi
`
