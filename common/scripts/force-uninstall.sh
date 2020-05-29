#!/bin/bash
#
# Copyright 2020 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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

function wait_for_deleted(){
  kinds=$1
  ns=${2:---all-namespaces}
  index=0
  retries=${3:-10}
  while true; do
    rc=0
    for kind in ${kinds}; do
      cr=$(oc get ${kind} ${ns})
      rc=$?
      [[ "${rc}" != "0" ]] && break
      [[ "X${cr}" != "X" ]] && rc=99 && break
    done

    if [[ "${rc}" != "0" ]]; then
      [[ $(( $index % 5 )) -eq 0 ]] && msg "Resources are deleting, waiting for complete..."
      if [[ ${index} -eq ${retries} ]]; then
        error "Timeout for wait all resource deleted"
        exit 1
      fi
      sleep 60
      index=$(( index + 1 ))
    else
      success "All resources have been deleted"
      break
    fi
  done
}

#----------------------------- Clean UP -----------------------------#
namespace=ibm-common-services
title "Deleting common-service OperandRequest from namespace ${namespace}..."
oc delete OperandRequest common-service -n ${namespace} --ignore-not-found &
wait_for_deleted OperandRequest "-n ${namespace}" 20

title "Deleting other OperandRequest from all namespaces..."
oc delete OperandRequest --all --all-namespaces --ignore-not-found &
wait_for_deleted OperandRequest "--all-namespaces" 20

title "Deleting ODLM sub and csv"
csv_name=$(oc get sub operand-deployment-lifecycle-manager-app -o=jsonpath='{.status.installedCSV}' -n openshift-operators --ignore-not-found)
[[ "X${csv_name}" != "X" ]] && oc delete csv ${csv_name}  -n openshift-operators --ignore-not-found
oc delete sub operand-deployment-lifecycle-manager-app -n openshift-operators --ignore-not-found

title "Deleting RBAC resource"
oc delete ClusterRole ibm-common-service-webhook --ignore-not-found
oc delete ClusterRoleBinding ibm-common-service-webhook --ignore-not-found
oc delete RoleBinding ibmcloud-cluster-info -n kube-public --ignore-not-found
oc delete Role ibmcloud-cluster-info -n kube-public --ignore-not-found
oc delete RoleBinding ibmcloud-cluster-ca-cert -n kube-public --ignore-not-found
oc delete Role ibmcloud-cluster-ca-cert -n kube-public --ignore-not-found

title "Force deleting resource"
crds=$(oc get crd | grep operator.ibm.com | awk '{print $1}')
for crd in ${crds}; do
  msg "Deleting ${crd} kind resource from namespace ${namespace}"
  oc delete ${crd} --all -n ${namespace} --ignore-not-found &
done
wait_for_deleted "${crds}" "-n ${namespace}" 20
if [[ "$?" != "0" ]]; then
  for crd in ${crds}; do
    crs=$(oc get ${crd} --no-headers -n ${namespace} | awk '{print $1}')
    for cr in ${crs}; do
      msg "Removing the finalizers for resource: ${crd} $cr"
      oc patch ${crd} $cr -n ${namespace} --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]' 2>/dev/null
    done
  done
  wait_for_deleted "${crds}" "-n ${namespace}" 20
fi

title "Remove namespace ${namespace}"
oc delete namespace ${namespace} --ignore-not-found
[[ "$?" != "0" ]] && error "Something wrong, woooow...." && exit 1

