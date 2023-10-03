#!/bin/bash
# Used by multi-namespace
CS_NAMESPACE=$1
CONVERT=""
if [[ ! -z $2 ]]; then
  CONVERT=$2
fi
s390x="false"
if [[ ! -z $3 ]]; then
  s390x=$3
fi
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

function cleanup() {
  if [[ -z $CS_NAMESPACE ]]; then
    export CS_NAMESPACE=ibm-common-services
  fi
  msg "[1] Cleaning up from previous backups..."
  oc delete job mongodb-backup --ignore-not-found -n $CS_NAMESPACE
  pv=$(oc get pvc cs-mongodump -n $CS_NAMESPACE --no-headers=true 2>/dev/null | awk '{print $3 }')
  if [[ -n $pv ]]
  then
    oc delete pvc cs-mongodump --ignore-not-found -n $CS_NAMESPACE
    oc delete pv $pv --ignore-not-found
  fi
  success "Cleanup Complete"
}

function backup_mongodb(){
  msg "[3] Backing Up MongoDB"
  #
  #  Get the storage class from the existing PVCs for use in creating the backup volume
  #
  SAMPLEPV=$(oc get pvc -n $CS_NAMESPACE | grep mongodb | awk '{ print $3 }')
  SAMPLEPV=$( echo $SAMPLEPV | awk '{ print $1 }' )
  #STGCLASS=$(oc get pvc --no-headers=true mongodbdir-icp-mongodb-0 -n $CS_NAMESPACE | awk '{ print $6 }')
  STGCLASS=ibmc-block-retain-gold
  # Used by multi-namespace
  if [[ $CONVERT != "" ]]; then
    STGCLASS=$CONVERT
  fi
  #
  # Backup MongoDB
  #
  cat <<EOF | oc apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: cs-mongodump
  namespace: $CS_NAMESPACE
  labels:
    foundationservices.cloudpak.ibm.com: data
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: $STGCLASS
EOF

  #
  # Start the backup
  #
  msg "Starting backup"
  ibm_mongodb_image=$(oc get pod icp-mongodb-0 -n $CS_NAMESPACE -o=jsonpath='{range .spec.containers[0]}{.image}{end}')
  if [[ $s390x == "false" ]]; then
    cat <<EOF | oc apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: mongodb-backup
  namespace: $CS_NAMESPACE
spec:
  parallelism: 1
  completions: 1
  backoffLimit: 20
  template:
    spec:
      containers:
      - name: cs-mongodb-backup
        image: $ibm_mongodb_image
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 100m
            memory: 128Mi
        command: ["bash", "-c", "cat /cred/mongo-certs/tls.crt /cred/mongo-certs/tls.key > /work-dir/mongo.pem; cat /cred/cluster-ca/tls.crt /cred/cluster-ca/tls.key > /work-dir/ca.pem; mongodump --oplog --out /dump/dump --host mongodb:27017 --username \$ADMIN_USER --password \$ADMIN_PASSWORD --authenticationDatabase admin --ssl --sslCAFile /work-dir/ca.pem --sslPEMKeyFile /work-dir/mongo.pem"]
        volumeMounts:
        - mountPath: "/work-dir"
          name: tmp-mongodb
        - mountPath: "/dump"
          name: mongodump
        - mountPath: "/cred/mongo-certs"
          name: icp-mongodb-client-cert
        - mountPath: "/cred/cluster-ca"
          name: cluster-ca-cert
        env:
          - name: ADMIN_USER
            valueFrom:
              secretKeyRef:
                name: icp-mongodb-admin
                key: user
          - name: ADMIN_PASSWORD
            valueFrom:
              secretKeyRef:
                name: icp-mongodb-admin
                key: password
      volumes:
      - name: mongodump
        persistentVolumeClaim:
          claimName: cs-mongodump
      - name: tmp-mongodb
        emptyDir: {}
      - name: icp-mongodb-client-cert
        secret:
          secretName: icp-mongodb-client-cert
      - name: cluster-ca-cert
        secret:
          secretName: mongodb-root-ca-cert
      restartPolicy: OnFailure
EOF
  else
    scale_down
    cat <<EOF | oc apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: mongodb-backup
  namespace: $CS_NAMESPACE
spec:
  parallelism: 1
  completions: 1
  backoffLimit: 20
  template:
    spec:
      containers:
      - name: cs-mongodb-backup
        image: $ibm_mongodb_image
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 100m
            memory: 128Mi
        command: ["bash", "-c", "cat /cred/mongo-certs/tls.crt /cred/mongo-certs/tls.key > /work-dir/mongo.pem; cat /cred/cluster-ca/tls.crt /cred/cluster-ca/tls.key > /work-dir/ca.pem; mongodump --oplog --out /dump/dump --host mongodb:27017 --username \$ADMIN_USER --password \$ADMIN_PASSWORD --authenticationDatabase admin"]
        volumeMounts:
        - mountPath: "/work-dir"
          name: tmp-mongodb
        - mountPath: "/dump"
          name: mongodump
        - mountPath: "/cred/mongo-certs"
          name: icp-mongodb-client-cert
        - mountPath: "/cred/cluster-ca"
          name: cluster-ca-cert
        env:
          - name: ADMIN_USER
            valueFrom:
              secretKeyRef:
                name: icp-mongodb-admin
                key: user
          - name: ADMIN_PASSWORD
            valueFrom:
              secretKeyRef:
                name: icp-mongodb-admin
                key: password
      volumes:
      - name: mongodump
        persistentVolumeClaim:
          claimName: cs-mongodump
      - name: tmp-mongodb
        emptyDir: {}
      - name: icp-mongodb-client-cert
        secret:
          secretName: icp-mongodb-client-cert
      - name: cluster-ca-cert
        secret:
          secretName: mongodb-root-ca-cert
      restartPolicy: OnFailure
EOF
    scale_up
  fi
  sleep 15s

  LOOK=$(oc get po --no-headers=true -n $CS_NAMESPACE | grep mongodb-backup | awk '{ print $1 }')
  waitforpods "mongodb-backup" $CS_NAMESPACE
  success "Dump completed: Use the [oc logs $LOOK -n $CS_NAMESPACE] command for details on the backup operation"

} # backup-mongodb()

