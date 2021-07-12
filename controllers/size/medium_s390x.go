//
// Copyright 2021 IBM Corporation
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
            cpu: 100m
            memory: 770Mi
          requests:
            cpu: 40m
            memory: 581Mi
      certManagerController:
        resources:
          limits:
            cpu: 110m
            memory: 782Mi
          requests:
            cpu: 70m
            memory: 673Mi
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
            memory: 150Mi
          requests:
            cpu: 10m
            memory: 80Mi
- name: ibm-mongodb-operator
  spec:
    mongoDB:
      replicas: 3
      resources:
        limits:
          cpu: 2000m
          memory: 2048Mi
        requests:
          cpu: 500m
          memory: 2048Mi
- name: ibm-iam-operator
  spec:
    authentication:
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
            cpu: 230m
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
            memory: 525Mi
          requests:
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 355Mi
          requests:
            cpu: 320m
            memory: 250Mi
      replicas: 2
    oidcclientwatcher:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 325Mi
        requests:
          cpu: 30m
          memory: 67Mi
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
          memory: 75Mi
        requests:
          cpu: 20m
          memory: 75Mi
    policydecision:
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            cpu: 20m
            memory: 50Mi
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 85Mi
        requests:
          cpu: 20m
          memory: 50Mi
    secretwatcher:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 220Mi
        requests:
          cpu: 30m
          memory: 220Mi
    securityonboarding:
      iamOnboarding:
        resources:
          limits:
            cpu: 1000m
            memory: 1024Mi
          requests:
            cpu: 20m
            memory: 64Mi
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 50Mi
        requests:
          cpu: 20m
          memory: 50Mi
- name: ibm-management-ingress-operator
  spec:
    managementIngress:
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 1024Mi
        requests:
          cpu: 200m
          memory: 256Mi
- name: ibm-ingress-nginx-operator
  spec:
    nginxIngress:
      defaultBackend:
        replicas: 1
        resources:
          limits:
            cpu: 50m
            memory: 128Mi
          requests:
            cpu: 30m
            memory: 64Mi
      ingress:
        replicas: 2
        resources:
          limits:
            cpu: 1000m
            memory: 1024Mi
          requests:
            cpu: 200m
            memory: 256Mi
      kubectl:
        resources:
          limits:
            cpu: 100m
            memory: 350Mi
          requests:
            cpu: 50m
            memory: 150Mi
- name: ibm-licensing-operator
  spec:
    IBMLicenseServiceReporter:
      databaseContainer:
        resources:
          limits:
            cpu: 300m
            memory: 300Mi
          requests:
            cpu: 200m
            memory: 256Mi
      receiverContainer:
        resources:
          limits:
            cpu: 300m
            memory: 384Mi
          requests:
            cpu: 200m
            memory: 256Mi
    IBMLicensing:
      resources:
        limits:
          cpu: 300m
          memory: 350Mi
        requests:
          cpu: 200m
          memory: 230Mi
- name: ibm-commonui-operator
  spec:
    commonWebUI:
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 430Mi
        requests:
          cpu: 300m
          memory: 376Mi
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
            memory: 100Mi
          requests:
            cpu: 25m
            memory: 67Mi
      replicas: 1
- name: ibm-healthcheck-operator
  spec:
    healthService:
      healthService:
        replicas: 1
        resources:
          limits:
            cpu: 200m
            memory: 250Mi
          requests:
            cpu: 20m
            memory: 125Mi
      memcached:
        replicas: 1
        resources:
          limits:
            cpu: 200m
            memory: 100Mi
          requests:
            cpu: 20m
            memory: 50Mi
- name: ibm-auditlogging-operator
  spec:
    auditLogging:
      fluentd:
        resources:
          limits:
            cpu: 50m
            memory: 200Mi
          requests:
            cpu: 35m
            memory: 128Mi
- name: ibm-monitoring-exporters-operator
  spec:
    exporter:
      collectd:
        resource:
          limits:
            cpu: 30m
            memory: 50Mi
          requests:
            cpu: 30m
            memory: 50Mi
        routerResource:
          limits:
            cpu: 25m
            memory: 50Mi
          requests:
            cpu: 20m
            memory: 50Mi
      kubeStateMetrics:
        resource:
          limits:
            cpu: 540m
            memory: 185Mi
          requests:
            cpu: 500m
            memory: 155Mi
        routerResource:
          limits:
            cpu: 25m
            memory: 50Mi
          requests:
            cpu: 20m
            memory: 50Mi
      nodeExporter:
        resource:
          limits:
            cpu: 20m
            memory: 67Mi
          requests:
            cpu: 5m
            memory: 67Mi
        routerResource:
          limits:
            cpu: 100m
            memory: 256Mi
          requests:
            cpu: 50m
            memory: 128Mi
- name: ibm-monitoring-grafana-operator
  spec:
    grafana:
      dashboardConfig:
        resources:
          limits:
            cpu: 70m
            memory: 123Mi
          requests:
            cpu: 25m
            memory: 93Mi
      grafanaConfig:
        resources:
          limits:
            cpu: 150m
            memory: 148Mi
          requests:
            cpu: 25m
            memory: 87Mi
      routerConfig:
        resources:
          limits:
            cpu: 70m
            memory: 80Mi
          requests:
            cpu: 25m
            memory: 65Mi
- name: ibm-monitoring-prometheusext-operator
  spec:
    prometheusExt:
      alertManagerConfig:
        resource:
          limits:
            cpu: 30m
            memory: 50Mi
          requests:
            cpu: 30m
            memory: 50Mi
      mcmMonitor:
        resource:
          limits:
            cpu: 50m
            memory: 50Mi
          requests:
            cpu: 30m
            memory: 50Mi
      prometheusConfig:
        resource:
          limits:
            cpu: 230m
            memory: 7885Mi
          requests:
            cpu: 150m
            memory: 6190Mi
        routerResource:
          limits:
            cpu: 75m
            memory: 50Mi
          requests:
            cpu: 10m
            memory: 50Mi
`
