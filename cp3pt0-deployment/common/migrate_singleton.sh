#!/usr/bin/env bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product. 
# Please refer to that particular license for additional information. 

# ---------- Command arguments ----------

OC=oc
YQ=yq
OPERATOR_NS=""
CONTROL_NS=""
SOURCE_NS="openshift-marketplace"
ENABLE_LICENSING=0
LSR_NAMESPACE="ibm-lsr"
LICENSING_NS=""
NEW_MAPPING=""
NEW_TENANT=0
DEBUG=0
PREVIEW_MODE=0

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# counter to keep track of installation steps
STEP=0

# ---------- Main functions ----------

. ${BASE_DIR}/utils.sh

function main() {
    parse_arguments "$@"
    pre_req

    # Delete CP2.0 Cert-Manager CR
    ${OC} delete certmanager.operator.ibm.com default --ignore-not-found --timeout=10s
    if [ $? -ne 0 ]; then
        warning "Failed to delete Cert Manager CR, patching its finalizer to null..."
        ${OC} patch certmanagers.operator.ibm.com default --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
    fi

    if [ ! -z "$CONTROL_NS" ]; then
        # Delegation of CP2 Cert Manager
        ${BASE_DIR}/delegate_cp2_cert_manager.sh --control-namespace $CONTROL_NS "--skip-user-vertify"
    fi

    delete_operator "ibm-cert-manager-operator" "$OPERATOR_NS"
    
    if [[ $ENABLE_LICENSING -eq 1 ]]; then

        #Prepare LSR PV/PVC which was decoupled in isolate.sh
        # delete old LSR CR - PV should stay,
        ${OC} delete IBMLicenseServiceReporter instance -n ibm-common-services

        # in case PVC is blocked with deletion, the finalizer needs to be removed
        lsr_pvcs=$("${OC}" get pvc license-service-reporter-pvc -n ibm-common-services  --no-headers | wc -l)
        if [[ lsr_pvcs -gt 0 ]]; then
            info "Failed to delete pvc license-service-reporter-pvc, patching its finalizer to null..."
            ${OC} patch pvc license-service-reporter-pvc -n ibm-common-services  --type="json" -p '[{"op": "remove", "path":"/metadata/finalizers"}]'
        else
            debug1 "No pvc license-service-reporter-pvc as expected"
        fi

        lsr_pv_nr=$("${OC}" get pv -l license-service-reporter-pv=true --no-headers | wc -l )
        if [[ lsr_pv_nr -gt 1 ]]; then
          error "More than on PV with label license-service-reporter-pv=true was found. Only one is allowed."
        fi

        if [[ lsr_pv_nr -eq 1 ]]; then

          debug1 "LSR namespace: ${LSR_NAMESPACE}" 
          create_namespace "${LSR_NAMESPACE}"

          # on ROKS storage class name cannot be proviced during PVC creation
          LSR_PV_NAME=$("${OC}" get pv -l license-service-reporter-pv=true -o=jsonpath='{.items[0].metadata.name}')
          desc=$("${OC}" get pv $LSR_PV_NAME -n $LSR_NAMESPACE -o yaml)
          debug1 "1: $desc"
          # get storage class name
          roks=$(${OC} cluster-info | grep 'containers.cloud.ibm.com')
          if [[ -z $roks ]]; then
            LSR_STORAGE_CLASS=$("${OC}" get pv -l license-service-reporter-pv=true -o=jsonpath='{.items[0].spec.storageClassName}')
            if [[ -z $LSR_STORAGE_CLASS ]]; then
                error "Cannnot get storage class name from PVC license-service-reporter-pv in $LSR_NAMESPACE"
            fi
          else
            debug1 "Run on ROKS, not setting storageclass name"
            LSR_STORAGE_CLASS=""
          fi

          # create PVC
          TEMP_LSR_PVC_FILE="_TEMP_LSR_PVC_FILE.yaml"

          cat <<EOF >$TEMP_LSR_PVC_FILE
          apiVersion: v1
          kind: PersistentVolumeClaim
          metadata:
            name: license-service-reporter-pvc
            namespace: ${LSR_NAMESPACE}
          spec:
            accessModes:
            - ReadWriteOnce
            resources:
              requests:
                storage: 1Gi
            storageClassName: "${LSR_STORAGE_CLASS}"
            volumeMode: Filesystem
            volumeName: ${LSR_PV_NAME}
EOF
          ${OC} create -f ${TEMP_LSR_PVC_FILE}
          # checking status of PVC - in case it cannot be boud, the claimRef needs to be set to null
          status=$("${OC}" get pvc license-service-reporter-pvc -n $LSR_NAMESPACE --no-headers | awk '{print $2}')
          while [[ "$status" != "Bound" ]]
          do
            namespace=$("${OC}" get pv ${LSR_PV_NAME} -o=jsonpath='{.spec.claimRef.namespace}')
            if [[ $namespace != $LSR_NAMESPACE ]]; then
                ${OC} patch pv ${LSR_PV_NAME} --type=merge -p '{"spec": {"claimRef":null}}'
            fi
            info "Waiting for pvc license-service-reporter-pvc to bind"
            sleep 10
            status=$("${OC}" get pvc license-service-reporter-pvc -n $LSR_NAMESPACE --no-headers | awk '{print $2}')
          done
        fi
        
        backup_ibmlicensing
        isExists=$("${OC}" get deployments -n "${CONTROL_NS}" --ignore-not-found ibm-licensing-operator)
        if [ ! -z "$isExists" ]; then
            "${OC}" delete  --ignore-not-found ibmlicensing instance
        fi

        # Delete licensing csv/subscriptions
        delete_operator "ibm-licensing-operator" "$OPERATOR_NS"

        # restore licensing configuration so that subsequent License Service install will pick them up
        restore_ibmlicensing
    fi

    success "Migration is completed for Cloud Pak 3.0 Foundational singleton services."
}


