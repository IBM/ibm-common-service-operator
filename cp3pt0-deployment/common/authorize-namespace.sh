#!/bin/bash
#
# Copyright 2022 IBM Corporation
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

#
# Project roles and role bindings to another namespace
#

function help() {
    echo "authorize-namespace.sh - Authorize a namespace to be manageable from another namespace through the NamespaceScope operator"
    echo "See https://www.ibm.com/docs/en/cloud-paks/foundational-services/4.0?topic=co-authorizing-foundational-services-perform-operations-workloads-in-namespace for more information."
    echo "SYNTAX:"
    echo "authorize-namespace.sh [namespace | default current namespace] [-to namespace | default ibm-common-services] [-delete]"
    echo "WHERE:"
    echo "  namespace: It is the name of the namespace you wish to authorize.  This namespace MUST exist."
    echo "             By default, the current namespace is assumed"
    echo "  -to namespace: It is the name of the namespace of the NamespaceScope operator."
    echo "                 This namespace MUST exist.  The default is ibm-common-services."
    echo "  -delete: It removes the ability for the NamespaceScope operator in tonamespace to manage artifacts in the namespace."    
    echo ""
    echo "You must be logged into the Openshift cluster from the oc command line"
    echo ""
}

#
# MAIN LOGIC
#

TARGETNS=""
TONS="ibm-common-services"
DELETE=0

while (( $# )); do
  case "$1" in
    -to|--to)
      if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
        TONS=$2
        shift 2
      else
        echo "Error: Argument for $1 is missing" >&2
        exit 1
      fi
      ;;
    -delete|--delete)
      DELETE=1
      shift 1
      ;;
    -*|--*=) # unsupported flags
      echo "Error: Unsupported flag $1" >&2
      help
      exit 1
      ;;
    *) # preserve positional arguments
      TARGETNS="$TARGETNS $1"
      shift
      ;;
  esac
done

#
# Validate parameters
#

if [ -z $TARGETNS ]; then
    TARGETNS=$(oc project -q)
    if [ $? -ne 0 ]; then
      echo "Error: You do not seem to be logged into Openshift" >&2
      help
      exit 1
    fi
fi

COUNT=$(echo $TARGETNS | wc -w)
if [ $COUNT -ne 1 ]; then
    echo "Invalid  namespace " $TARGETNS >&2
    help
    exit 1
fi

TARGETNS=${TARGETNS//[[:blank:]]/}

oc get ns $TARGETNS
if [ $? -ne 0 ]; then
    echo "Invalid  namespace " $TARGETNS >&2
    help
    exit 1
fi

oc get ns $TONS
if [ $? -ne 0 ]; then
    echo "Invalid  namespace " $TARGETNS >&2
    help
    exit 1
fi

if [ $DELETE -eq 1 ]; then
  echo "Deleting authorization that the NamespaceScope operator in $TONS to manages namespace $TARGETNS" >&2
else
  echo "Authorizing the NamespaceScope operator in $TONS to manage namespace $TARGETNS " >&2
fi

#
# Delete permissions and update the list if needed
#
if [ $DELETE -ne 0 ]; then
  oc delete role nss-managed-role-from-$TONS -n $TARGETNS --ignore-not-found
  oc delete rolebinding nss-managed-role-from-$TONS -n $TARGETNS --ignore-not-found
  exit 0
fi


#
# Define a role for service accounts
#
cat <<EOF | oc apply -n $TARGETNS -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: nss-managed-role-from-$TONS
rules:
- apiGroups:
  - "*"
  resources:
  - "*"
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
  - deletecollection
EOF

#
# Bind the service account in the TO namespace to the Role in the target namespace
#
cat <<EOF | oc apply -n $TARGETNS -f -
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: nss-managed-role-from-$TONS
subjects:
- kind: ServiceAccount
  name: ibm-namespace-scope-operator
  namespace: $TONS
roleRef:
  kind: Role
  name: nss-managed-role-from-$TONS
  apiGroup: rbac.authorization.k8s.io
EOF
