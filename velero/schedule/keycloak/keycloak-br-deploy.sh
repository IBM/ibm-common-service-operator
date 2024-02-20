#!/usr/bin/env bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product.
# Please refer to that particular license for additional information.

# ---------- Command arguments ----------

set -o errtrace
set -o errexit

CLEANUP="false"
STORAGE_CLASS="default"

function main() {
  parse_arguments "$@"
  if [[ $CLEANUP == "true" ]]; then
    cleanup
  else
    deploy_resources
  fi
}

function parse_arguments() {
  # process options
  while [[ "$@" != "" ]]; do
    case "$1" in
    --keycloak-ns)
      shift
      KEYCLOAK_NAMESPACE=$1
      ;;
    --storage-class)
      shift
      STORAGE_CLASS=$1
      ;;
    "-c" | "--cleanup")
      CLEANUP="true"
      ;;
    -h | --help)
      print_usage
      exit 1
      ;;
    *) 
      warning "$1 not a supported parameter for keycloak-deploy.sh"
      ;;
    esac
    shift
  done
}

function print_usage() {
  echo "Usage: ${script_name} --keycloak-ns <Namespace where keycloak is installed> [OPTIONS]..."
  echo ""
  echo "Deploy the necessary resources for Backup of Keycloak."
  #TODO change below to point to correct docs
  #echo "See step 4 here https://www.ibm.com/docs/en/cloud-paks/foundational-services/4.0?topic=4x-isolated-migration for more information."
  echo ""
  echo "Options:"
  echo "   --keycloak-ns string                           Required. Namespace where Keycloak is installed."
  echo "   --storage-class string                         Optional. Storage class used by keycloak. Default value is cluster's default storage class."
  echo "   -c, --cleanup                                  Optional. Automated cleanup of Keycloak BR resources. Will run cleanup instead of deployment logic."
  echo "   -h, --help                                     Print usage information"
  echo ""
}

