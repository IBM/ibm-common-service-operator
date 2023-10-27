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

package size

const Large = `
- name: ibm-cert-manager-operator
  spec:
    certManager:
      certManagerCAInjector:
        resources:
          limits:
            cpu: 200m
            memory: 814Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 40m
            memory: 581Mi
      certManagerController:
        resources:
          limits:
            cpu: 550m
            memory: 782Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 70m
            memory: 673Mi
      certManagerWebhook:
        resources:
          limits:
            cpu: 150m
            memory: 450Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 50m
            memory: 90Mi
- name: ibm-mongodb-operator
  spec:
    mongoDB:
      replicas: 3
      resources:
        limits:
          cpu: 3000m
          memory: 3072Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 3072Mi
      metrics:
        resources:
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 300Mi
          limits:
            cpu: 1000m
            memory: 350Mi
- name: ibm-im-mongodb-operator
  spec:
    mongoDB:
      replicas: 3
      resources:
        limits:
          cpu: 3000m
          memory: 3072Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 3072Mi
      metrics:
        resources:
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 300Mi
          limits:
            cpu: 1000m
            memory: 350Mi
- name: ibm-im-mongodb-operator-v4.0
  spec:
    mongoDB:
      replicas: 3
      resources:
        limits:
          cpu: 3000m
          memory: 3072Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 3072Mi
      metrics:
        resources:
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 300Mi
          limits:
            cpu: 1000m
            memory: 350Mi
- name: ibm-im-mongodb-operator-v4.1
  spec:
    mongoDB:
      replicas: 3
      resources:
        limits:
          cpu: 3000m
          memory: 3072Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 3072Mi
      metrics:
        resources:
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 300Mi
          limits:
            cpu: 1000m
            memory: 350Mi
- name: ibm-im-mongodb-operator-v4.2
  spec:
    mongoDB:
      replicas: 3
      resources:
        limits:
          cpu: 3000m
          memory: 3072Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 3072Mi
      metrics:
        resources:
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 300Mi
          limits:
            cpu: 1000m
            memory: 350Mi
- name: ibm-iam-operator
  spec:
    authentication:
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 400Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 75m
            memory: 50Mi
      authService:
        resources:
          limits:
            cpu: 3000m
            memory: 1201Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 725m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 300Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 645Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 340m
            memory: 385Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 480Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 410m
            memory: 335Mi
      replicas: 3
    oidcclientwatcher:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 325Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 30m
          memory: 67Mi
    pap:
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 70Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 75m
            memory: 50Mi
      papService:
        resources:
          limits:
            cpu: 1000m
            memory: 943Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 50m
            memory: 195Mi
      replicas: 3
    policycontroller:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 450Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 20m
          memory: 75Mi
    policydecision:
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 84Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 30m
            memory: 50Mi
      replicas: 3
      resources:
        limits:
          cpu: 1000m
          memory: 420Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 195m
          memory: 270Mi
    secretwatcher:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 336Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 30m
          memory: 220Mi
    securityonboarding:
      iamOnboarding:
        resources:
          limits:
            cpu: 1000m
            memory: 1024Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 64Mi
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 50Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 20m
          memory: 50Mi
- name: ibm-im-operator
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 3000m
            memory: 1201Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 725m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 300Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 645Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 340m
            memory: 385Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 480Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 410m
            memory: 335Mi
      replicas: 3
- name: ibm-im-operator-v4.0
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 3000m
            memory: 1201Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 725m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 300Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 645Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 340m
            memory: 385Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 480Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 410m
            memory: 335Mi
      replicas: 3
- name: ibm-im-operator-v4.1
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 3000m
            memory: 1201Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 725m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 300Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 645Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 340m
            memory: 385Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 480Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 410m
            memory: 335Mi
      replicas: 3
- name: ibm-im-operator-v4.2
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 3000m
            memory: 1201Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 725m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 300Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 645Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 340m
            memory: 385Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 480Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 410m
            memory: 335Mi
      replicas: 3
- name: ibm-management-ingress-operator
  spec:
    managementIngress:
      replicas: 3
      resources:
        limits:
          cpu: 1000m
          memory: 1288Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 200m
          memory: 442Mi
- name: ibm-ingress-nginx-operator
  spec:
    nginxIngress:
      defaultBackend:
        replicas: 1
        resources:
          limits:
            cpu: 50m
            memory: 183Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 30m
            memory: 116Mi
      ingress:
        replicas: 3
        resources:
          limits:
            cpu: 1000m
            memory: 1188Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 200m
            memory: 512Mi
      kubectl:
        resources:
          limits:
            cpu: 150m
            memory: 495Mi
          requests:
            ephemeral-storage: 256Mi
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
            ephemeral-storage: 256Mi
            cpu: 200m
            memory: 256Mi
      receiverContainer:
        resources:
          limits:
            cpu: 300m
            memory: 384Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 200m
            memory: 256Mi
    IBMLicensing:
      resources:
        limits:
          cpu: 400m
          memory: 634Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 270Mi
- name: ibm-commonui-operator
  spec:
    commonWebUI:
      replicas: 3
      resources:
        limits:
          cpu: 1000m
          memory: 1225Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 384Mi
- name: ibm-idp-config-ui-operator
  spec:
    commonWebUI:
      replicas: 3
      resources:
        limits:
          cpu: 1000m
          memory: 1225Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 384Mi
- name: ibm-idp-config-ui-operator-v4.0
  spec:
    commonWebUI:
      replicas: 3
      resources:
        limits:
          cpu: 1000m
          memory: 1225Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 384Mi
- name: ibm-idp-config-ui-operator-v4.1
  spec:
    commonWebUI:
      replicas: 3
      resources:
        limits:
          cpu: 1000m
          memory: 1225Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 384Mi
- name: ibm-idp-config-ui-operator-v4.2
  spec:
    commonWebUI:
      replicas: 3
      resources:
        limits:
          cpu: 1000m
          memory: 1225Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 384Mi
- name: ibm-platform-api-operator
  spec:
    platformApi:
      auditService:
        resources:
          limits:
            cpu: 25m
            memory: 300Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 25m
            memory: 50Mi
      platformApi:
        resources:
          limits:
            cpu: 25m
            memory: 117Mi
          requests:
            ephemeral-storage: 256Mi
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
            ephemeral-storage: 256Mi
            cpu: 27m
            memory: 153Mi
      memcached:
        replicas: 1
        resources:
          limits:
            cpu: 200m
            memory: 100Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
- name: ibm-auditlogging-operator
  spec:
    auditLogging:
      fluentd:
        resources:
          limits:
            cpu: 75m
            memory: 375Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 59m
            memory: 231Mi
- name: ibm-monitoring-grafana-operator
  spec:
    grafana:
      dashboardConfig:
        resources:
          limits:
            cpu: 515m
            memory: 412Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 25m
            memory: 145Mi
      grafanaConfig:
        resources:
          limits:
            cpu: 300m
            memory: 419Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 30m
            memory: 195Mi
      routerConfig:
        resources:
          limits:
            cpu: 70m
            memory: 344Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 25m
            memory: 65Mi
- name: ibm-apicatalog
  spec:
    apicatalogmanager:
      profile: large
- name: edb-keycloak
  spec:
    Cluster:
      instances: 3
      resources:
        requests:
          cpu: 1000m
          memory: 1Gi
        limits:
          cpu: 1000m
          memory: 1Gi
`
