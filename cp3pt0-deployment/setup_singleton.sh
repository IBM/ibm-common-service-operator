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
ENABLE_LICENSING=0
ENABLE_LSR=0
ENABLE_PRIVATE_CATALOG=0
MIGRATE_SINGLETON=0
OPERATOR_NS=""
CONTROL_NS=""
CHANNEL="v4.0"
SOURCE_NS="openshift-marketplace"
INSTALL_MODE="Automatic"
CERT_MANAGER_SOURCE="ibm-cert-manager-catalog"
LICENSING_SOURCE="ibm-licensing-catalog"
LSR_SOURCE="ibm-license-service-reporter-operator-catalog"
CERT_MANAGER_NAMESPACE="ibm-cert-manager"
LICENSING_NAMESPACE="ibm-licensing"
LSR_NAMESPACE="ibm-lsr"
LICENSE_ACCEPT=0
DEBUG=0

CUSTOMIZED_LICENSING_NAMESPACE=0
SKIP_INSTALL=0
CHECK_LICENSING_ONLY=0

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# log file
LOG_FILE="setup_singleton_log_$(date +'%Y%m%d%H%M%S').log"

# counter to keep track of installation steps
STEP=0

# ---------- Main functions ----------

. ${BASE_DIR}/common/utils.sh

function main() {
    parse_arguments "$@"
    save_log "logs" "setup_singleton_log" "$DEBUG"
    trap cleanup_log EXIT
    pre_req
    if [ $MIGRATE_SINGLETON -eq 1 ]; then
        info "Found parameter '--operator-namespace', migrating singleton services"
        if [ $ENABLE_LICENSING -eq 1 ]; then
            ${BASE_DIR}/common/migrate_singleton.sh "--operator-namespace" "$OPERATOR_NS" "--enable-licensing"
        else
            ${BASE_DIR}/common/migrate_singleton.sh "--operator-namespace" "$OPERATOR_NS" 
        fi
    fi

    install_cert_manager
    install_licensing
}

function parse_arguments() {
    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --oc)
            shift
            OC=$1
            ;;
        --operator-namespace)
            shift
            OPERATOR_NS=$1
            ;;
        --enable-licensing)
            ENABLE_LICENSING=1
            ;;
        --enable-private-catalog)
            ENABLE_PRIVATE_CATALOG=1
            ;;
        --cert-manager-source)
            shift
            CERT_MANAGER_SOURCE=$1
            ;;
        --licensing-source)
            shift
            LICENSING_SOURCE=$1
            ;;
        --license-accept)
            LICENSE_ACCEPT=1
            ;;
        --check-cert-manager)
            SKIP_INSTALL=1
            ;;
        --check-licensing)
            CHECK_LICENSING_ONLY=1
            SKIP_INSTALL=1
            ;;
        -cmNs | --cert-manager-namespace)
            shift
            CERT_MANAGER_NAMESPACE=$1
            ;;
        -licensingNs | --licensing-namespace)
            shift
            LICENSING_NAMESPACE=$1
            CUSTOMIZED_LICENSING_NAMESPACE=1
            ;;
        -c | --channel)
            shift
            CHANNEL=$1
            ;;
        -i | --install-mode)
            shift
            INSTALL_MODE=$1
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
    echo "Usage: ${script_name} --license-accept [OPTIONS]..."
    echo ""
    echo "Install Cloud Pak 3 pre-reqs if they do not already exist: ibm-cert-manager-operator and optionally ibm-licensing-operator"
    echo "The ibm-cert-manager-operator will be installed in namespace ibm-cert-manager"
    echo "The ibm-licensing-operator will be installed in namespace ibm-licensing"
    echo "The --license-accept must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --operator-namespace string                    Namespace to migrate Cloud Pak 2 Foundational services"
    echo "   --enable-licensing                             Set this flag to install ibm-licensing-operator"
    echo "   --enable-private-catalog                       Set this flag to use namespace scoped CatalogSource. Default is in openshift-marketplace namespace"
    echo "   --cert-manager-source string                   CatalogSource name of ibm-cert-manager-operator. This assumes your CatalogSource is already created. Default is ibm-cert-manager-catalog"
    echo "   --licensing-source string                      CatalogSource name of ibm-licensing. This assumes your CatalogSource is already created. Default is ibm-licensing-catalog"
    echo "   -cmNs, --cert-manager-namespace string         Set custom namespace for ibm-cert-manager-operator. Default is ibm-cert-manager"
    echo "   -licensingNs, --licensing-namespace string     Set custom namespace for ibm-licensing-operator. Default is ibm-licensing"
    echo "   --license-accept                               Set this flag to accept the license agreement."
    echo "   -c, --channel string                           Channel for Subscription(s). Default is v4.0"   
    echo "   -i, --install-mode string                      InstallPlan Approval Mode. Default is Automatic. Set to Manual for manual approval mode"
    echo "   -v, --debug integer                            Verbosity of logs. Default is 0. Set to 1 for debug logs"
    echo "   -h, --help                                     Print usage information"
    echo ""
}

