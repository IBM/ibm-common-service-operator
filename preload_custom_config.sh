#!/usr/bin/env bash
#
# Copyright 2023 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# set -o errexit
set -o pipefail
set -o errtrace
# set -o nounset

OC=oc
YQ=yq
FROM_NAMESPACE=
TO_NAMESPACE=
NUM=$#
TEMPFILE="_TMP.yaml"

function main() {
    parse_arguments "$@"
    prereq
    # run backup preload
    backup_preload_mongo
    # copy im credentials
    copy_auth_idp_secret
    # any extra config
}

function parse_arguments() {
    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --oc)
            shift
            OC=$1
            ;;
        --yq)
            shift
            yq=$1
            ;;
        --original-cs-ns)
            shift
            FROM_NAMESPACE=$1
            ;;
        --target-ns)
            shift
            TO_NAMESPACE=$1
            ;;
        -h | --help)
            print_usage
            exit 1
            ;;
        *) 
            echo "wildcard"
            ;;
        esac
        shift
    done
}

function print_usage() {
    script_name=`basename ${0}`
    echo "Usage: ${script_name} [OPTIONS]..."
    echo ""
    echo "Install Cloud Pak 3 pre-reqs if they do not already exist: ibm-cert-manager-operator and optionally ibm-licensing-operator"
    echo "The ibm-cert-manager-operator will be installed in namespace ibm-cert-manager"
    echo "The ibm-licensing-operator will be installed in namespace ibm-licensing"
    echo ""
    echo "Options:"
    echo "   --oc string                                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --yq string                                    File path to yq CLI. Default uses yq in your PATH"
    echo "   --target-ns string                             Namespace to migrate Cloud Pak 2 Foundational services data too"
    echo "   --original-cs-ns string                        Namespace to migrate Cloud Pak 2 Foundational services data from."
    echo "   -h, --help                                     Print usage information"
    echo ""
}

function prereq() {
    
    if [[ -z "$FROM_NAMESPACE" ]] || [[ -z "$TO_NAMESPACE" ]]; then
        error "Both --original-cs-ns and --target-ns need to be set for script to execute. Please rerun script with both parameters set. Run with \"-h\" flag for more details"
        exit 1
    fi

    exists=$(oc get ns $FROM_NAMESPACE --no-headers --ignore-not-found)
    if [[ -z "$exists" ]]; then
        error "Namespace $FROM_NAMESPACE does not exist (or oc command line is not logged in)"
        exit 1
    fi 

    exists=$(oc get ns $TO_NAMESPACE --no-headers --ignore-not-found)
    if [[ -z "$exists" ]]; then
        error "Namespace $TO_NAMESPACE does not exist (or oc command line is not logged in)"
        exit 1
    fi
}

function copy_auth_idp_secret() {
    local secret="platform-auth-idp-credentials"
    title " Copying secret $secret from $FROM_NAMESPACE to $TO_NAMESPACE "
    
    $OC get secret "$secret" -n $FROM_NAMESPACE -o yaml > tmp.yaml
    $YQ -i 'del(.metadata.creationTimestamp)' tmp.yaml
    $YQ -i 'del(.metadata.resourceVersion)' tmp.yaml
    $YQ -i 'del(.metadata.uid)' tmp.yaml
    $YQ -i 'del(.metadata.ownerReferences)' tmp.yaml
    $YQ -i '.metadata.namespace = "'${TO_NAMESPACE}'"' tmp.yaml
    $OC apply -f tmp.yaml || error "Failed to copy over secret $secret."
    rm -f tmp.yaml
    success "Secret $secret copied over to $TO_NAMESPACE"
}

#
# backup_preload_mongo script logic
#
function backup_preload_mongo() {
  pre_req_bpm
  cleanup
  deploymongocopy
  createdumppvc
  dumpmongo
  swapmongopvc
  loadmongo
  deletemongocopy
} # backup_preload_mongo
  

#
# Parse and validate the namespaces
#
function pre_req_bpm() {

  info "Copying mongodb from namespace $FROM_NAMESPACE to namespace $TO_NAMESPACE"
 
  runningmongo=$(oc get po icp-mongodb-0 --no-headers --ignore-not-found -n $FROM_NAMESPACE | awk '{print $3}')
  if [[ -z "$runningmongo" ]] || [[ "$runningmongo" != "Running" ]]; then
    error "Mongodb is not running in Namespace $FROM_NAMESPACE"
    exit -1
  fi

  runningmongo=$(oc get po icp-mongodb-0 --no-headers --ignore-not-found -n $TO_NAMESPACE | awk '{print $3}')
  if [[ ! -z "$runningmongo" ]]; then
    error "Mongodb is deployedoc g in Namespace $TO_NAMESPACE - this copy depends on mongo being uninitialzed in the target namespace"
    exit -1
  fi
} # parse


#
# Cleanup artifacts from previous executions
#
function cleanup() {
  title "Cleaning up any previous copy operations..."
  msg "-----------------------------------------------------------------------"
  rm $TEMPFILE
  oc delete job mongodb-backup -n $FROM_NAMESPACE
  oc delete job mongodb-restore -n $TO_NAMESPACE
  pvcexists=$(oc get pvc cs-mongodump -n $FROM_NAMESPACE --no-headers --ignore-not-found | awk '{print $2}')
  if [[ -n "$pvcexists" ]]; then
    if [[ "$pvcexists" == "Bound" ]]; then
      dv=$(oc get pvc cs-mongodump -n $FROM_NAMESPACE -o=jsonpath='{.spec.volumeName}')
      oc patch pv $dv -p '{"spec": { "persistentVolumeReclaimPolicy" : "Delete" }}'
    fi
    #TODO remove finalizers before deleting
    oc delete pvc cs-mongodump -n $FROM_NAMESPACE --ignore-not-found --timeout=10s
    if [ $? -ne 0 ]; then
        info "Failed to delete pvc cs-mongodump, patching its finalizer to null..."
        oc patch pvc cs-mongodump -n $FROM_NAMESPACE --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
    fi
  fi
  pvcexists=$(oc get pvc cs-mongodump -n $TO_NAMESPACE --no-headers --ignore-not-found | awk '{print $2}')
  if [[ -n "$pvcexists" ]]; then
    if [[ "$pvcexists" == "Bound" ]]; then
      dv=$(oc get pvc cs-mongodump -n $TO_NAMESPACE -o=jsonpath='{.spec.volumeName}')
      oc patch pv $dv -p '{"spec": { "persistentVolumeReclaimPolicy" : "Delete" }}'
    fi
    oc delete pvc cs-mongodump -n $TO_NAMESPACE --ignore-not-found --timeout=10s
    if [ $? -ne 0 ]; then
        info "Failed to delete pvc cs-mongodump, patching its finalizer to null..."
        oc patch pvc cs-mongodump -n $TO_NAMESPACE --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
    fi
  fi
  success "Previous run cleaned up."
} # cleanup