function waitforpods() {
  index=0
  retries=60
  msg "Waiting for $1 pod(s) to start ..."
  while true; do
      [[ $index -eq $retries ]] && exit 1
      if [ -z $1 ]; then
        pods=$(oc get pods --no-headers -n $2 2>&1)
      else
        pods=$(oc get pods --no-headers -n $2 | grep $1 2>&1)
      fi
      echo "$pods" | egrep -q -v 'Completed|Succeeded|No resources found.' || break
      [[ $(( $index % 10 )) -eq 0 ]] && echo "$pods" | egrep -v 'Completed|Succeeded'
      sleep 10
      index=$(( index + 1 ))
  done
  if [ -z $1 ]; then
    oc get pods --no-headers=true -n $2
  else
    oc get pods --no-headers=true -n $2 | grep $1
  fi
}

function scale_up(){
    info "Z cluster detected, be prepared for multiple restarts of mongo pods. This is expected behavior."
    mongo_op_scaled_original=$(oc get deploy -n $CS_NAMESPACE | grep ibm-mongodb-operator | egrep '1/1' || echo false)
    if [[ $mongo_op_scaled_original == "false" ]]; then
        info "Mongo operator in $CS_NAMESPACE still scaled down, scaling up."
        oc scale deploy -n $CS_NAMESPACE ibm-mongodb-operator --replicas=1
        info "Wait for mongo operator to reconcile resources"
        sleep 60
        delete_mongo_pods "$CS_NAMESPACE"
    fi
    success "Mongo reset successfully."
}

function scale_down(){
    info "Z cluster detected, be prepared for multiple restarts of mongo pods. This is expected behavior."
    info "Scaling down MongoDB operator"
    oc scale deploy -n $CS_NAMESPACE ibm-mongodb-operator --replicas=0
    #get cache size value
    cacheSizeGB=$(oc get cm icp-mongodb -n $CS_NAMESPACE -o yaml | grep cacheSizeGB | awk '{print $2}')

    info "Editing configmap icp-mongodb"
    cat << EOF | oc apply -n $CS_NAMESPACE -f -
kind: ConfigMap
apiVersion: v1
metadata:
  name: icp-mongodb
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: icp-mongodb
    app.kubernetes.io/managed-by: operator
    app.kubernetes.io/name: icp-mongodb
    app.kubernetes.io/part-of: common-services-cloud-pak
    app.kubernetes.io/version: 4.0.12-build.3
    release: mongodb
data:
  mongod.conf: |-
    storage:
      dbPath: /data/db
      wiredTiger:
        engineConfig:
          cacheSizeGB: $cacheSizeGB
    net:
      bindIpAll: true
      port: 27017
      ssl:
        mode: preferSSL
        CAFile: /data/configdb/tls.crt
        PEMKeyFile: /work-dir/mongo.pem
    replication:
      replSetName: rs0
    # Uncomment for TLS support or keyfile access control without TLS
    security:
      authorization: enabled
      keyFile: /data/configdb/key.txt
EOF
    delete_mongo_pods "$CS_NAMESPACE"
    success "Mongo prepped for backup or restore successfully."
}

function delete_mongo_pods() {
  local namespace=$1
  local pods=$(oc get pods -n $namespace | grep icp-mongodb | awk '{print $1}' | tr "\n" " ")
  for pod in $pods
  do
    info "Deleting pod $pod"
    oc delete pod $pod -n $CS_NAMESPACE --ignore-not-found
    local condition="oc get pod -n $namespace --no-headers --ignore-not-found | grep ${pod} | egrep '2/2' || oc get pod -n $namespace --no-headers --ignore-not-found | grep ${pod} | egrep '1/1' || true"
    local retries=15
    local sleep_time=15
    local total_time_mins=$(( sleep_time * retries / 60))
    local wait_message="Waiting for mongo pod $pod to restart "
    local success_message="Pod $pod restarted with new mongo config"
    local error_message="Timeout after ${total_time_mins} minutes waiting for pod $pod "
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
  done
}

function wait_for_condition() {
    local condition=$1
    local retries=$2
    local sleep_time=$3
    local wait_message=$4
    local success_message=$5
    local error_message=$6

    info "${wait_message}"
    while true; do
        result=$(eval "${condition}")

        if [[ ( ${retries} -eq 0 ) && ( -z "${result}" ) ]]; then
            error "${error_message}"
        fi
 
        sleep ${sleep_time}
        result=$(eval "${condition}")
        
        if [[ -z "${result}" ]]; then
            info "RETRYING: ${wait_message} (${retries} left)"
            retries=$(( retries - 1 ))
        else
            break
        fi
    done

    if [[ ! -z "${success_message}" ]]; then
        success "${success_message}\n"
    fi
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

if [[ -z $CS_NAMESPACE ]]; then
  export CS_NAMESPACE=ibm-common-services
fi

cleanup
backup_mongodb