function install_cert_manager() {
    if [ $CHECK_LICENSING_ONLY -eq 1 ]; then
        return
    fi

    title "Installing cert-manager\n"
    is_sub_exist "cert-manager" # this will catch the packagenames of all cert-manager-operators
    if [ $? -eq 0 ]; then
        warning "There is a cert-manager Subscription already\n"
    fi

    pods_exist=$(${OC} get pods -A | grep -w cert-manager-webhook)
    if [ $? -eq 0 ]; then
        warning "There is a cert-manager-webhook pod Running, so most likely another cert-manager is already installed\n"
        return 0
    elif [ $SKIP_INSTALL -eq 1 ]; then
        error "There is no cert-manager-webhook pod running\n"
    fi

    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        SOURCE_NS="${CERT_MANAGER_NAMESPACE}"
    fi

    create_namespace "${CERT_MANAGER_NAMESPACE}"
    create_operator_group "ibm-cert-manager-operator" "${CERT_MANAGER_NAMESPACE}" "{}"
    create_subscription "ibm-cert-manager-operator" "${CERT_MANAGER_NAMESPACE}" "$CHANNEL" "ibm-cert-manager-operator" "${CERT_MANAGER_SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    wait_for_operator "${CERT_MANAGER_NAMESPACE}" "ibm-cert-manager-operator"
    accept_license "certmanagerconfig.operator.ibm.com" "" "default"
}

function install_licensing() {
    if [ $ENABLE_LICENSING -ne 1 ] && [ $CHECK_LICENSING_ONLY -ne 1 ]; then
        return
    fi

    title "Installing licensing\n"
    is_sub_exist "ibm-licensing-operator-app" # this will catch the packagenames of all ibm-licensing-operator-app
    if [ $? -eq 0 ]; then
        warning "There is an ibm-licensing-operator-app Subscription already\n"
        return 0
    elif [ $SKIP_INSTALL -eq 1 ]; then
        error "There is no ibm-licensing-operator-app Subscription installed\n"
    fi

    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        SOURCE_NS="${LICENSING_NAMESPACE}"
    fi

    create_namespace "${LICENSING_NAMESPACE}"

    target=$(cat <<EOF
        
  targetNamespaces:
    - ${LICENSING_NAMESPACE}
EOF
)
    create_operator_group "ibm-licensing-operator-app" "${LICENSING_NAMESPACE}" "$target"
    create_subscription "ibm-licensing-operator-app" "${LICENSING_NAMESPACE}" "$CHANNEL" "ibm-licensing-operator-app" "${LICENSING_SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    wait_for_operator "${LICENSING_NAMESPACE}" "ibm-licensing-operator"
    wait_for_license_instance
    accept_license "ibmlicensing" "" "instance"
}

