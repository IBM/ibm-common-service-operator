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

const Small = `
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
            cpu: 110m
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
      replicas: 1
      resources:
        limits:
          cpu: 1500m
          memory: 1Gi
        requests:
          cpu: 1500m
          memory: 1Gi
- name: ibm-iam-operator
  spec:
    authentication:
      replicas: 1
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            cpu: 20m
            memory: 50Mi
      authService:
        resources:
          limits:
            cpu: 1000m
            memory: 650Mi
          requests:
            cpu: 140m
            memory: 525Mi
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
            memory: 220Mi
          requests:
            cpu: 50m
            memory: 120Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 230Mi
          requests:
            cpu: 80m
            memory: 130Mi
    oidcclientwatcher:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 256Mi
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
            cpu: 20m
            memory: 50Mi
      papService:
        resources:
          limits:
            cpu: 1000m
            memory: 330Mi
          requests:
            cpu: 50m
            memory: 160Mi
      replicas: 1
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
          memory: 50Mi
        requests:
          cpu: 20m
          memory: 50Mi
      replicas: 1
    secretwatcher:
      resources:
        limits:
          cpu: 1000m
          memory: 145Mi
        requests:
          cpu: 30m
          memory: 120Mi
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
      replicas: 1
      resources:
        requests:
          cpu: 50m
          memory: 100Mi
        limits:
          cpu: 50m
          memory: 170Mi
- name: ibm-ingress-nginx-operator
  spec:
    nginxIngress:
      ingress:
        replicas: 1
        resources:
          requests:
            cpu: 100m
            memory: 140Mi
          limits:
            cpu: 100m
            memory: 350Mi
      defaultBackend:
        replicas: 1
        resources:
          requests:
            cpu: 20m
            memory: 50Mi
          limits:
            cpu: 20m
            memory: 50Mi
      kubectl:
        resources:
          requests:
            memory: 150Mi
            cpu: 30m
          limits:
            memory: 150Mi
            cpu: 30m
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
              memory: 140Mi
      reader:
        rdr:
          resources:
            limits:
              cpu: 50m
              memory: 200Mi
            requests:
              cpu: 25m
              memory: 175Mi
    meteringReportServer:
      reportServer:
        resources:
          limits:
            cpu: 50m
            memory: 50Mi
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
          cpu: 100m
          memory: 220Mi
        limits:
          cpu: 200m
          memory: 320Mi
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
      replicas: 1
      resources:
        requests:
          memory: 256Mi
          cpu: 150m
        limits:
          memory: 310Mi
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
            memory: 220Mi
          requests:
            cpu: 35m
            memory: 105Mi
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
            memory: 50Mi
            cpu: 20m
          limits:
            memory: 100Mi
            cpu: 200m
- name: ibm-auditlogging-operator
  spec:
    auditLogging:
      fluentd:
        resources:
          requests:
            cpu: 25m
            memory: 100Mi
          limits:
            cpu: 35m
            memory: 150Mi
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
            cpu: 10m
            memory: 50Mi
      nodeExporter:
        resource:
          requests:
            cpu: 5m
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
            memory: 110Mi
          limits:
            cpu: 540m
            memory: 160Mi
        routerResource:
          limits:
            cpu: 25m
            memory: 50Mi
          requests:
            cpu: 10m
            memory: 50Mi
- name: ibm-monitoring-grafana-operator
  spec:
    grafana:
      grafanaConfig:
        resources:
          requests:
            cpu: 20m
            memory: 65Mi
          limits:
            cpu: 150m
            memory: 75Mi
      dashboardConfig:
        resources:
          requests:
            cpu: 5m
            memory: 50Mi
          limits:
            cpu: 20m
            memory: 60Mi
      routerConfig:
        resources:
          requests:
            cpu: 10m
            memory: 50Mi
          limits:
            cpu: 50m
            memory: 50Mi
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
            cpu: 65m
            memory: 1920Mi
          limits:
            cpu: 130m
            memory: 2570Mi
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
            cpu: 30m
            memory: 50Mi
`
