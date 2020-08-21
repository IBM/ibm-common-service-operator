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
            memory: 290Mi
          requests:
            cpu: 30m
            memory: 230Mi
      certManagerController:
        resources:
          limits:
            cpu: 30m
            memory: 230Mi
          requests:
            cpu: 10m
            memory: 205Mi
      certManagerWebhook:
        resources:
          limits:
            cpu: 35m
            memory: 40Mi
          requests:
            cpu: 15m
            memory: 30Mi
      configMapWatcher:
        resources:
          limits:
            cpu: 10m
            memory: 25Mi
          requests:
            cpu: 10m
            memory: 20Mi
- name: ibm-mongodb-operator
  spec:
    mongoDB:
      replicas: 3
      resources:
        limits:
          cpu: 1200m
          memory: 2Gi
        requests:
          cpu: 1200m
          memory: 2Gi
- name: ibm-iam-operator
  spec:
    authentication:
      replicas: 1
      auditService:
        resources:
          limits:
            cpu: 20m
            memory: 20Mi
          requests:
            cpu: 20m
            memory: 20Mi
      authService:
        resources:
          limits:
            cpu: 650m
            memory: 745Mi
          requests:
            cpu: 230m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 20m
            memory: 20Mi
          requests:
            cpu: 20m
            memory: 20Mi
      identityManager:
        resources:
          limits:
            cpu: 250m
            memory: 185Mi
          requests:
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 485m
            memory: 305Mi
          requests:
            cpu: 320m
            memory: 250Mi
    oidcclientwatcher:
      replicas: 1
      resources:
        limits:
          cpu: 30m
          memory: 50Mi
        requests:
          cpu: 30m
          memory: 50Mi
    pap:
      auditService:
        resources:
          limits:
            cpu: 20m
            memory: 20Mi
          requests:
            cpu: 20m
            memory: 20Mi
      papService:
        resources:
          limits:
            cpu: 45m
            memory: 355Mi
          requests:
            cpu: 20m
            memory: 195Mi
      replicas: 1
    policycontroller:
      replicas: 1
      resources:
        limits:
          cpu: 20m
          memory: 30Mi
        requests:
          cpu: 20m
          memory: 20Mi
    policydecision:
      auditService:
        resources:
          limits:
            cpu: 20m
            memory: 20Mi
          requests:
            cpu: 20m
            memory: 20Mi
      pdpService:
        resources:
          limits:
            cpu: 30m
            memory: 30Mi
          requests:
            cpu: 20m
            memory: 30Mi
      replicas: 1
    secretwatcher:
      resources:
        limits:
          cpu: 30m
          memory: 220Mi
        requests:
          cpu: 30m
          memory: 220Mi
      replicas: 1
    securityonboarding:
      replicas: 1
      securityOnboardingJob:
        resources:
          limits:
            cpu: 20m
            memory: 20Mi
          requests:
            cpu: 20m
            memory: 20Mi
      iamOnboarding:
        resources:
          limits:
            cpu: 20m
            memory: 1024Mi
          requests:
            cpu: 20m
            memory: 64M
- name: ibm-management-ingress-operator
  spec:
    managementIngress:
      replicas: 1
      resources:
        requests:
          cpu: 50m
          memory: 125Mi
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
            memory: 225Mi
      defaultBackend:
        replicas: 1
        resources:
          requests:
            cpu: 20m
            memory: 30Mi
          limits:
            cpu: 20m
            memory: 30Mi
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
              memory: 230Mi
      reader:
        rdr:
          resources:
            limits:
              cpu: 30m
              memory: 290Mi
            requests:
              cpu: 25m
              memory: 230Mi
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
          cpu: 200m
          memory: 230Mi
        limits:
          cpu: 300m
          memory: 250Mi
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
          memory: 335Mi
          cpu: 300m
        limits:
          memory: 430Mi
          cpu: 300m
- name: ibm-platform-api-operator
  spec:
    platformApi:
      auditService:
        resources:
          limits:
            cpu: 25m
            memory: 30Mi
          requests:
            cpu: 25m
            memory: 30Mi
      platformApi:
        resources:
          limits:
            cpu: 25m
            memory: 25Mi
          requests:
            cpu: 25m
            memory: 25Mi
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
            cpu: 25m
            memory: 100Mi
          limits:
            cpu: 35m
            memory: 150Mi
- name: ibm-monitoring-exporters-operator
  spec:
    exporter:
      collectd:
        resources:
          requests:
            cpu: 30m
            memory: 30Mi
          limits:
            cpu: 30m
            memory: 30Mi
        routerResource:
          limits:
            cpu: 25m
            memory: 20Mi
          requests:
            cpu: 20m
            memory: 20Mi
      nodeExporter:
        resources:
          requests:
            cpu: 5m
            memory: 30Mi
          limits:
            cpu: 20m
            memory: 35Mi
        routerResource:
          requests:
            cpu: 20m
            memory: 20Mi
          limits:
            cpu: 25m
            memory: 20Mi
      kubeStateMetrics:
        resources:
          requests:
            cpu: 500m
            memory: 155Mi
          limits:
            cpu: 540m
            memory: 185Mi
        routerResource:
          limits:
            cpu: 25m
            memory: 20Mi
          requests:
            cpu: 20m
            memory: 20Mi
- name: ibm-monitoring-grafana-operator
  spec:
    grafana:
      resources:
        grafanaConfig:
          resources:
            requests:
              cpu: 25m
              memory: 65Mi
            limitis:
              cpu: 70m
              memory: 80Mi
        dashboardConfig:
          resources:
            requests:
              cpu: 25m
              memory: 65Mi
            limits:
              cpu: 70m
              memory: 80Mi
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
        resources:
          requests:
            cpu: 150m
            memory: 6190Mi
          limits:
            cpu: 230m
            memory: 7885Mi
      alertManagerConfig:
        resources:
          requests:
            cpu: 30m
            memory: 50Mi
          limits:
            cpu: 30m
            memory: 50Mi
      mcmMonitor:
        resources:
          requests:
            cpu: 30m
            memory: 30Mi
          limits:
            cpu: 30m
            memory: 30Mi
- name: ibm-elastic-stack-operator
  spec:
    elasticStack:
      curator:
        resources:
          limits:
            memory: 915Mi
          requests:
            memory: 2320Mi
        routerImage:
          resources:
            limits:
              memory: 256Mi
            requests:
              memory: 64Mi
      filebeat:
        resources:
          limits:
            memory: 125Mi
          requests:
            memory: 100Mi
      logstash:
        probe:
          resources:
            limits:
              memory: 1740Mi
            requests:
              memory: 1500Mi
        replicas: 1
`
