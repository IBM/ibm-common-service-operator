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

package rules

// ConfigurationRules is a yaml defines the rule of patching paramaters
const ConfigurationRules = `
- name: ibm-cert-manager-operator
  spec:
    certManager:
      certManagerCAInjector:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      certManagerController:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      certManagerWebhook:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
- name: ibm-mongodb-operator
  spec:
    mongoDB:
      replicas: LARGEST_VALUE
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
      metrics:
        resources:
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
- name: ibm-iam-operator
  spec:
    authentication:
      config:
        fipsEnabled: LARGEST_VALUE
      replicas: LARGEST_VALUE
      auditService:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      authService:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      clientRegistration:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      identityManager:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      identityProvider:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
    oidcclientwatcher:
      replicas: LARGEST_VALUE
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
    pap:
      auditService:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      papService:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      replicas: LARGEST_VALUE
    policycontroller:
      replicas: LARGEST_VALUE
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
    policydecision:
      auditService:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
      replicas: LARGEST_VALUE
    secretwatcher:
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
      replicas: LARGEST_VALUE
    securityonboarding:
      replicas: LARGEST_VALUE
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
      iamOnboarding:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
- name: ibm-im-operator
  spec:
    authentication:
      config:
        fipsEnabled: LARGEST_VALUE
      replicas: LARGEST_VALUE
      auditService:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      authService:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      clientRegistration:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      identityManager:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      identityProvider:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      replicas: LARGEST_VALUE
    policycontroller:
      replicas: LARGEST_VALUE
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
    policydecision:
      auditService:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
      replicas: LARGEST_VALUE
    secretwatcher:
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
      replicas: LARGEST_VALUE
    securityonboarding:
      replicas: LARGEST_VALUE
      resources:
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
- name: ibm-management-ingress-operator
  spec:
    managementIngress:
      fipsEnabled: LARGEST_VALUE
      replicas: LARGEST_VALUE
      resources:
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
- name: ibm-ingress-nginx-operator
  spec:
    nginxIngress:
      fips_enabled: LARGEST_VALUE
      ingress:
        replicas: LARGEST_VALUE
        resources:
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      defaultBackend:
        replicas: LARGEST_VALUE
        resources:
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      kubectl:
        resources:
          requests:
            memory: LARGEST_VALUE
            cpu: LARGEST_VALUE
          limits:
            memory: LARGEST_VALUE
            cpu: LARGEST_VALUE
- name: ibm-licensing-operator
  spec:
    IBMLicensing:
      resources:
        requests:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
        limits:
          cpu: LARGEST_VALUE
          memory: LARGEST_VALUE
    IBMLicenseServiceReporter:
      databaseContainer:
        resources:
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      receiverContainer:
        resources:
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
- name: ibm-commonui-operator
  spec:
    commonWebUI:
      replicas: LARGEST_VALUE
      resources:
        requests:
          memory: LARGEST_VALUE
          cpu: LARGEST_VALUE
        limits:
          memory: LARGEST_VALUE
          cpu: LARGEST_VALUE
      commonWebUIConfig:
        dashboardData:
          resources:
            limits:
              cpu: LARGEST_VALUE
              memory: LARGEST_VALUE
            requests:
              cpu: LARGEST_VALUE
              memory: LARGEST_VALUE
- name: ibm-platform-api-operator
  spec:
    platformApi:
      auditService:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      platformApi:
        resources:
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      replicas: LARGEST_VALUE
- name: ibm-healthcheck-operator
  spec:
    healthService:
      memcached:
        replicas: LARGEST_VALUE
        resources:
          requests:
            memory: LARGEST_VALUE
            cpu: LARGEST_VALUE
          limits:
            memory: LARGEST_VALUE
            cpu: LARGEST_VALUE
      healthService:
        replicas: LARGEST_VALUE
        resources:
          requests:
            memory: LARGEST_VALUE
            cpu: LARGEST_VALUE
          limits:
            memory: LARGEST_VALUE
            cpu: LARGEST_VALUE
- name: ibm-auditlogging-operator
  spec:
    auditLogging:
      fluentd:
        resources:
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
- name: ibm-monitoring-grafana-operator
  spec:
    grafana:
      grafanaConfig:
        resources:
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      dashboardConfig:
        resources:
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
      routerConfig:
        resources:
          requests:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
          limits:
            cpu: LARGEST_VALUE
            memory: LARGEST_VALUE
- name: ibm-apicatalog
  spec:
    apicatalogmanager:
      profile: LARGEST_VALUE
`
