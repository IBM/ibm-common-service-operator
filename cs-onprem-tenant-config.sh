#!/bin/bash
configMapCustomHostname="cs-onprem-tenant-config"
csNamespace=""
custom_hostname=""
topology=""
map_to_common_service_namespace=""

configmap_yaml=$(oc get configmap common-service-maps -n kube-public -ojsonpath='{.data.common-service-maps\.yaml}')
temp_yaml_file=$(mktemp)
echo "$configmap_yaml" > "$temp_yaml_file"
count=$(cat $temp_yaml_file |grep -A2 'requested-from-namespace:' | sed -n '/^  - /p' |wc -l)
if [ "$count" -eq 2 ]; then
    # Fetch the value of the "map-to-common-service-namespace" property
    map_to_common_service_namespace=$(cat $temp_yaml_file |grep 'map-to-common-service-namespace:' | awk '{print $3}')
    topology="SOD"
    csNamespace=$(cat $temp_yaml_file |grep -A2 'requested-from-namespace:' | sed -n '/^  - /p' |grep -v $map_to_common_service_namespace|awk '{print $2}')
    #echo "The value of 'map-to-common-service-namespace' is: $map_to_common_service_namespace and control namespace is $csNamespace"
else
    map_to_common_service_namespace=$(cat $temp_yaml_file |grep 'map-to-common-service-namespace:' | awk '{print $3}')
    topology="simple"
    csNamespace=$map_to_common_service_namespace
fi

echo "This cluster is configured with $topology topology"
wlp_client_id=$(oc get secret -n $map_to_common_service_namespace platform-oidc-credentials -o jsonpath='{.data.WLP_CLIENT_ID}'|base64 -d)
wlp_client_secret=$(oc get secret -n $map_to_common_service_namespace platform-oidc-credentials -o jsonpath='{.data.WLP_CLIENT_SECRET}'|base64 -d)
oauth2_client_registration_secret=$(oc get secret -n $map_to_common_service_namespace platform-oidc-credentials -o jsonpath='{.data.OAUTH2_CLIENT_REGISTRATION_SECRET}'|base64 -d)
admin_password=$(oc get secret -n $map_to_common_service_namespace platform-auth-idp-credentials  -ojsonpath='{.data.admin_password}'|base64 -d)
admin_username=$(oc get secret -n $map_to_common_service_namespace platform-auth-idp-credentials  -ojsonpath='{.data.admin_username}'|base64 -d)


checkIfConfigMapExist(){
  count=$(oc get cm -n $map_to_common_service_namespace |grep $configMapCustomHostname |wc -l)
  if [[ "$count" -ne 1 ]]; then
  echo "$configMapCustomHostname not found"
  exit 1
  fi
  #checkIfNamespaceExist
  checkIfhostReachable
}

checkIfSecretExist(){
  count=$(oc get configmap -n $map_to_common_service_namespace cs-onprem-tenant-config -o jsonpath='{.data.custom_host_certificate_secret}' |wc -w)
  if [[ "$count" -eq 1 ]]; then
    checkCrtFilesExist
    echo "Deleting old custom-tls-secret if exists"
    oc delete secret custom-tls-secret -n $map_to_common_service_namespace --ignore-not-found
    custom_secret=$(oc get configmap -n $map_to_common_service_namespace cs-onprem-tenant-config -o jsonpath='{.data.custom_host_certificate_secret}')
    echo "Creating custom-tls-secret"
    oc create secret generic $custom_secret -n $map_to_common_service_namespace --from-file=ca.crt=./ca.crt --from-file=tls.crt=./tls.crt --from-file=tls.key=./tls.key
  else
    echo "Custom secret not configured"
  fi
}

checkIfNamespaceExist(){
  csNamespace=$(oc get configmap -n $map_to_common_service_namespace cs-onprem-tenant-config -o jsonpath='{.metadata.namespace}')
  count=$(oc get namespaces |grep $csNamespace |wc -l)
  if [[ "$count" -ne 1 ]]; then
  echo "$csNamespace not found"
  exit 1
  fi
}

checkIfhostReachable() {
  custom_hostname=$(oc get configmap -n $map_to_common_service_namespace cs-onprem-tenant-config -o jsonpath='{.data.custom_hostname}')
  if [ -n "$custom_hostname" ]; then
      echo "Given Custom Hostname: $custom_hostname"
      if ping -c 1 "$custom_hostname" >/dev/null 2>&1; then
        echo "Host is reachable. Proceeding further..."
      else
        echo "$custom_hostname is not reachable. Exiting the script."
        exit 1
      fi
  fi
}

