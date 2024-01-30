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
            ephemeral-storage: 256Mi
            cpu: 40m
            memory: 581Mi
      certManagerController:
        resources:
          limits:
            cpu: 110m
            memory: 782Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 70m
            memory: 673Mi
      certManagerWebhook:
        resources:
          limits:
            cpu: 60m
            memory: 100Mi
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
          cpu: 2000m
          memory: 2048Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 2048Mi
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
          cpu: 2000m
          memory: 2048Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 2048Mi
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
          cpu: 2000m
          memory: 2048Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 2048Mi
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
          cpu: 2000m
          memory: 2048Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 2048Mi
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
          cpu: 2000m
          memory: 2048Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 500m
          memory: 2048Mi
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
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 50m
            memory: 50Mi
      authService:
        resources:
          limits:
            cpu: 1000m
            memory: 745Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 230m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 525Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 355Mi
          requests:
            ephemeral-storage: 256Mi
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
          ephemeral-storage: 256Mi
          cpu: 30m
          memory: 67Mi
    pap:
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 50m
            memory: 50Mi
      papService:
        resources:
          limits:
            cpu: 1000m
            memory: 600Mi
          requests:
            ephemeral-storage: 256Mi
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
          ephemeral-storage: 256Mi
          cpu: 20m
          memory: 75Mi
    policydecision:
      auditService:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 85Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 20m
          memory: 50Mi
    secretwatcher:
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 220Mi
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
            cpu: 1000m
            memory: 745Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 230m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 525Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 355Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 320m
            memory: 250Mi
      replicas: 2
- name: ibm-im-operator-v4.0
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 1000m
            memory: 745Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 230m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 525Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 355Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 320m
            memory: 250Mi
      replicas: 2
- name: ibm-im-operator-v4.1
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 1000m
            memory: 745Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 230m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 525Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 355Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 320m
            memory: 250Mi
      replicas: 2
- name: ibm-im-operator-v4.2
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 1000m
            memory: 745Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 230m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 525Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 355Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 320m
            memory: 250Mi
      replicas: 2
- name: ibm-im-operator-v4.3
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 1000m
            memory: 745Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 230m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 525Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 355Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 320m
            memory: 250Mi
      replicas: 2
- name: ibm-im-operator-v4.4
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 1000m
            memory: 745Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 230m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 525Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 355Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 320m
            memory: 250Mi
      replicas: 2
- name: ibm-im-operator-v4.5
  spec:
    authentication:
      authService:
        resources:
          limits:
            cpu: 1000m
            memory: 745Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 230m
            memory: 695Mi
      clientRegistration:
        resources:
          limits:
            cpu: 1000m
            memory: 50Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 20m
            memory: 50Mi
      identityManager:
        resources:
          limits:
            cpu: 1000m
            memory: 525Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 100m
            memory: 140Mi
      identityProvider:
        resources:
          limits:
            cpu: 1000m
            memory: 355Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 320m
            memory: 250Mi
      replicas: 2
- name: ibm-management-ingress-operator
  spec:
    managementIngress:
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 1024Mi
        requests:
          ephemeral-storage: 256Mi
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
            ephemeral-storage: 256Mi
            cpu: 30m
            memory: 64Mi
      ingress:
        replicas: 2
        resources:
          limits:
            cpu: 1000m
            memory: 1024Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 200m
            memory: 256Mi
      kubectl:
        resources:
          limits:
            cpu: 100m
            memory: 350Mi
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
          cpu: 300m
          memory: 350Mi
        requests:
          ephemeral-storage: 256Mi
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
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 376Mi
- name: ibm-idp-config-ui-operator
  spec:
    commonWebUI:
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 430Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 376Mi
- name: ibm-idp-config-ui-operator-v4.0
  spec:
    commonWebUI:
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 430Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 376Mi
- name: ibm-idp-config-ui-operator-v4.1
  spec:
    commonWebUI:
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 430Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 376Mi
- name: ibm-idp-config-ui-operator-v4.2
  spec:
    commonWebUI:
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 430Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 376Mi
- name: ibm-idp-config-ui-operator-v4.3
  spec:
    commonWebUI:
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 430Mi
        requests:
          ephemeral-storage: 256Mi
          cpu: 300m
          memory: 376Mi
- name: ibm-idp-config-ui-operator-v4.4
  spec:
    commonWebUI:
      replicas: 2
      resources:
        limits:
          cpu: 1000m
          memory: 430Mi
        requests:
          ephemeral-storage: 256Mi
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
            ephemeral-storage: 256Mi
            cpu: 25m
            memory: 50Mi
      platformApi:
        resources:
          limits:
            cpu: 25m
            memory: 100Mi
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
            cpu: 20m
            memory: 125Mi
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
            cpu: 50m
            memory: 200Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 35m
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
            ephemeral-storage: 256Mi
            cpu: 25m
            memory: 93Mi
      grafanaConfig:
        resources:
          limits:
            cpu: 150m
            memory: 148Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 25m
            memory: 87Mi
      routerConfig:
        resources:
          limits:
            cpu: 70m
            memory: 80Mi
          requests:
            ephemeral-storage: 256Mi
            cpu: 25m
            memory: 65Mi
- name: ibm-apicatalog
  spec:
    apicatalogmanager:
      profile: medium
- name: edb-keycloak
  resources:
  - apiVersion: postgresql.k8s.enterprisedb.io/v1
    kind: Cluster
    name: keycloak-edb-cluster
    data:
      spec:
        instances: 2
        resources:
          limits:
            cpu: 500m
            memory: 1024Mi
          requests:
            cpu: 500m
            memory: 1024Mi
- name: keycloak-operator
  resources:
  - apiVersion: k8s.keycloak.org/v2alpha1
    kind: Keycloak
    name: cs-keycloak
    data:
      spec:
        instances: 2
        unsupported:
          podTemplate:
            spec:
              containers:
                - resources:
                    limits:
                      cpu: 1000m
                      memory: 1Gi
                    requests:
                      cpu: 1000m
                      memory: 1Gi
`
