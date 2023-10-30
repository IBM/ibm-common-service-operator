#!/bin/bash
CS_NAMESPACE=""
if [[ ! -z $1 ]]; then
  CS_NAMESPACE=$1
fi
s390x="false"
if [[ ! -z $2 ]]; then
  s390x=$2
fi

#
# Restore Mongo
#
function restore_mongodb () {
  msg "[$STEP] Restore the mongo database"
  STEP=$(( $STEP+1 ))

  oc delete secret icp-mongo-setaccess -n $CS_NAMESPACE >/dev/null 2>&1
  oc create secret generic icp-mongo-setaccess -n $CS_NAMESPACE --from-file=set_access.js

  oc get job -n $CS_NAMESPACE | grep mongodb-restore 2>&1
  if [ $? -eq 0 ]
  then
    echo "database restore job already run"
    echo "enter oc delete job mongodb-restore and re-run this script to do it again"
    exit -1
  else
    echo Starting restore

    ibm_mongodb_image=$(oc get pod icp-mongodb-0 -n $CS_NAMESPACE -o=jsonpath='{range .spec.containers[0]}{.image}{end}')
    if [[ $s390x == "false" ]]; then
      cat <<EOF | oc apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: mongodb-restore
  namespace: $CS_NAMESPACE
spec:
  parallelism: 1
  completions: 1
  backoffLimit: 20
  template:
    spec:
      containers:
      - name: icp-mongodb-restore
        image: $ibm_mongodb_image
        command: ["bash", "-c", "cat /cred/mongo-certs/tls.crt /cred/mongo-certs/tls.key > /work-dir/mongo.pem; cat /cred/cluster-ca/tls.crt /cred/cluster-ca/tls.key > /work-dir/ca.pem; mongorestore --host rs0/icp-mongodb:27017 --username \$ADMIN_USER --password \$ADMIN_PASSWORD --authenticationDatabase admin --ssl --sslCAFile /work-dir/ca.pem --sslPEMKeyFile /work-dir/mongo.pem /dump/dump"]
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 100m
            memory: 128Mi
        volumeMounts:
        - mountPath: "/dump"
          name: mongodump
        - mountPath: "/work-dir"
          name: tmp-mongodb
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
      restartPolicy: Never
EOF
    else
      scale_down
      cat <<EOF | oc apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: mongodb-restore
  namespace: $CS_NAMESPACE
spec:
  parallelism: 1
  completions: 1
  backoffLimit: 20
  template:
    spec:
      containers:
      - name: icp-mongodb-restore
        image: $ibm_mongodb_image
        command: ["bash", "-c", "cat /cred/mongo-certs/tls.crt /cred/mongo-certs/tls.key > /work-dir/mongo.pem; cat /cred/cluster-ca/tls.crt /cred/cluster-ca/tls.key > /work-dir/ca.pem; mongorestore --host rs0/icp-mongodb:27017 --username \$ADMIN_USER --password \$ADMIN_PASSWORD --authenticationDatabase admin /dump/dump"]
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 100m
            memory: 128Mi
        volumeMounts:
        - mountPath: "/dump"
          name: mongodump
        - mountPath: "/work-dir"
          name: tmp-mongodb
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
      restartPolicy: Never
EOF
    fi
    sleep 20s

    LOOK=$(oc get po --no-headers=true -n $CS_NAMESPACE | grep mongodb-restore | awk '{ print $1 }')
    waitforpodscompleted "mongodb-restore" $CS_NAMESPACE
    scale_up

    success "Restore completed: Use the [oc logs $LOOK -n $CS_NAMESPACE] command for details on the restore operation"
  fi
} # restore_mongodb


function waitforpodscompleted() {
  index=0
  retries=60
  echo "Waiting for $1 pod(s) to start ..."
  while true; do
      if [ $index -eq $retries ]; then
        error "Pods are not running or completed, Correct errors and re-run the script"
        exit -1
      fi
      sleep 10
      if [ -z $1 ]; then
        pods=$(oc get pods --no-headers -n $2 2>&1)
      else
        pods=$(oc get pods --no-headers -n $2 | grep $1 2>&1)
      fi
      #echo watching $pods
      echo "$pods" | egrep -q -v 'Completed|Succeeded|No resources found.' || break
      [[ $(( $index % 10 )) -eq 0 ]] && echo "$pods" | egrep -v 'Completed|Succeeded'
      index=$(( index + 1 ))
      # If one matching pod Completed and other matching pods in Error,  remove Error pods
      nothing=$(echo $pods | grep Completed)
      if [ $? -eq 0 ]; then
        nothing=$(echo $pods | grep Error)
        if [ $? -eq 0 ]; then
          echo "$pods" | grep Error | awk '{ print "oc delete po " $1 }' | bash -
        fi
      fi
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

restore_mongodb