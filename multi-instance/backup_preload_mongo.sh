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

#set -o errexit
set -o pipefail
set -o errtrace
#set -o nounset

OC=${3:-oc}

NUM=$#
TEMPFILE="_TMP.yaml"

#
# main - main logic
#
function main() {
  parse $*
  cleanup
  deploymongocopy
  createdumppvc
  dumpmongo
  loadmongo
  swapmongopvc
} # main


#
# Help function - print out when error are found
#
function help() {
  echo "SYNTAX: copymongo.sh from-namespace to-namespace"
  echo "Where:"
  echo "  from-namespace: is the namespace in which mongodb is running and FROM which database content should be copied"
  echo "  to-namespace:   is the namespace in which mongodb is running and TO which database content should be copied"
} # help
  

#
# Parse and validate the namespaces
#
function parse() {
  info "Checking parameters and namespaces..."
  if [ $NUM -ne 2 ]; then
    help
    exit -1
  fi
  FROM_NAMESPACE=$1
  TO_NAMESPACE=$2

  info "Copying mongodb from namespace $FROM_NAMESPACE to namespace $TO_NAMESPACE"

  exists=$(oc get ns $FROM_NAMESPACE --no-headers --ignore-not-found)
  if [[ -z "$exists" ]]; then
    error "Namespace $FROM_NAMESPACE does not exist (or oc command line is not logged in)"
    exit -1
  fi 
  runningmongo=$(oc get po icp-mongodb-0 --no-headers --ignore-not-found -n $FROM_NAMESPACE | awk '{print $3}')
  if [[ -z "$runningmongo" ]] || [[ "$runningmongo" != "Running" ]]; then
    error "Mongodb is not running in Namespace $FROM_NAMESPACE"
    exit -1
  fi

  exists=$(oc get ns $TO_NAMESPACE --no-headers --ignore-not-found)
  if [[ -z "$exists" ]]; then
    error "Namespace $TO_NAMESPACE does not exist (or oc command line is not logged in)"
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
  info "Cleaning up any previous copy operations..."
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
    oc patch pvc cs-mongodump -n $FROM_NAMESPACE --type=merge -p '{"metadata": {"finalizers":null}}'
    oc delete pvc cs-mongodump -n $FROM_NAMESPACE
  fi
  pvcexists=$(oc get pvc cs-mongodump -n $TO_NAMESPACE --no-headers --ignore-not-found | awk '{print $2}')
  if [[ -n "$pvcexists" ]]; then
    if [[ "$pvcexists" == "Bound" ]]; then
      dv=$(oc get pvc cs-mongodump -n $TO_NAMESPACE -o=jsonpath='{.spec.volumeName}')
      oc patch pv $dv -p '{"spec": { "persistentVolumeReclaimPolicy" : "Delete" }}'
    fi
    oc patch pvc cs-mongodump -n $FROM_NAMESPACE --type=merge -p '{"metadata": {"finalizers":null}}'
    oc delete pvc cs-mongodump -n $TO_NAMESPACE
  fi
  deletemongocopy
} # cleanup


#
#  Create the dump PVC
#
function createdumppvc() {
  info "Creating a PVC for the MongoDB dump"

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

  #DUMPVOL=$(oc get pvc cs-mongodump  -o=jsonpath='{.spec.volumeName}')
  #oc patch pv $DUMPVOL -p '{"spec": { "persistentVolumeReclaimPolicy" : "Retain" }}'

} # createdumppvc


