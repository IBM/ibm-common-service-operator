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

const Large = `
- name: ibm-healthcheck-operator
  spec:
    healthService:
      spec:
        memcached:
          replicas: 3
          resources:
            requests:
              memory: "64Mi"
              cpu: "50m"
            limits:
              memory: "512Mi"
              cpu: "500m"
        healthService:
          replicas: 1
          resources:
            requests:
              memory: "64Mi"
              cpu: "50m"
            limits:
              memory: "512Mi"
              cpu: "500m"
- name: ibm-iam-operator
  spec:
    authentication:
      spec:
        replicas: 1
        auditService:
          resources:
            limits:
              cpu: 100m
              memory: 128Mi
            requests:
              cpu: 10m
              memory: 100Mi
        authService:
          resources:
            limits:
              cpu: 1000m
              memory: 1Gi
            requests:
              cpu: 100m
              memory: 350Mi
        clientRegistration:
          resources:
            limits:
              cpu: 1000m
              memory: 1Gi
            requests:
              cpu: 100m
              memory: 128Mi
        identityManager:
          resources:
            limits:
              cpu: 1000m
              memory: 1Gi
            requests:
              cpu: 50m
              memory: 150Mi
        identityProvider:
          resources:
            limits:
              cpu: 1000m
              memory: 1Gi
            requests:
              cpu: 50m
              memory: 150Mi
`