function wait_for_license_instance() {
    local name="instance"
    local condition="${OC} get ibmlicensing -A --no-headers --ignore-not-found | grep ${name} || true"
    local retries=20
    local sleep_time=15
    local total_time_mins=$(( sleep_time * retries / 60))
    local success_message="ibmlicensing ${name} present"
    local error_message="Timeout after ${total_time_mins} minutes waiting for ibmlicensing ${name} to be present."
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function installing_license_service_reporter() {

  if [ $ENABLE_LSR -ne 1 ] ; then
    return
  fi

  title "Installing License Service Reporter\n"
  is_sub_exist "ibm-license-service-reporter-operator" # this will catch the package names of all ibm-license-service-reporter-operator
  if [ $? -eq 0 ]; then
      warning "There is an ibm-license-service-reporter-operator Subscription already\n"
      return 0
  elif [ $SKIP_INSTALL -eq 1 ]; then
      error "There is no ibm-license-service-reporter-operator Subscription installed\n"
  fi

  if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
      SOURCE_NS="${LSR_NAMESPACE}"
  fi

  create_namespace "${LSR_NAMESPACE}"

  #Prepare LSR PV/PVC which was decoupled in isolate.sh

  lsr_pv_nr=$("${OC}" get pv -l license-service-reporter-pv=true --no-headers | wc -l )
  if [[ lsr_pv_nr -gt 1 ]]; then
    error "More than on PV with label license-service-reporter-pv=true was found. Only one is allowed."
  fi

  if [[ lsr_pv_nr -eq 1 ]]; then
    # get pv name
    LSR_PV_NAME=${OC} get pv -l license-service-reporter-pv=true -o=jsonpath='{.items[0].metadata.name}'

    # get storage class name
    LSR_STORAGE_CLASS=${OC} get pv -l license-service-reporter-pv=true -o=jsonpath='{.items[0].spec.storageClassName}'

    # create PVC
    TEMP_LSR_PVC_FILE="_TMP.yaml"

    cat <<EOF >TEMP_LSR_PVC_FILE
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: license-service-reporter-pvc
      namespace: $LSR_NAMESPACE
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
      storageClassName: "$LSR_STORAGE_CLASS"
      volumeMode: Filesystem
      volumeName: $LSR_PV_NAME
EOF
    ${OC} create -f TEMP_LSR_PVC_FILE
    lsr_pvc_status=$("${OC}" get pvc license-service-reporter-pvc -n $LSR_NAMESPACE --no-headers | awk '{print $2}')
    ${OC} patch pv $LSR_PV_NAME --type=merge -p '{"spec": {"claimRef":null}}'
  fi


  #install LSR operator with cmd

#  target=$(cat <<EOF
#
#  targetNamespaces:
#    - ${LSR_NAMESPACE}
#EOF
#)
  #create_operator_group "ibm-license-service-reporter-operator" "${LSR_NAMESPACE}" "$target"
  create_subscription "ibm-license-service-reporter-operator" "${LSR_NAMESPACE}" "$CHANNEL" "ibm-license-service-reporter-operator" "${LSR_SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
  wait_for_operator "${LSR_NAMESPACE}" "ibm-license-service-reporter-operator"

  #create_reporter_instance


  #wait_for_license_instance
  #accept_license "ibmlicensing" "" "instance"
}

function pre_req() {
    # Check the value of DEBUG
    if [[ "$DEBUG" != "1" && "$DEBUG" != "0" ]]; then
        error "Invalid value for DEBUG. Expected 0 or 1."
    fi

    check_command "${OC}"

    # Checking oc command logged in
    user=$(oc whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi

    if [ "$LICENSE_ACCEPT" -ne 1 ] && [ "$SKIP_INSTALL" -ne 1 ]; then
        error "License not accepted. Rerun script with --license-accept flag set. See https://ibm.biz/integration-licenses for more details"
    fi

    # Check INSTALL_MODE
    if [[ "$INSTALL_MODE" != "Automatic" && "$INSTALL_MODE" != "Manual" ]]; then
        error "Invalid INSTALL_MODE: $INSTALL_MODE, allowed values are 'Automatic' or 'Manual'"
    fi
    
    # Check if channel is semantic vx.y
    if [[ $CHANNEL =~ ^v[0-9]+\.[0-9]+$ ]]; then
        # Check if channel is equal or greater than v4.0
        if [[ $CHANNEL == v[4-9].* || $CHANNEL == v[4-9] ]]; then  
            success "Channel is valid"
        else
            error "Channel is less than v4.0"
        fi
    else
        error "Channel is not semantic vx.y"
    fi

    if [ "$OPERATOR_NS" == "" ]; then
        MIGRATE_SINGLETON=0
    else
        MIGRATE_SINGLETON=1
        get_and_validate_arguments
        if [[ "$ENABLE_LICENSING" == 1 ]];then
            if [[ "$CUSTOMIZED_LICENSING_NAMESPACE" -eq 1 ]] && [[ "$CONTROL_NS" != "$LICENSING_NAMESPACE" ]] && [[ "$CONTROL_NS" != "" ]]; then
                error "Licensing Migration could only be done in $CONTROL_NS, please do not set parameter '-licensingNs $LICENSING_NAMESPACE'"
            elif [[ "$CONTROL_NS" != "" ]]; then
                LICENSING_NAMESPACE="${CONTROL_NS}"
            fi
        fi
    fi
}

# TODO validate argument
function get_and_validate_arguments() {
    get_control_namespace
}

main $*