#
#  Create the dump PVC
#
function createdumppvc() {
  title "Creating a PVC for the MongoDB dump"
  msg "-----------------------------------------------------------------------"
  oc project $FROM_NAMESPACE
  currentns=$(oc project -q)
  if [[ "$currentns" -ne "$FROM_NAMESPACE" ]]; then
    error "Cannot switch to $FROM_NAMESPACE"
  fi

  stgclass=$(oc get pvc mongodbdir-icp-mongodb-0 -o=jsonpath='{.spec.storageClassName}')
  if [[ -z $stgclass ]]; then
    error "Cannnot get storage class name from PVC mongodbdir-icp-mongodb-0 in $FROM_NAMESPACE"
  fi

  cat <<EOF >$TEMPFILE
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: cs-mongodump
  namespace: $FROM_NAMESPACE
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: $stgclass
  volumeMode: Filesystem
EOF

  oc apply -f $TEMPFILE

  status=$(oc get pvc cs-mongodump --no-headers | awk '{print $2}')
  while [[ "$status" != "Bound" ]]
  do
    info "Waiting for pvc cs-mongodump to bind"
    sleep 10
    status=$(oc get pvc cs-mongodump --no-headers | awk '{print $2}')
  done
  success "MongoDB PVC ready"

} # createdumppvc


#
# Backup(Dump) the mongodb in the from: namespace
#
function dumpmongo() {
  title "Backing up MongoDB in namespace $FROM_NAMESPACE"
  msg "-----------------------------------------------------------------------"
  currentns=$(oc project $FROM_NAMESPACE -q)
  if [[ "$currentns" -ne "$FROM_NAMESPACE" ]]; then
    error "Cannot switch to $FROM_NAMESPACE"
  fi

  cat <<EOF >$TEMPFILE
apiVersion: batch/v1
kind: Job
metadata:
  name: mongodb-backup
spec:
  parallelism: 1
  completions: 1
  backoffLimit: 20
  template:
    spec:
      containers:
      - name: cs-mongodb-backup
        image: quay.io/opencloudio/ibm-mongodb:4.0.24
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

  status="Unknown"
  info "Running Backup" 

  while [[ "$status" != "Completed" ]]
  do
    oc apply -f $TEMPFILE
    sleep 10
    retries=10
    while [ $retries > 0 ]
    do
      info "waiting for completion"
      status=$(oc get po | grep mongodb-backup | awk '{print $3}')
      oc get po | grep mongodb-backup
      if [[ "$status" == "Completed" ]]; then
        break
      elif [[ "$status" == "Running" ]]; then
        retries=10
        sleep 10
      elif [[ "$status" == "" ]]; then
        break
      else
        retries=$(( $retries - 1 ))
        sleep 10
      fi  
    done
    if [[ "$status" != "Completed" ]]; then
      info "Retrying mongodb-backup"
      oc delete job mongodb-backup
    fi
  done

  dumplogs mongodb-backup
  success "Backup Complete"
} # dumpmongo


#
# Swap the PVC from the from_namespace to the to_namespace
#
function swapmongopvc() {
  title "Moving restored mongodb volume to $TO_NAMESPACE"
  msg "-----------------------------------------------------------------------"

  status=$(oc get pvc cs-mongodump -n $FROM_NAMESPACE)
  if [[ -z "$status" ]]; then
    error "PVC cs-mongodump not found in $FROM_NAMESPACE"
  fi

  VOL=$(oc get pvc cs-mongodump -n $FROM_NAMESPACE  -o=jsonpath='{.spec.volumeName}')
  if [[ -z "$VOL" ]]; then
    error "Volume for pvc  cs-mongodump not found in $FROM_NAMESPACE"
  fi

  IMAGE=$(oc get pod icp-mongodb-0 -n $FROM_NAMESPACE  -o=jsonpath='{range .spec.containers[0]}{.image}{end}')
  if [[ -z "$IMAGE" ]]; then
    error "IMAGE for pod icp-mongodb-0 not found in $FROM_NAMESPACE"
  fi

  oc patch pv $VOL -p '{"spec": { "persistentVolumeReclaimPolicy" : "Retain" }}'
  oc patch pv $VOL --type=merge -p '{"spec": {"claimRef":null}}'
  oc patch pv $VOL --type json -p '[{ "op": "remove", "path": "/spec/claimRef" }]'
  
  oc delete pvc cs-mongodump -n $FROM_NAMESPACE --ignore-not-found --timeout=10s
    if [ $? -ne 0 ]; then
        info "Failed to delete pvc cs-mongodump, patching its finalizer to null..."
        oc patch pvc cs-mongodump -n $FROM_NAMESPACE --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
    fi

  stgclass=$(oc get pvc mongodbdir-icp-mongodb-0 -n $FROM_NAMESPACE -o=jsonpath='{.spec.storageClassName}')
  if [[ -z $stgclass ]]; then
    error "Cannnot get storage class name from PVC mongodbdir-icp-mongodb-0 in $FROM_NAMESPACE"
  fi

  cat <<EOF >$TEMPFILE
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: cs-mongodump
  namespace: $TO_NAMESPACE
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: $stgclass
  volumeMode: Filesystem
  volumeName: $VOL
EOF

  oc create -f $TEMPFILE

  status=$(oc get pvc cs-mongodump -n $TO_NAMESPACE --no-headers | awk '{print $2}')
  while [[ "$status" != "Bound" ]]
  do
    namespace=$(oc get pv $VOL -o=jsonpath='{.spec.claimRef.namespace}')
    if [[ $namespace != $TO_NAMESPACE ]]; then
      oc patch pv $VOL --type=merge -p '{"spec": {"claimRef":null}}'
    fi
    info "Waiting for pvc cs-mongodump to bind"
    sleep 10
    status=$(oc get pvc cs-mongodump -n $TO_NAMESPACE --no-headers | awk '{print $2}')
  done

  success "Restored MongoDB volume moved to namespace $TO_NAMESPACE"
} # swappvc