#
# Backup(Dump) the mongodb in the from: namespace
#
function dumpmongo() {
  info "Backing up MongoDB in namespace $FROM_NAMESPACE"

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
  info "Moving restored mongodb volume to $TO_NAMESPACE"

  status=$(oc get pvc mongodbdircopy-icp-mongodb-copy-0 -n $FROM_NAMESPACE)
  if [[ -z "$status" ]]; then
    error "PVC mongodbdircopy-icp-mongodb-copy-0 not found in $FROM_NAMESPACE"
  fi

  VOL=$(oc get pvc mongodbdircopy-icp-mongodb-copy-0 -n $FROM_NAMESPACE  -o=jsonpath='{.spec.volumeName}')
  if [[ -z "$VOL" ]]; then
    error "Volume for pvc  mongodbdircopy-icp-mongodb-copy-0 not found in $FROM_NAMESPACE"
  fi

  IMAGE=$(oc get pod icp-mongodb-0 -n $FROM_NAMESPACE  -o=jsonpath='{range .spec.containers[0]}{.image}{end}')
  if [[ -z "$IMAGE" ]]; then
    error "IMAGE for pod icp-mongodb-0 not found in $FROM_NAMESPACE"
  fi

  oc patch pv $VOL -p '{"spec": { "persistentVolumeReclaimPolicy" : "Retain" }}'
  deletemongocopy
  oc patch pv $VOL --type=merge -p '{"spec": {"claimRef":null}}'

  stgclass=$(oc get pvc mongodbdir-icp-mongodb-0 -n $FROM_NAMESPACE -o=jsonpath='{.spec.storageClassName}')
  if [[ -z $stgclass ]]; then
    error "Cannnot get storage class name from PVC mongodbdir-icp-mongodb-0 in $FROM_NAMESPACE"
  fi

  cat <<EOF >$TEMPFILE
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mongodbdir-icp-mongodb-0
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

  status=$(oc get pvc mongodbdir-icp-mongodb-0 -n $TO_NAMESPACE --no-headers | awk '{print $2}')
  while [[ "$status" != "Bound" ]]
  do
    info "Waiting for pvc mongodbdir-icp-mongodb-0 to bind"
    sleep 10
    status=$(oc get pvc mongodbdir-icp-mongodb-0 -n $TO_NAMESPACE --no-headers | awk '{print $2}')
  done
} # swappvc