function deploy_resources(){
  info "Creating Keycloak Backup/Restore resources"
  cat << EOF | oc apply -f -
kind: Deployment
apiVersion: apps/v1
metadata:
  name: keycloak-backup
  namespace: $KEYCLOAK_NAMESPACE
  labels:
    foundationservices.cloudpak.ibm.com: keycloak-data
spec:
  selector:
    matchLabels:
      foundationservices.cloudpak.ibm.com: keycloak-data
  template:
    metadata:
      annotations:
        backup.velero.io/backup-volumes: keycloak-backup
        pre.hook.backup.velero.io/command: '["sh", "-c", "/keycloak/br_keycloak.sh backup $KEYCLOAK_NAMESPACE"]'
        pre.hook.backup.velero.io/timeout: 300s
        post.hook.backup.velero.io/command: '["sh", "-c", "rm -rf /keycloak/keycloak-backup/database"]'
        post.hook.restore.velero.io/command: '["sh", "-c", "/keycloak/br_keycloak.sh restore $KEYCLOAK_NAMESPACE"]'
        post.hook.restore.velero.io/wait-timeout: 600s
        post.hook.restore.velero.io/exec-timeout: 600s
        post.hook.restore.velero.io/timeout: 720s
      name: keycloak-backup
      namespace: $KEYCLOAK_NAMESPACE
      labels:
        foundationservices.cloudpak.ibm.com: keycloak-data
    spec:
        containers:
        - command:
          - sh
          - -c
          - sleep infinity
          image: icr.io/cpopen/cpfs/cpfs-utils:4.4.0 #4.1.0 if using CS 4.1, 4.2.0 if using CS 4.2
          imagePullPolicy: IfNotPresent
          name: keycloak-br
          resources:
            limits:
              cpu: 500m
              ephemeral-storage: 512Mi
              memory: 512Mi
            requests:
              cpu: 200m
              ephemeral-storage: 128Mi
              memory: 256Mi
          terminationMessagePath: /dev/termination-log
          terminationMessagePolicy: File
          volumeMounts:
          - mountPath: /keycloak/keycloak-backup
            name: keycloak-backup-mount
          - name: scripts
            mountPath: "/keycloak"
          - name: secrets
            mountPath: /keycloak/keycloak-backup/secrets
        dnsPolicy: ClusterFirst
        schedulerName: default-scheduler
        securityContext:
          runAsNonRoot: true
        serviceAccount: keycloak-backup-sa
        serviceAccountName: keycloak-backup-sa
        terminationGracePeriodSeconds: 30
        volumes:
        - name: keycloak-backup-mount
          persistentVolumeClaim:
            claimName: keycloak-backup-pvc
        - name: scripts
          configMap:
            name: keycloak-br-configmap
            defaultMode: 0777
        - emptyDir: {}
          name: secrets
EOF
  if [[ $STORAGE_CLASS == "default" ]]; then
    STORAGE_CLASS=$(oc get sc | grep default | awk '{print $1}')
    info "Using default storage class $STORAGE_CLASS."
  else
    info "Using specified storage class $STORAGE_CLASS."
  fi

  cat << EOF | oc apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: keycloak-backup-pvc
  namespace: $KEYCLOAK_NAMESPACE
  labels:
    foundationservices.cloudpak.ibm.com: keycloak-data
spec:
  storageClassName: $STORAGE_CLASS
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  volumeMode: Filesystem
EOF

  oc apply -f keycloak-br-script-cm.yaml -n $KEYCLOAK_NAMESPACE

  cat << EOF | oc apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: keycloak-backup-role
  namespace: $KEYCLOAK_NAMESPACE
  labels:
    foundationservices.cloudpak.ibm.com: keycloak-data
rules:
  - verbs:
      - create
      - get
      - delete
      - watch
      - update
      - list
      - patch
    apiGroups:
      - ''
      - batch
      - extensions
      - apps
      - policy
    resources:
      - pods
      - pods/log
      - deployments
      - deployments/scale
      - statefulsets
      - statefulsets/scale
      - pods/exec
      - pods/portforward
      - endpoints
      - pods/status
      - jobs
      - secrets
  - verbs:
      - get
      - list
    apiGroups:
      - postgresql.k8s.enterprisedb.io
    resources:
      - clusters
EOF

  cat << EOF | oc apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: keycloak-backup-rolebinding
  namespace: $KEYCLOAK_NAMESPACE
  labels:
    foundationservices.cloudpak.ibm.com: keycloak-data
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: keycloak-backup-role
  namespace: $KEYCLOAK_NAMESPACE
subjects:
- kind: ServiceAccount
  name: keycloak-backup-sa
  namespace: $KEYCLOAK_NAMESPACE
EOF

  cat << EOF | oc apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: keycloak-backup-sa
  namespace: $KEYCLOAK_NAMESPACE
  labels:
    foundationservices.cloudpak.ibm.com: keycloak-data
EOF

  success "Backup/Restore resources created."

}

function cleanup() {
  info "Clean up Keycloak BR resources..."
  if [[ $MODE == "restore" ]]; then
    rs=$(oc get rs -n $KEYCLOAK_NAMESPACE --no-headers --ignore-not-found | grep keycloak-backup | awk '{print $1}' | tr "\n" " ")
  fi
  oc delete deploy keycloak-backup -n $KEYCLOAK_NAMESPACE --ignore-not-found && oc delete rs $rs -n $KEYCLOAK_NAMESPACE --ignore-not-found && oc delete sa keycloak-backup-sa -n $KEYCLOAK_NAMESPACE --ignore-not-found && oc delete role keycloak-backup-role -n $KEYCLOAK_NAMESPACE --ignore-not-found && oc delete rolebinding keycloak-backup-rolebinding -n $KEYCLOAK_NAMESPACE --ignore-not-found && oc delete pvc keycloak-backup-pvc -n $KEYCLOAK_NAMESPACE --ignore-not-found
  success "Keycloak BR resources cleaned up."
}

function msg() {
  printf '%b\n' "$1"
}

function success() {
  msg "\33[32m[✔] ${1}\33[0m"
}

function warning() {
  msg "\33[33m[✗] ${1}\33[0m"
}

function error() {
  msg "\33[31m[✘] ${1}\33[0m"
  exit 1
}

function title() {
  msg "\33[34m# ${1}\33[0m"
}

function info() {
  msg "[INFO] ${1}"
}

main $*