#
# Restore the mongodb in the to: namespace
#
function loadmongo() {
  title "Restoring MongoDB to copy in namespace $TO_NAMESPACE"
  msg "-----------------------------------------------------------------------"

  currentns=$(oc project $TO_NAMESPACE -q)
  if [[ "$currentns" -ne "$TO_NAMESPACE" ]]; then
    error "Cannot switch to $TO_NAMESPACE"
  fi

  cat <<EOF >$TEMPFILE
apiVersion: batch/v1
kind: Job
metadata:
  name: mongodb-restore
spec:
  parallelism: 1
  completions: 1
  backoffLimit: 20
  template:
    spec:
      containers:
      - name: icp-mongodb-restore
        image: quay.io/opencloudio/ibm-mongodb:4.0.24
        command: ["bash", "-c", "cat /cred/mongo-certs/tls.crt /cred/mongo-certs/tls.key > /work-dir/mongo.pem; cat /cred/cluster-ca/tls.crt /cred/cluster-ca/tls.key > /work-dir/ca.pem; mongorestore --host rs0/icp-mongodb:27017 --username \$ADMIN_USER --password \$ADMIN_PASSWORD --authenticationDatabase admin --ssl --sslCAFile /work-dir/ca.pem --sslPEMKeyFile /work-dir/mongo.pem /dump/dump"]
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

  status="Unknown"
  info "Running Restore" 
  
  while [[ "$status" != "Completed" ]]
  do
    info "Starting MongoDB Restore Job "
    oc apply -f $TEMPFILE
    sleep 10
    retries=10
    while [ $retries > 0 ]
    do
      info "waiting for completion"
      status=$(oc get po | grep mongodb-restore | awk '{print $3}')
      oc get po | grep mongodb-restore
      if [[ "$status" == "Completed" ]] || [[ "$status" == "" ]]; then
        break
      elif [[ "$status" == "Running" ]]; then
        retries=10
        sleep 10
      else
        retries=$(( $retries - 1 ))
        sleep 10
      fi  
    done
    if [[ "$status" != "Completed" ]]; then
      info "Retrying MongoDB Restore"
      oc delete job mongodb-restore
    fi
  done
  dumplogs mongodb-restore
  success "Restore Complete"
} # loadmongo


#
# Dump logs for amtching pod
#
function dumplogs() {
  info "Saving $1 logs in _${1}.log"
  pod=$(oc get po | grep $1 | awk '{print $1}')
  if [[ -n "$pod" ]]; then
    oc logs $pod >_${1}.log
  else
    echo "No pod" >_${1}.log
  fi
} # dumplogs


