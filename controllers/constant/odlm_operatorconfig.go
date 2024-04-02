package constant

const PostGresOperatorConfig = `
apiVersion: operator.ibm.com/v1alpha1
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
  services:
    - name: edb-keycloak
      replicas: placeholder-size
      affinity:
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
              app.kubernetes.io/name: cloud-native-postgresql
    - name: cloud-native-postgresql
      replicas: placeholder-size
      affinity:
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
              app.kubernetes.io/name: cloud-native-postgresql
    - name: common-service-postgresql
      replicas: placeholder-size
      affinity:
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
              app.kubernetes.io/name: cloud-native-postgresql
`
