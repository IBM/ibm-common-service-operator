#!/bin/bash

# Prompt for user input
read -rp "Enter the Common services namespace: " NAMESPACE
read -rp "Enter the case number to generate archive (e.g., TS123456): " CASE_NUMBER

# Validate if provided namespace exist or not
if ! oc get namespace "$NAMESPACE" &>/dev/null; then
  echo "Error: Namespace '$NAMESPACE' does not exist!"
  exit 1
fi

# Check if jq is installed
if ! command -v jq &>/dev/null; then
  echo "Error: 'jq' command not found. Please install jq to proceed."
  exit 1
fi

# Define output directory
AUTHMGDIR="AuthidpLogs-$(date '+%y%b%dT%H-%M-%S')"
mkdir -p "$AUTHMGDIR/$NAMESPACE"

echo "Collecting logs for namespace: $NAMESPACE"
echo "Logs will be stored in: $AUTHMGDIR/$NAMESPACE"

# Fetch platform-auth-service pods
PODS=$(oc -n "$NAMESPACE" get pods -l component=platform-auth-service --no-headers -o custom-columns=name:.metadata.name)

if [[ -z "$PODS" ]]; then
  echo "No pods found with label component=platform-auth-service in namespace '$NAMESPACE'"
  exit 1
fi

for pod in $PODS; do
  echo "===== Collecting logs from $pod ====="
  LIBDIR="$AUTHMGDIR/$NAMESPACE/$pod-liberty"
  mkdir -p "$LIBDIR"

  echo "$pod: Collecting liberty logs..."
  oc -n "$NAMESPACE" cp "$pod:/logs" -c platform-auth-service "$LIBDIR/logs"

  echo "$pod: Collecting liberty configuration..."
  oc -n "$NAMESPACE" cp "$pod:/opt/ibm/wlp/usr/servers/defaultServer/" -c platform-auth-service "$LIBDIR/defaultserver"

  echo "$pod: Collecting keystore (key.jks)..."
  oc -n "$NAMESPACE" cp "$pod:/opt/ibm/wlp/output/defaultServer/resources/security/key.jks" -c platform-auth-service "$LIBDIR/key.jks"
done

# Gathering namespace-wide resources
echo "Gathering info from namespace: $NAMESPACE"
oc get all,secrets,cm,events -n "$NAMESPACE" -o wide &> "$AUTHMGDIR/$NAMESPACE/all-list.txt"

# Describe all pods
echo "Describing all pods in $NAMESPACE..."
oc get pods -n "$NAMESPACE" | awk -v ns="$NAMESPACE" -v dir="$AUTHMGDIR/$NAMESPACE" 'NR>1{print "oc -n "ns" describe pod "$1" > "dir"/"$1"-describe.txt && echo Described "$1}' | bash

# Collect logs from all containers
CONTAINER_LIST="$AUTHMGDIR/$NAMESPACE/container-list.txt"
oc get pods -n "$NAMESPACE" -o go-template='{{range $i := .items}}{{range $c := $i.spec.containers}}{{println $i.metadata.name $c.name}}{{end}}{{end}}' > "$CONTAINER_LIST"

echo "Gathering logs from containers..."
awk -v ns="$NAMESPACE" -v dir="$AUTHMGDIR/$NAMESPACE" '{print "oc -n "ns" logs "$1" -c "$2" --previous > "dir"/"$1"_"$2"_previous.log && echo Gathered previous logs of "$1"_"$2}' "$CONTAINER_LIST" | bash
awk -v ns="$NAMESPACE" -v dir="$AUTHMGDIR/$NAMESPACE" '{print "oc -n "ns" logs "$1" -c "$2" > "dir"/"$1"_"$2".log && echo Gathered logs of "$1"_"$2}' "$CONTAINER_LIST" | bash

### IAM Token and SCIM Queries
IAM_DIR="$AUTHMGDIR/$NAMESPACE/iam-data"
mkdir -p "$IAM_DIR"

echo "Fetching IAM admin credentials..."
IAM_ADMIN=$(oc get secret platform-auth-idp-credentials -n "$NAMESPACE" -o jsonpath='{.data.admin_username}' | base64 -d)
IAM_PASS=$(oc get secret platform-auth-idp-credentials -n "$NAMESPACE" -o jsonpath='{.data.admin_password}' | base64 -d)
IAM_HOST="https://$(oc get route cp-console -n "$NAMESPACE" -o jsonpath="{.spec.host}")"

echo "Obtaining IAM access token..."
IAM_ACCESS_TOKEN=$(curl -sk -X POST -H "Content-Type: application/x-www-form-urlencoded;charset=UTF-8" \
  -d "grant_type=password&username=$IAM_ADMIN&password=$IAM_PASS&scope=openid" \
  "$IAM_HOST/idprovider/v1/auth/identitytoken" | jq -r .access_token)

if [[ -z "$IAM_ACCESS_TOKEN" || "$IAM_ACCESS_TOKEN" == "null" ]]; then
  echo "Error: Failed to obtain IAM access token." | tee "$IAM_DIR/iam-error.log"
else
  echo "IAM access token retrieved successfully."
fi

echo "Fetching Identity Provider details..."
curl -sk -X GET --header "Authorization: Bearer $IAM_ACCESS_TOKEN" "$IAM_HOST/idprovider/v3/auth/idsource/" | jq > "$IAM_DIR/idp_configs.txt"

echo "Fetching IAM users..."
curl -sk -X GET --header "Authorization: Bearer $IAM_ACCESS_TOKEN" "$IAM_HOST/idmgmt/identity/api/v1/users" | jq > "$IAM_DIR/users.txt"

echo "Fetching SCIM attribute mappings..."
curl -sk -X GET --header "Authorization: Bearer $IAM_ACCESS_TOKEN" "$IAM_HOST/idmgmt/identity/api/v1/scim/attributemappings" | jq > "$IAM_DIR/scim_attribute_mappings.txt"

### IBM IAM Authentication CR data collection
IAM_AUTH_DIR="$AUTHMGDIR/$NAMESPACE/iam-auth-cr"
mkdir -p "$IAM_AUTH_DIR"

echo "Fetching IBM IAM authentication custom resource details..."
oc get authentications.operator.ibm.com -A &> "$IAM_AUTH_DIR/authentication-list.txt"

# Extract the namespace of the authentication resource
AUTH_NAMESPACE=$(oc get authentications.operator.ibm.com -A --no-headers | awk '{print $1}')
AUTH_NAME=$(oc get authentications.operator.ibm.com -A --no-headers | awk '{print $2}')

if [[ -n "$AUTH_NAMESPACE" && -n "$AUTH_NAME" ]]; then
  echo "Collecting authentication CR details from namespace: $AUTH_NAMESPACE"
  oc get authentications.operator.ibm.com "$AUTH_NAME" -o yaml -n "$AUTH_NAMESPACE" > "$IAM_AUTH_DIR/ibm-iam-authentication.yaml"
else
  echo "Error: Could not find authentication resource!" | tee "$IAM_AUTH_DIR/authentication-error.log"
fi

# Compress logs with case number
TAR_FILE="Case${CASE_NUMBER}-${AUTHMGDIR}.tgz"
echo "Creating archive: $TAR_FILE"
tar czf "$TAR_FILE" "$AUTHMGDIR/"

echo "Log collection complete. Archive saved as: $TAR_FILE"