#
# Restore the mongodb in the to: namespace
#
function loadmongo() {
  info "Restoring MongoDB to copy in namespace $FROM_NAMESPACE"

  currentns=$(oc project $FROM_NAMESPACE -q)
  if [[ "$currentns" -ne "$FROM_NAMESPACE" ]]; then
    error "Cannot switch to $FROM_NAMESPACE"
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
        command: ["bash", "-c", "cat /cred/mongo-certs/tls.crt /cred/mongo-certs/tls.key > /work-dir/mongo.pem; cat /cred/cluster-ca/tls.crt /cred/cluster-ca/tls.key > /work-dir/ca.pem; mongorestore --host rs0/icp-mongodb-copy:27017 --username \$ADMIN_USER --password \$ADMIN_PASSWORD --authenticationDatabase admin --ssl --sslCAFile /work-dir/ca.pem --sslPEMKeyFile /work-dir/mongo.pem /dump/dump"]
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
  info "Deploying a duplicate mongodb copy in $FROM_NAMESPACE"

  currentns=$(oc project $FROM_NAMESPACE -q)
  if [[ "$currentns" -ne "$FROM_NAMESPACE" ]]; then
    error "Cannot switch to $FROM_NAMESPACE"
  fi

  STGCLASS=$(oc get pvc mongodbdir-icp-mongodb-0 -o=jsonpath='{.spec.storageClassName}')
  if [[ -z $STGCLASS ]]; then
    error "Cannnot get storage class name from PVC mongodbdir-icp-mongodb-0 in $FROM_NAMESPACE"
  fi

#If this is a true copy, should we be copying existing to yaml files and changing the name?


  cat << EOF | oc apply -f -
kind: Service
apiVersion: v1
metadata:
  name: icp-mongodb-copy
  namespace: ibm-common-services
  labels:
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
    app: icp-mongodb-copy
    release: mongodb
EOF

  cat <<EOF >$TEMPFILE
kind: StatefulSet
apiVersion: apps/v1
metadata:
  name: icp-mongodb-copy
  namespace: ibm-common-services
  labels:
    app: icp-mongodb-copy
    app.kubernetes.io/instance: mongodbs.operator.ibm.com
    app.kubernetes.io/managed-by: mongodbs.operator.ibm.com
    app.kubernetes.io/name: mongodbs.operator.ibm.com
    release: mongodb
spec:
  replicas: 1
  selector:
    matchLabels:
      app: icp-mongodb-copy
      release: mongodb
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: icp-mongodb-copy
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
            - name: mongodbdircopy
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
            - name: mongodbdircopy
              mountPath: /data/db
              subPath: datadir
            - name: tmp-mongodb
              mountPath: /tmp
          terminationMessagePolicy: File
          image: >-
            docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-daily-docker-local/ibmcom/ibm-mongodb-install@sha256:bb236428cd36f3937d268c4475a1c62ac1a4e7cb9ca0f3de482f08817378b003
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
            - name: mongodbdircopy
              mountPath: /work-dir
              subPath: workdir
            - name: configdir
              mountPath: /data/configdb
            - name: init
              mountPath: /init
            - name: mongodbdircopy
              mountPath: /data/db
              subPath: datadir
            - name: tmp-mongodb
              mountPath: /tmp
          terminationMessagePolicy: File
          image: >-
            docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-daily-docker-local/ibmcom/ibm-mongodb@sha256:3a44fcf5656cdd3f062d3ca45d7fd0a46ff3ed90f6ed34ba260ad50938e95f57
          args:
            - '-on-start=/init/on-start.sh'
            - '-service=icp-mongodb-copy'
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
                        - icp-mongodb-copy
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
          name: icp-mongodb-copy
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
            - name: mongodbdircopy
              mountPath: /data/db
              subPath: datadir
            - name: configdir
              mountPath: /data/configdb
            - name: mongodbdircopy
              mountPath: /work-dir
              subPath: workdir
            - name: tmp-mongodb
              mountPath: /tmp
          terminationMessagePolicy: File
          image: >-
            docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-daily-docker-local/ibmcom/ibm-mongodb@sha256:3a44fcf5656cdd3f062d3ca45d7fd0a46ff3ed90f6ed34ba260ad50938e95f57
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
                  mongodb://\$METRICS_USER:\$METRICS_PASSWORD@localhost:27017
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
              mongodb://\$METRICS_USER:\$METRICS_PASSWORD@localhost:27017
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
                  mongodb://\$METRICS_USER:\$METRICS_PASSWORD@localhost:27017
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
            - name: mongodbdircopy
              mountPath: /work-dir
              subPath: workdir
            - name: tmp-metrics
              mountPath: /tmp
          terminationMessagePolicy: File
          image: >-
            docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-daily-docker-local/ibmcom/ibm-mongodb-exporter@sha256:f6456dc4e473295648c779cdabe1e4a40b660d69c9190fb2d5dfe7e94656ef17
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
        name: mongodbdircopy
        creationTimestamp: null
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 20Gi
        storageClassName: $STGCLASS
        volumeMode: Filesystem
  serviceName: icp-mongodb-copy
  podManagementPolicy: OrderedReady
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      partition: 0
  revisionHistoryLimit: 10
EOF

  oc apply -f $TEMPFILE

  status="unknown"
  
  #could use check_healthy $FROM_NAMESPACE instead
  while [[ "$status" != "Running" ]]
  do
    info "Waiting for MongoDB copy to initialize"
    sleep 10
    oc get po icp-mongodb-copy-0 --no-headers
    status=$(oc get po icp-mongodb-copy-0 --no-headers | awk '{print $3}')
  done

} # deploymongocopy


#
# Delete the mongo copy
#
function deletemongocopy {
  info "Deleting the duplicate mongodb copy in $FROM_NAMESPACE"

  currentns=$(oc project $FROM_NAMESPACE -q)
  if [[ "$currentns" -ne "$FROM_NAMESPACE" ]]; then
    error "Cannot switch to $FROM_NAMESPACE"
  fi

  oc delete statefulset icp-mongodb-copy
  oc delete service icp-mongodb-copy
  oc delete pvc mongodbdircopy-icp-mongodb-copy-0

} # deletemongocopy

#
# Messaging functions
#
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


main $*