#
# deploymongocopy
#
function deploymongocopy {
  title "Deploying a temporary mongodb in $TO_NAMESPACE"
  msg "-----------------------------------------------------------------------"
  currentns=$(oc project $TO_NAMESPACE -q)
  if [[ "$currentns" -ne "$TO_NAMESPACE" ]]; then
    error "Cannot switch to $TO_NAMESPACE"
  fi

  STGCLASS=$(oc get pvc mongodbdir-icp-mongodb-0 -n $FROM_NAMESPACE -o=jsonpath='{.spec.storageClassName}')
  if [[ -z $STGCLASS ]]; then
    error "Cannnot get storage class name from PVC mongodbdir-icp-mongodb-0 in $FROM_NAMESPACE"
  fi

    cat << EOF > mongo-init-cm.yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: icp-mongodb-init
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: icp-mongodb
    app.kubernetes.io/managed-by: operator
    app.kubernetes.io/name: icp-mongodb
    app.kubernetes.io/part-of: common-services-cloud-pak
    app.kubernetes.io/version: 4.0.12-build.3
    release: mongodb
data:
  on-start.sh: >-
    #!/bin/bash

    ## workaround
    https://serverfault.com/questions/713325/openshift-unable-to-write-random-state

    export RANDFILE=/tmp/.rnd

    port=27017

    replica_set=\$REPLICA_SET

    script_name=\${0##*/}

    credentials_file=/work-dir/credentials.txt

    config_dir=/data/configdb


    function log() {
        local msg="\$1"
        local timestamp=\$(date --iso-8601=ns)
        1>&2 echo "[\$timestamp] [\$script_name] \$msg"
        echo "[\$timestamp] [\$script_name] \$msg" >> /work-dir/log.txt
    }


    if [[ "\$AUTH" == "true" ]]; then

        if [ !  -f "\$credentials_file" ]; then
            log "Creds File Not found!"
            log "Original User: \$ADMIN_USER"
            echo \$ADMIN_USER > \$credentials_file
            echo \$ADMIN_PASSWORD >> \$credentials_file
        fi
        admin_user=\$(head -n 1 \$credentials_file)
        admin_password=\$(tail -n 1 \$credentials_file)
        admin_auth=(-u "\$admin_user" -p "\$admin_password")
        log "Original User: \$admin_user"
        if [[ "\$METRICS" == "true" ]]; then
            metrics_user="\$METRICS_USER"
            metrics_password="\$METRICS_PASSWORD"
        fi
    fi


    function shutdown_mongo() {

        log "Running fsync..."
        mongo admin "\${admin_auth[@]}" "\${ssl_args[@]}" --eval "db.adminCommand( { fsync: 1, lock: true } )"

        log "Running fsync unlock..."
        mongo admin "\${admin_auth[@]}" "\${ssl_args[@]}" --eval "db.adminCommand( { fsyncUnlock: 1 } )"

        log "Shutting down MongoDB..."
        mongo admin "\${admin_auth[@]}" "\${ssl_args[@]}" --eval "db.adminCommand({ shutdown: 1, force: true, timeoutSecs: 60 })"
    }


    #Check if Password has change and updated in mongo , if so update Creds

    function update_creds_if_changed() {
      if [ "\$admin_password" != "\$ADMIN_PASSWORD" ]; then
          passwd_changed=true
          log "password has changed = \$passwd_changed"
          log "checking if passwd  updated in mongo"
          mongo admin  "\${ssl_args[@]}" --eval "db.auth({user: '\$admin_user', pwd: '\$ADMIN_PASSWORD'})" | grep "Authentication failed"
          if [[ \$? -eq 1 ]]; then
            log "New Password worked, update creds"
            echo \$ADMIN_USER > \$credentials_file
            echo \$ADMIN_PASSWORD >> \$credentials_file
            admin_password=\$ADMIN_PASSWORD
            admin_auth=(-u "\$admin_user" -p "\$admin_password")
            passwd_updated=true
          fi
      fi
    }


    function update_mongo_password_if_changed() {
      log "checking if mongo passwd needs to be  updated"
      if [[ "\$passwd_changed" == "true" ]] && [[ "\$passwd_updated" != "true" ]]; then
        log "Updating to new password "
        if [[ \$# -eq 1 ]]; then
            mhost="--host \$1"
        else
            mhost=""
        fi

        log "host for password upd (\$mhost)"
        mongo admin \$mhost "\${admin_auth[@]}" "\${ssl_args[@]}" --eval "db.changeUserPassword('\$admin_user', '\$ADMIN_PASSWORD')" >> /work-dir/log.txt 2>&1
        sleep 10
        log "mongo passwd change attempted; check and update creds file if successful"
        update_creds_if_changed
      fi
    }




    my_hostname=\$(hostname)

    log "Bootstrapping MongoDB replica set member: \$my_hostname"


    log "Reading standard input..."

    while read -ra line; do
        log "line is  \${line}"
        if [[ "\${line}" == *"\${my_hostname}"* ]]; then
            service_name="\$line"
        fi
        peers=("\${peers[@]}" "\$line")
    done


    # Move into /work-dir

    pushd /work-dir

    pwd >> /work-dir/log.txt

    ls -l  >> /work-dir/log.txt


    # Generate the ca cert

    ca_crt=\$config_dir/tls.crt

    if [ -f \$ca_crt  ]; then
        log "Generating certificate"
        ca_key=\$config_dir/tls.key
        pem=/work-dir/mongo.pem
        ssl_args=(--ssl --sslCAFile \$ca_crt --sslPEMKeyFile \$pem)

        echo "ca stuff created" >> /work-dir/log.txt

    cat >openssl.cnf <<DUMMYEOL

    [req]

    req_extensions = v3_req

    distinguished_name = req_distinguished_name

    [req_distinguished_name]

    [ v3_req ]

    basicConstraints = CA:FALSE

    keyUsage = nonRepudiation, digitalSignature, keyEncipherment

    subjectAltName = @alt_names

    [alt_names]

    DNS.1 = \$(echo -n "\$my_hostname" | sed s/-[0-9]*\$//)

    DNS.2 = \$my_hostname

    DNS.3 = \$service_name

    DNS.4 = localhost

    DNS.5 = 127.0.0.1

    DNS.6 = mongodb

    DUMMYEOL

        # Generate the certs
        echo "cnf stuff" >> /work-dir/log.txt
        echo "genrsa " >> /work-dir/log.txt
        openssl genrsa -out mongo.key 2048 >> /work-dir/log.txt 2>&1

        echo "req " >> /work-dir/log.txt
        openssl req -new -key mongo.key -out mongo.csr -subj "/CN=\$my_hostname" -config openssl.cnf >> /work-dir/log.txt 2>&1

        echo "x509 " >> /work-dir/log.txt
        openssl x509 -req -in mongo.csr \
            -CA \$ca_crt -CAkey \$ca_key -CAcreateserial \
            -out mongo.crt -days 3650 -extensions v3_req -extfile openssl.cnf >> /work-dir/log.txt 2>&1

        echo "mongo stuff" >> /work-dir/log.txt

        rm mongo.csr

        cat mongo.crt mongo.key > \$pem
        rm mongo.key mongo.crt
    fi



    log "Peers: \${peers[@]}"


    log "Starting a MongoDB instance..."

    mongod --config \$config_dir/mongod.conf >> /work-dir/log.txt 2>&1 &

    pid=\$!

    trap shutdown_mongo EXIT



    log "Waiting for MongoDB to be ready..."

    until [[ \$(mongo "\${ssl_args[@]}" --quiet --eval
    "db.adminCommand('ping').ok") == "1" ]]; do
        log "Retrying..."
        sleep 2
    done


    log "Initialized."


    if [[ "\$AUTH" == "true" ]]; then
        update_creds_if_changed
    fi


    iter_counter=0

    while [  \$iter_counter -lt 5 ]; do
      log "primary check, iter_counter is \$iter_counter"
      # try to find a master and add yourself to its replica set.
      for peer in "\${peers[@]}"; do
          log "Checking if \${peer} is primary"
          mongo admin --host "\${peer}" --ipv6 "\${admin_auth[@]}" "\${ssl_args[@]}" --quiet --eval "rs.status()"  >> log.txt

          # Check rs.status() first since it could be in primary catch up mode which db.isMaster() doesn't show
          if [[ \$(mongo admin --host "\${peer}" --ipv6 "\${admin_auth[@]}" "\${ssl_args[@]}" --quiet --eval "rs.status().myState") == "1" ]]; then
              log "Found master \${peer}, wait while its in primary catch up mode "
              until [[ \$(mongo admin --host "\${peer}" --ipv6 "\${admin_auth[@]}" "\${ssl_args[@]}" --quiet --eval "db.isMaster().ismaster") == "true" ]]; do
                  sleep 1
              done
              primary="\${peer}"
              log "Found primary: \${primary}"
              break
          fi
      done

      if [[ -z "\${primary}" ]]  && [[ \${#peers[@]} -gt 1 ]] && (mongo "\${ssl_args[@]}" --eval "rs.status()" | grep "no replset config has been received"); then
        log "waiting before creating a new replicaset, to avoid conflicts with other replicas"
        sleep 30
      else
        break
      fi

      let iter_counter=iter_counter+1
    done



    if [[ "\${primary}" = "\${service_name}" ]]; then
        log "This replica is already PRIMARY"

    elif [[ -n "\${primary}" ]]; then

        if [[ \$(mongo admin --host "\${primary}" --ipv6 "\${admin_auth[@]}" "\${ssl_args[@]}" --quiet --eval "rs.conf().members.findIndex(m => m.host == '\${service_name}:\${port}')") == "-1" ]]; then
          log "Adding myself (\${service_name}) to replica set..."
          if (mongo admin --host "\${primary}" --ipv6 "\${admin_auth[@]}" "\${ssl_args[@]}" --eval "rs.add('\${service_name}')" | grep 'Quorum check failed'); then
              log 'Quorum check failed, unable to join replicaset. Exiting.'
              exit 1
          fi
        fi
        log "Done,  Added myself to replica set."

        sleep 3
        log 'Waiting for replica to reach SECONDARY state...'
        until printf '.'  && [[ \$(mongo admin "\${admin_auth[@]}" "\${ssl_args[@]}" --quiet --eval "rs.status().myState") == '2' ]]; do
            sleep 1
        done
        log '✓ Replica reached SECONDARY state.'

    elif (mongo "\${ssl_args[@]}" --eval "rs.status()" | grep "no replset config
    has been received"); then

        log "Initiating a new replica set with myself (\$service_name)..."

        mongo "\${ssl_args[@]}" --eval "rs.initiate({'_id': '\$replica_set', 'members': [{'_id': 0, 'host': '\$service_name'}]})"
        mongo "\${ssl_args[@]}" --eval "rs.status()"

        sleep 3

        log 'Waiting for replica to reach PRIMARY state...'

        log ' Waiting for rs.status state to become 1'
        until printf '.'  && [[ \$(mongo "\${ssl_args[@]}" --quiet --eval "rs.status().myState") == '1' ]]; do
            sleep 1
        done

        log ' Waiting for master to complete primary catchup mode'
        until [[ \$(mongo  "\${ssl_args[@]}" --quiet --eval "db.isMaster().ismaster") == "true" ]]; do
            sleep 1
        done

        primary="\${service_name}"
        log '✓ Replica reached PRIMARY state.'


        if [[ "\$AUTH" == "true" ]]; then
            # sleep a little while just to be sure the initiation of the replica set has fully
            # finished and we can create the user
            sleep 3

            log "Creating admin user..."
            mongo admin "\${ssl_args[@]}" --eval "db.createUser({user: '\$admin_user', pwd: '\$admin_password', roles: [{role: 'root', db: 'admin'}]})"
        fi

        log "Done initiating replicaset."

    fi


    log "Primary: \${primary}"


    if [[  -n "\${primary}"   && "\$AUTH" == "true" ]]; then
        # you r master and passwd has changed.. then update passwd
        update_mongo_password_if_changed \$primary

        if [[ "\$METRICS" == "true" ]]; then
            log "Checking if metrics user is already created ..."
            metric_user_count=\$(mongo admin --host "\${primary}" "\${admin_auth[@]}" "\${ssl_args[@]}" --eval "db.system.users.find({user: '\${metrics_user}'}).count()" --quiet)
            log "User count is \${metric_user_count} "
            if [[ "\${metric_user_count}" == "0" ]]; then
                log "Creating clusterMonitor user... user - \${metrics_user}  "
                mongo admin --host "\${primary}" "\${admin_auth[@]}" "\${ssl_args[@]}" --eval "db.createUser({user: '\${metrics_user}', pwd: '\${metrics_password}', roles: [{role: 'clusterMonitor', db: 'admin'}, {role: 'read', db: 'local'}]})"
                log "User creation return code is \$? "
                metric_user_count=\$(mongo admin --host "\${primary}" "\${admin_auth[@]}" "\${ssl_args[@]}" --eval "db.system.users.find({user: '\${metrics_user}'}).count()" --quiet)
                log "User count now is \${metric_user_count} "
            fi
        fi
    fi


    log "MongoDB bootstrap complete"

    exit 0
EOF

    #oc apply -f mongo-restore-resources/restore-icp-mongodb-install-cm.yaml
    cat << EOF > mongo-install-cm.yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: icp-mongodb-install
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: icp-mongodb
    app.kubernetes.io/managed-by: operator
    app.kubernetes.io/name: icp-mongodb
    app.kubernetes.io/part-of: common-services-cloud-pak
    app.kubernetes.io/version: 4.0.12-build.3
    release: mongodb
data:
  install.sh: >-
    #!/bin/bash


    # Copyright 2016 The Kubernetes Authors. All rights reserved.

    #

    # Licensed under the Apache License, Version 2.0 (the "License");

    # you may not use this file except in compliance with the License.

    # You may obtain a copy of the License at

    #

    #     http://www.apache.org/licenses/LICENSE-2.0

    #

    # Unless required by applicable law or agreed to in writing, software

    # distributed under the License is distributed on an "AS IS" BASIS,

    # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

    # See the License for the specific language governing permissions and

    # limitations under the License.


    # This volume is assumed to exist and is shared with the peer-finder

    # init container. It contains on-start/change configuration scripts.

    WORKDIR_VOLUME="/work-dir"

    CONFIGDIR_VOLUME="/data/configdb"


    for i in "\$@"

    do

    case \$i in
        -c=*|--config-dir=*)
        CONFIGDIR_VOLUME="\${i#*=}"
        shift
        ;;
        -w=*|--work-dir=*)
        WORKDIR_VOLUME="\${i#*=}"
        shift
        ;;
        *)
        # unknown option
        ;;
    esac

    done


    echo installing config scripts into "\${WORKDIR_VOLUME}"

    mkdir -p "\${WORKDIR_VOLUME}"

    cp /peer-finder "\${WORKDIR_VOLUME}"/

    echo "I am running as " \$(whoami)


    cp /configdb-readonly/mongod.conf "\${CONFIGDIR_VOLUME}"/mongod.conf

    cp /keydir-readonly/key.txt "\${CONFIGDIR_VOLUME}"/

    cp /ca-readonly/tls.key "\${CONFIGDIR_VOLUME}"/tls.key

    cp /ca-readonly/tls.crt "\${CONFIGDIR_VOLUME}"/tls.crt


    chmod 600 "\${CONFIGDIR_VOLUME}"/key.txt

    # chown -R 999:999 /work-dir

    # chown -R 999:999 /data


    # Root file system is readonly but still need write and execute access to
    tmp

    # chmod -R 777 /tmp
EOF
    oc apply -f mongo-install-cm.yaml
    rm -f mongo-install-cm.yaml

    oc apply -f mongo-init-cm.yaml
    rm -f mongo-init-cm.yaml

    #god-issuer-issuer.yaml
    cat << EOF | oc apply -f -
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: god-issuer
  labels:
    app.kubernetes.io/instance: mongodbs.operator.ibm.com
    app.kubernetes.io/managed-by: mongodbs.operator.ibm.com
    app.kubernetes.io/name: mongodbs.operator.ibm.com
spec:
  selfSigned: {}
EOF
    #ibm-cpp-config-cm.yaml
    cat << EOF | oc apply -f -
kind: ConfigMap
apiVersion: v1
metadata:
  name: ibm-cpp-config
data:
  storageclass.default: rook-ceph-block
  storageclass.list: 'rook-ceph-block,rook-cephfs'
EOF
    #icp-mongodb-admin-secret.yaml
    cat << EOF | oc apply -f -
kind: Secret
apiVersion: v1
metadata:
  name: icp-mongodb-admin
  labels:
    app: icp-mongodb
data:
  password: VlZWVlZWVlZWVlZWVg==
  user: QkJCQkJCQkI=
type: Opaque
EOF
    #icp-mongodb-client-cert-cert.yaml
    cat << EOF | oc apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: icp-mongodb-client-cert
spec:
  commonName: mongodb-service
  dnsNames:
    - mongodb
  duration: 17520h
  isCA: false
  issuerRef:
    kind: Issuer
    name: mongodb-root-ca-issuer
  secretName: icp-mongodb-client-cert
EOF
    #icp-mongodb-cm.yaml
    cat << EOF | oc apply -f -
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
          cacheSizeGB: 0.26
    net:
      bindIpAll: true
      port: 27017
      ssl:
        mode: requireSSL
        CAFile: /data/configdb/tls.crt
        PEMKeyFile: /work-dir/mongo.pem
    replication:
      replSetName: rs0
    # Uncomment for TLS support or keyfile access control without TLS
    security:
      authorization: enabled
      keyFile: /data/configdb/key.txt
EOF
    #icp-mongodb-keyfile-secret.yaml
    cat << EOF | oc apply -f -
kind: Secret
apiVersion: v1
metadata:
  name: icp-mongodb-keyfile
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: icp-mongodb
    app.kubernetes.io/managed-by: operator
    app.kubernetes.io/name: icp-mongodb
    release: mongodb
data:
  key.txt: aWNwdGVzdA==
type: Opaque
EOF
    #icp-mongodb-metrics-secret.yaml
    cat << EOF | oc apply -f -
kind: Secret
apiVersion: v1
metadata:
  name: icp-mongodb-metrics
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: icp-mongodb
    app.kubernetes.io/managed-by: operator
    app.kubernetes.io/name: icp-mongodb
    release: mongodb
data:
  password: aWNwbWV0cmljcw==
  user: bWV0cmljcw==
type: Opaque
EOF
    #mongo-rbac.yaml
    cat << EOF | oc apply -f -
kind: ServiceAccount
apiVersion: v1
metadata:
  name: ibm-mongodb-operand
  labels:
    app.kubernetes.io/instance: mongodbs.operator.ibm.com
    app.kubernetes.io/managed-by: mongodbs.operator.ibm.com
    app.kubernetes.io/name: mongodbs.operator.ibm.com
secrets:
  - name: ibm-mongodb-operand-dockercfg-x7n5t
imagePullSecrets:
  - name: ibm-mongodb-operand-dockercfg-x7n5t
EOF
    #mongo-service.yaml
    cat << EOF | oc apply -f -
kind: Service
apiVersion: v1
metadata:
  name: mongodb
  labels:
    app.kubernetes.io/component: database
    app.kubernetes.io/instance: icp-mongodb
    app.kubernetes.io/managed-by: operator
    app.kubernetes.io/name: icp-mongodb
    app.kubernetes.io/part-of: common-services-cloud-pak
    app.kubernetes.io/version: 4.0.12-build.3
    release: mongodb
spec:
  ipFamilies:
    - IPv4
  ports:
    - protocol: TCP
      port: 27017
      targetPort: 27017
  internalTrafficPolicy: Cluster
  type: ClusterIP
  ipFamilyPolicy: SingleStack
  sessionAffinity: None
  selector:
    app: icp-mongodb
    release: mongodb
status:
  loadBalancer: {}
EOF
    #mongo-service2.yaml
    cat << EOF | oc apply -f -
kind: Service
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
spec:
  clusterIP: None
  publishNotReadyAddresses: true
  ipFamilies:
    - IPv4
  ports:
    - name: peer
      protocol: TCP
      port: 27017
      targetPort: 27017
  internalTrafficPolicy: Cluster
  clusterIPs:
    - None
  type: ClusterIP
  ipFamilyPolicy: SingleStack
  sessionAffinity: None
  selector:
    app: icp-mongodb
    release: mongodb
EOF
    #mongodb-root-ca-cert-certificate.yaml
    cat << EOF | oc apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: mongodb-root-ca-cert
  labels:
    app.kubernetes.io/instance: mongodbs.operator.ibm.com
    app.kubernetes.io/managed-by: mongodbs.operator.ibm.com
    app.kubernetes.io/name: mongodbs.operator.ibm.com
spec:
  commonName: mongodb
  dnsNames:
    - mongodb.root
  duration: 17520h
  isCA: true
  issuerRef:
    kind: Issuer
    name: god-issuer
  secretName: mongodb-root-ca-cert
EOF
    #mongodb-root-ca-issuer-issuer.yaml
    cat << EOF | oc apply -f -
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: mongodb-root-ca-issuer
  labels:
    app.kubernetes.io/instance: mongodbs.operator.ibm.com
    app.kubernetes.io/managed-by: mongodbs.operator.ibm.com
    app.kubernetes.io/name: mongodbs.operator.ibm.com
spec:
  ca:
    secretName: mongodb-root-ca-cert
EOF
    #namespace-scope-cm.yaml
    cat << EOF | oc apply -f -
kind: ConfigMap
apiVersion: v1
metadata:
  name: namespace-scope
data:
  namespaces: "$TO_NAMESPACE"
EOF
    #apply statefulset (in same dir)
    #get images from cp2 namespace
    ibm_mongodb_install_image=$(oc get pod icp-mongodb-0 -n $FROM_NAMESPACE -o=jsonpath='{range .spec.initContainers[0]}{.image}{end}')
    ibm_mongodb_image=$(oc get pod icp-mongodb-0 -n $FROM_NAMESPACE -o=jsonpath='{range .spec.containers[0]}{.image}{end}')
    ibm_mongodb_exporter_image=$(oc get pod icp-mongodb-0 -n $FROM_NAMESPACE -o=jsonpath='{range .spec.containers[1]}{.image}{end}')
    #icp-mongodb-ss.yaml
    cat << EOF | oc apply -f -
kind: StatefulSet
apiVersion: apps/v1
metadata:
  name: icp-mongodb
  labels:
    app: icp-mongodb
    app.kubernetes.io/instance: mongodbs.operator.ibm.com
    app.kubernetes.io/managed-by: mongodbs.operator.ibm.com
    app.kubernetes.io/name: mongodbs.operator.ibm.com
    release: mongodb
spec:
  replicas: 1
  selector:
    matchLabels:
      app: icp-mongodb
      release: mongodb
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: icp-mongodb
        app.kubernetes.io/instance: common-mongodb
        release: mongodb
      annotations:
        clusterhealth.ibm.com/dependencies: ibm-common-services.cert-manager
        productID: 068a62892a1e4db39641342e592daa25
        productMetric: FREE
        productName: IBM Cloud Platform Common Services
        prometheus.io/path: /metrics
        prometheus.io/port: '9216'
        prometheus.io/scrape: 'true'
    spec:
      restartPolicy: Always
      initContainers:
        - resources:
            limits:
              cpu: '1'
              memory: 640Mi
            requests:
              cpu: 500m
              memory: 640Mi
          terminationMessagePath: /dev/termination-log
          name: install
          command:
            - /install/install.sh
          imagePullPolicy: IfNotPresent
          volumeMounts:
            - name: mongodbdir
              mountPath: /work-dir
              subPath: workdir
            - name: configdir
              mountPath: /data/configdb
            - name: config
              mountPath: /configdb-readonly
            - name: install
              mountPath: /install
            - name: keydir
              mountPath: /keydir-readonly
            - name: ca
              mountPath: /ca-readonly
            - name: mongodbdir
              mountPath: /data/db
              subPath: datadir
            - name: tmp-mongodb
              mountPath: /tmp
          terminationMessagePolicy: File
          image: >-
            $ibm_mongodb_install_image
          args:
            - '--work-dir=/work-dir'
            - '--config-dir=/data/configdb'
        - resources:
            limits:
              cpu: '1'
              memory: 640Mi
            requests:
              cpu: 500m
              memory: 640Mi
          terminationMessagePath: /dev/termination-log
          name: bootstrap
          command:
            - /work-dir/peer-finder
          env:
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: metadata.namespace
            - name: REPLICA_SET
              value: rs0
            - name: AUTH
              value: 'true'
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
            - name: METRICS
              value: 'true'
            - name: METRICS_USER
              valueFrom:
                secretKeyRef:
                  name: icp-mongodb-metrics
                  key: user
            - name: METRICS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: icp-mongodb-metrics
                  key: password
            - name: NETWORK_IP_VERSION
              value: ipv4
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
          imagePullPolicy: IfNotPresent
          volumeMounts:
            - name: mongodbdir
              mountPath: /work-dir
              subPath: workdir
            - name: configdir
              mountPath: /data/configdb
            - name: init
              mountPath: /init
            - name: mongodbdir
              mountPath: /data/db
              subPath: datadir
            - name: tmp-mongodb
              mountPath: /tmp
          terminationMessagePolicy: File
          image: >-
            $ibm_mongodb_image
          args:
            - '-on-start=/init/on-start.sh'
            - '-service=icp-mongodb'
      serviceAccountName: ibm-mongodb-operand
      schedulerName: default-scheduler
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 50
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app
                      operator: In
                      values:
                        - icp-mongodb
                topologyKey: kubernetes.io/hostname
      terminationGracePeriodSeconds: 30
      securityContext: {}
      containers:
        - resources:
            limits:
              cpu: '1'
              memory: 640Mi
            requests:
              cpu: 500m
              memory: 640Mi
          readinessProbe:
            exec:
              command:
                - mongo
                - '--ssl'
                - '--sslCAFile=/data/configdb/tls.crt'
                - '--sslPEMKeyFile=/work-dir/mongo.pem'
                - '--eval'
                - db.adminCommand('ping')
            initialDelaySeconds: 5
            timeoutSeconds: 5
            periodSeconds: 10
            successThreshold: 1
            failureThreshold: 3
          terminationMessagePath: /dev/termination-log
          name: icp-mongodb
          command:
            - mongod
            - '--config=/data/configdb/mongod.conf'
          livenessProbe:
            exec:
              command:
                - mongo
                - '--ssl'
                - '--sslCAFile=/data/configdb/tls.crt'
                - '--sslPEMKeyFile=/work-dir/mongo.pem'
                - '--eval'
                - db.adminCommand('ping')
            initialDelaySeconds: 30
            timeoutSeconds: 10
            periodSeconds: 30
            successThreshold: 1
            failureThreshold: 5
          env:
            - name: AUTH
              value: 'true'
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
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
          ports:
            - name: peer
              containerPort: 27017
              protocol: TCP
          imagePullPolicy: IfNotPresent
          volumeMounts:
            - name: mongodbdir
              mountPath: /data/db
              subPath: datadir
            - name: configdir
              mountPath: /data/configdb
            - name: mongodbdir
              mountPath: /work-dir
              subPath: workdir
            - name: tmp-mongodb
              mountPath: /tmp
          terminationMessagePolicy: File
          image: >-
            $ibm_mongodb_image
        - resources:
            limits:
              cpu: '1'
              memory: 350Mi
            requests:
              cpu: 100m
              memory: 300Mi
          readinessProbe:
            exec:
              command:
                - sh
                - '-ec'
                - >-
                  /bin/mongodb_exporter --mongodb.uri
                  mongodb://$METRICS_USER:$METRICS_PASSWORD@localhost:27017
                  --mongodb.tls --mongodb.tls-ca=/data/configdb/tls.crt
                  --mongodb.tls-cert=/work-dir/mongo.pem --test
            timeoutSeconds: 1
            periodSeconds: 10
            successThreshold: 1
            failureThreshold: 3
          terminationMessagePath: /dev/termination-log
          name: metrics
          command:
            - sh
            - '-ec'
            - >-
              /bin/mongodb_exporter --mongodb.uri
              mongodb://$METRICS_USER:$METRICS_PASSWORD@localhost:27017
              --mongodb.tls --mongodb.tls-ca=/data/configdb/tls.crt
              --mongodb.tls-cert=/work-dir/mongo.pem --mongodb.socket-timeout=3s
              --mongodb.sync-timeout=1m --web.telemetry-path=/metrics
              --web.listen-address=:9216
          livenessProbe:
            exec:
              command:
                - sh
                - '-ec'
                - >-
                  /bin/mongodb_exporter --mongodb.uri
                  mongodb://$METRICS_USER:$METRICS_PASSWORD@localhost:27017
                  --mongodb.tls --mongodb.tls-ca=/data/configdb/tls.crt
                  --mongodb.tls-cert=/work-dir/mongo.pem --test
            initialDelaySeconds: 30
            timeoutSeconds: 10
            periodSeconds: 30
            successThreshold: 1
            failureThreshold: 10
          env:
            - name: METRICS_USER
              valueFrom:
                secretKeyRef:
                  name: icp-mongodb-metrics
                  key: user
            - name: METRICS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: icp-mongodb-metrics
                  key: password
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
          ports:
            - name: metrics
              containerPort: 9216
              protocol: TCP
          imagePullPolicy: IfNotPresent
          volumeMounts:
            - name: configdir
              mountPath: /data/configdb
            - name: mongodbdir
              mountPath: /work-dir
              subPath: workdir
            - name: tmp-metrics
              mountPath: /tmp
          terminationMessagePolicy: File
          image: >-
            $ibm_mongodb_exporter_image
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              key: app
              values: icp-mongodb
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/region
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels:
              key: app
              values: icp-mongodb
      serviceAccount: ibm-mongodb-operand
      volumes:
        - name: config
          configMap:
            name: icp-mongodb
            defaultMode: 420
        - name: init
          configMap:
            name: icp-mongodb-init
            defaultMode: 493
        - name: install
          configMap:
            name: icp-mongodb-install
            defaultMode: 493
        - name: ca
          secret:
            secretName: mongodb-root-ca-cert
            defaultMode: 493
        - name: keydir
          secret:
            secretName: icp-mongodb-keyfile
            defaultMode: 493
        - name: configdir
          emptyDir: {}
        - name: tmp-mongodb
          emptyDir: {}
        - name: tmp-metrics
          emptyDir: {}
      dnsPolicy: ClusterFirst
      tolerations:
        - key: dedicated
          operator: Exists
          effect: NoSchedule
        - key: CriticalAddonsOnly
          operator: Exists
        - key: node.kubernetes.io/not-ready
          operator: Exists
          effect: NoExecute
        - key: node.kubernetes.io/unreachable
          operator: Exists
          effect: NoExecute
  volumeClaimTemplates:
    - kind: PersistentVolumeClaim
      apiVersion: v1
      metadata:
        name: mongodbdir
        creationTimestamp: null
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 20Gi
        storageClassName: $STGCLASS
        volumeMode: Filesystem
      status:
        phase: Pending
  serviceName: icp-mongodb
  podManagementPolicy: OrderedReady
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      partition: 0
  revisionHistoryLimit: 10
EOF

  #oc apply -f $TEMPFILE

  status="unknown"
  
  while [[ "$status" != "Running" ]]
  do
    info "Waiting for MongoDB copy to initialize"
    sleep 10
    oc get po icp-mongodb-0 --no-headers
    status=$(oc get po icp-mongodb-0 --no-headers | awk '{print $3}')
  done

  success "Temporary Mongo copy deployed to namespace $TO_NAMESPACE"

} # deploymongocopy


#
# Delete the mongo copy
#
function deletemongocopy {
  title "Deleting the stand up mongodb statefulset in $TO_NAMESPACE"
  msg "-----------------------------------------------------------------------"

  currentns=$(oc project $TO_NAMESPACE -q)
  if [[ "$currentns" -ne "$TO_NAMESPACE" ]]; then
    error "Cannot switch to $TO_NAMESPACE"
  fi

  #delete all other resources EXCEPT icp-mongodb-admin
  oc delete statefulset icp-mongodb --ignore-not-found
  oc delete service icp-mongodb --ignore-not-found
  oc delete issuer god-issuer --ignore-not-found
  oc delete cm ibm-cpp-config --ignore-not-found
  oc delete certificate icp-mongodb-client-cert --ignore-not-found
  oc delete cm icp-mongodb --ignore-not-found
  oc delete cm icp-mongodb-init --ignore-not-found
  oc delete cm icp-mongodb-install --ignore-not-found
  oc delete secret icp-mongodb-keyfile --ignore-not-found
  oc delete secret icp-mongodb-metrics --ignore-not-found
  oc delete sa ibm-mongodb-operand --ignore-not-found
  oc delete service mongodb --ignore-not-found
  oc delete certificate mongodb-root-ca-cert --ignore-not-found
  oc delete issuer mongodb-root-ca-issuer --ignore-not-found
  oc delete cm namespace-scope --ignore-not-found
  
  #delete mongodump pvc and pv
  VOL=$(oc get pvc cs-mongodump -o=jsonpath='{.spec.volumeName}')
  if [[ -z "$VOL" ]]; then
    error "Volume for pvc cs-mongodump not found in $TO_NAMESPACE"
  fi

  oc patch pv $VOL -p '{"spec": { "persistentVolumeReclaimPolicy" : "Delete" }}'
  
  oc delete pvc cs-mongodump -n $TO_NAMESPACE --ignore-not-found --timeout=10s
  if [ $? -ne 0 ]; then
    info "Failed to delete pvc cs-mongodump, patching its finalizer to null..."
    oc patch pvc cs-mongodump -n $TO_NAMESPACE --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
  fi
  oc delete pv $VOL --ignore-not-found --timeout=10s
  if [ $? -ne 0 ]; then
    info "Failed to delete pv $VOL, patching its finalizer to null..."
    oc patch pv $VOL --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
  fi

  success "MongoDB restored to new namespace $TO_NAMESPACE"

} # deletemongocopy

function msg() {
    printf '%b\n' "$1"
}

function success() {
    msg "\33[32m[✔] ${1}\33[0m"
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

# --- Run ---

main $*