function restore_ibmlicensing() {

    # extracts the previously saved IBMLicensing CR from ConfigMap and creates the IBMLicensing CR
    "${OC}" get cm ibmlicensing-instance-bak -n ${LICENSING_NS} -o yaml --ignore-not-found | "${YQ}" .data | sed -e 's/.*ibmlicensing.yaml.*//' | 
    sed -e 's/^  //g' | oc apply -f -

}

function backup_ibmlicensing() {

    instance=`"${OC}" get IBMLicensing instance -o yaml --ignore-not-found | "${YQ}" '
        with(.; del(.metadata.creationTimestamp) |
        del(.metadata.managedFields) |
        del(.metadata.resourceVersion) |
        del(.metadata.uid) |
        del(.status)
        )
    ' | sed -e 's/^/    /g'`
cat << _EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: ibmlicensing-instance-bak
  namespace: ${LICENSING_NS}
data:
  ibmlicensing.yaml: |
${instance}
_EOF

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
            YQ=$1
            ;;
        --operator-namespace)
            shift
            OPERATOR_NS=$1
            ;;
        --control-namespace)
            shift
            CONTROL_NS=$1
            ;;
        --licensing-namespace)
            shift
            LICENSING_NS=$1
            ;;
        --enable-licensing)
            shift
            ENABLE_LICENSING=1
            ;;
        --lsr-namespace)
            shift
            LSR_NAMESPACE=$1
            ;;
        -v | --debug)
            shift
            DEBUG=$1
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
    echo "Usage: ${script_name} --operator-namespace <foundational-services-namespace> [OPTIONS]..."
    echo ""
    echo "Migrate Cloud Pak 2.0 Foundational singleton services to in Cloud Pak 3.0 Foundational singleton services"
    echo "The --operator-namespace must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --yq string                                    File path to yq CLI. Default uses yq in your PATH"
    echo "   --operator-namespace string                    Required. Namespace to migrate Foundational services operator"
    echo "   --enable-licensing                             Set this flag to migrate ibm-licensing-operator"
    echo "   --lsr-namespace                                Required. Namespace to migrate License Service Reporter"
    echo "   -v, --debug integer                            Verbosity of logs. Default is 0. Set to 1 for debug logs."
    echo "   -h, --help                                     Print usage information"
    echo ""
}

function pre_req() {
    if [ "$CONTROL_NS" == "" ]; then
        CONTROL_NS=$OPERATOR_NS
    fi    
}

main "$@"