checkCrtFilesExist() {
  echo "Custom Secret is configured in configmap, so checking for crt availability"
  if [[ ! -f "tls.key" ]]; then
     echo "tls.key is not present in current directory, pls keep tls.key, tls.crt and ca.crt files in current directory"
     exit 1
  fi
  if [[ ! -f "tls.crt" ]]; then
     echo "tls.crt is not present in current directory,  pls keep tls.key, tls.crt and ca.crt files in current directory"
     exit 1
  fi
  if [[ ! -f "ca.crt" ]]; then
     echo "ca.crt is not present is not present in current directory,  pls keep tls.key, tls.crt and ca.crt files in current directory"
     exit 1
  fi

}

checkIfConfigMapExist
checkIfSecretExist

# delete completed job if exists
echo "Deleting old job of iam-custom-hostname if exists"
oc delete job iam-custom-hostname --ignore-not-found -n $csNamespace

echo "Running custom hostname job"
cat <<EOF | tee >(oc apply -f -) | cat >/dev/null
apiVersion: batch/v1
kind: Job
metadata:
  name: iam-custom-hostname
  namespace: $csNamespace
  labels:
    app: iam-custom-hostname
spec:
  template:
    metadata:
      labels:
        app: iam-custom-hostname
    spec:
      containers:
      - name: iam-custom-hostname
        image: icr.io/cpopen/cpfs/iam-custom-hostname:4.0.0
        command: ["python3", "/scripts/saas_script.py"]
        imagePullPolicy: Always
        env:
          - name: OPENSHIFT_URL
            value: https://kubernetes.default:443
          - name: IDENTITY_PROVIDER_URL
            value: https://platform-identity-provider.$map_to_common_service_namespace.svc:4300
          - name: PLATFORM_AUTH_URL
            value: https://platform-auth-service.$map_to_common_service_namespace.svc:9443
          - name: POD_NAMESPACE
            value: $map_to_common_service_namespace
          - name: WLP_CLIENT_ID
            value: $wlp_client_id
          - name: WLP_CLIENT_SECRET
            value: $wlp_client_secret
          - name: OAUTH2_CLIENT_REGISTRATION_SECRET
            value: $oauth2_client_registration_secret
          - name: DEFAULT_ADMIN_USER
            value: $admin_username
          - name: DEFAULT_ADMIN_PASSWORD
            value: $admin_password
      serviceAccountName: ibm-iam-operator
      restartPolicy: OnFailure
EOF

# Setting the label of the pod which needs to be deleted
AUTH_SERVICE_LABEL="app=platform-auth-service"
PROVIDER_LABEL="app=platform-identity-provider"
ZEN_OPERATOR_LABEL="name=ibm-zen-operator"
USERMGMT_LABEL="component=usermgmt"

# Function to check if the job has completed
check_job_completion() {
  job_name=$1
  namespace=$2
  timeout=120
  start_time=$(date +%s)

  while true; do
    # Get the job status
    job_status=$(oc get jobs "$job_name" -n "$namespace" -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}')

    # Check if the job is completed
    if [[ "$job_status" == "True" ]]; then
      echo "Job $job_name has completed."
      break
    fi

    current_time=$(date +%s)
    elapsed_time=$((current_time - start_time))
    if [[ $elapsed_time -ge $timeout ]]; then
      echo "Job did not complete within the timeout period."
      break
    fi

    echo "Waiting for job $job_name to complete..."
    sleep 5
  done
}



: '
# Restart the pod with the specified label
oc delete  -n $map_to_common_service_namespace $(oc get pods -n $map_to_common_service_namespace -l "$AUTH_SERVICE_LABEL" -o name) 
oc delete  -n $map_to_common_service_namespace $(oc get pods -n $map_to_common_service_namespace -l "$PROVIDER_LABEL" -o name) 
pods=$(oc get pods -l "$ZEN_OPERATOR_LABEL" -o name)
if [ -n "$pods" ]; then
  oc delete $pods
else
  echo "No pods found with label $ZEN_OPERATOR_LABEL"
fi

pods=$(oc get pods -l "$USERMGMT_LABEL" -o name)
if [ -n "$pods" ]; then
  oc delete $pods
else
  echo "No pods found with label $USERMGMT_LABEL"
fi
'

deployment_name="platform-auth-service"
timeout_seconds=180  # assuming auth-service will come in 3 mins after restart

start_time=$(date +%s)
end_time=$((start_time + timeout_seconds))

while true; do
  status=$(oc get deployment "$deployment_name" -n $map_to_common_service_namespace  -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')

  if [[ "$status" == "True" ]]; then
    echo "$deployment_name is available."
    break
  fi

  current_time=$(date +%s)
  if [[ "$current_time" -gt "$end_time" ]]; then
    echo "Timeout exceeded. $deployment_name is not available within the specified time."
    exit 1
  fi

  sleep 5  # Wait for 5 seconds before checking again
done

# Call the function to check job completion
check_job_completion iam-custom-hostname $csNamespace
