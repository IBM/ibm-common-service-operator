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
SERVICES_NS=""
NS_LIST=""
CONTROL_NS=""
CHANNEL="v4.2"
SOURCE="opencloud-operators"
CERT_MANAGER_SOURCE="ibm-cert-manager-catalog"
LICENSING_SOURCE="ibm-licensing-catalog"
SOURCE_NS="openshift-marketplace"
INSTALL_MODE="Automatic"
ENABLE_LICENSING=0
ENABLE_PRIVATE_CATALOG=0
NEW_MAPPING=""
NEW_TENANT=0
DEBUG=0
PREVIEW_MODE=0
LICENSE_ACCEPT=0
ENABLE_LICENSE_SERVICE_REPORTER=0
LSR_NAMESPACE="ibm-lsr"
LSR_SOURCE="ibm-license-service-reporter-bundle-catalog"

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# log file
LOG_FILE="migrate_tenant_log_$(date +'%Y%m%d%H%M%S').log"

# preview mode directory
PREVIEW_DIR="/tmp/preview"

# counter to keep track of installation steps
STEP=0

# ---------- Main functions ----------

. ${BASE_DIR}/common/utils.sh

function main() {
    parse_arguments "$@"
    save_log "logs" "migrate_tenant_log" "$DEBUG"
    trap cleanup_log EXIT
    pre_req
    prepare_preview_mode

    # TODO check Cloud Pak compatibility


    # # Update common-serivce-maps ConfigMap
    # ${BASE_DIR}/common/mapping_tenant.sh --operator-namespace $OPERATOR_NS --services-namespace $SERVICES_NS --tethered-namespaces $TETHERED_NS --control-namespace $CONTROL_NS
    
    # Scale down CS, ODLM and delete OperandReigsrty
    # It helps to prevent re-installing licensing and cert-manager services
    scale_down $OPERATOR_NS $SERVICES_NS $CHANNEL $SOURCE
    local arguments="--yq $YQ --oc $OC"

    # Migrate singleton services
    if [[ $ENABLE_LICENSING -eq 1 ]]; then
        arguments+=" --enable-licensing"
    fi
    
    if [[ $ENABLE_LICENSE_SERVICE_REPORTER -eq 1 ]]; then
        isolate_license_service_reporter
        arguments+=" --enable-license-service-reporter --license-service-reporter-namespace $LSR_NAMESPACE --lsr-source $LSR_SOURCE"
    fi

    if [[ $ENABLE_PRIVATE_CATALOG -eq 1 ]]; then
        arguments+=" --enable-private-catalog"
    fi
    
    ${BASE_DIR}/setup_singleton.sh "--operator-namespace" "$OPERATOR_NS" "-c" "$CHANNEL" "--cert-manager-source" "$CERT_MANAGER_SOURCE" "--licensing-source" "$LICENSING_SOURCE" "--license-accept" $arguments
    if [ $? -ne 0 ]; then
        error "Failed to migrate singleton services"
        exit 1
    fi

    # Update CommonService CR with OPERATOR_NS and SERVICES_NS
    # Propogate CommonService CR to every namespace in the tenant
    update_cscr "$OPERATOR_NS" "$SERVICES_NS" "$NS_LIST"
    accept_license "commonservice" "$OPERATOR_NS"  "common-service"

    # Validate ibm-common-service-operator CatalogSource and CatalogSourceNamespaces
    # Update ibm-common-service-operator channel
    for ns in ${NS_LIST//,/ }; do
        local pm="ibm-common-service-operator"
        local sub_name=$(${OC} get subscription.operators.coreos.com -n ${ns} -l operators.coreos.com/${pm}.${ns}='' --no-headers | awk '{print $1}')
        if [ ! -z "$sub_name" ]; then
            op_source=$SOURCE
            op_source_ns=$SOURCE_NS
            if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
                op_source_ns=$ns
            fi
            validate_operator_catalogsource $pm $ns $op_source $op_source_ns $CHANNEL op_source op_source_ns
            update_operator $pm $ns $CHANNEL $op_source $op_source_ns $INSTALL_MODE
        fi
    done

    # Wait for CS operator upgrade
    wait_for_operator_upgrade $OPERATOR_NS ibm-common-service-operator $CHANNEL $INSTALL_MODE
    # Scale up CS
    scale_up $OPERATOR_NS $SERVICES_NS ibm-common-service-operator ibm-common-service-operator

    # Wait for ODLM upgrade
    wait_for_operator_upgrade $OPERATOR_NS ibm-odlm $CHANNEL $INSTALL_MODE
    # Scale up ODLM
    scale_up $OPERATOR_NS $SERVICES_NS ibm-odlm operand-deployment-lifecycle-manager

    # Clean resources
    cleanup_cp2 "$OPERATOR_NS" "$CONTROL_NS" "$NS_LIST"

    # Update ibm-namespace-scope-operator channel
    is_sub_exist ibm-namespace-scope-operator-restricted $OPERATOR_NS
    if [ $? -eq 0 ]; then
        warning "There is a ibm-namespace-scope-operator-restricted Subscription\n"
        delete_operator ibm-namespace-scope-operator-restricted $OPERATOR_NS
        create_subscription ibm-namespace-scope-operator $OPERATOR_NS $CHANNEL ibm-namespace-scope-operator $SOURCE $SOURCE_NS $INSTALL_MODE
    else
        update_operator ibm-namespace-scope-operator $OPERATOR_NS $CHANNEL $SOURCE $SOURCE_NS $INSTALL_MODE
    fi

    wait_for_operator_upgrade "$OPERATOR_NS" "ibm-namespace-scope-operator" "$CHANNEL" $INSTALL_MODE
    accept_license "namespacescope" "$OPERATOR_NS" "common-service"
    # Authroize NSS operator
    for ns in ${NS_LIST//,/ }; do
        if [ "$ns" != "$OPERATOR_NS" ]; then
            ${BASE_DIR}/common/authorize-namespace.sh $ns -to $OPERATOR_NS
        fi
    done

    # Update NamespaceScope CR common-service
    update_nss_kind "$OPERATOR_NS" "$NS_LIST"

    # Check master CommonService CR status
    wait_for_cscr_status "$OPERATOR_NS" "common-service"
    
    success "Preparation is completed for upgrading Cloud Pak 3.0"
    info "Please update OperandRequest to upgrade foundational core services"
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
        --services-namespace)
            shift
            SERVICES_NS=$1
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
        --enable-license-service-reporter)
            ENABLE_LICENSE_SERVICE_REPORTER=1
            ;;
        --lsr-namespace)
            shift
            LSR_NAMESPACE=$1
            ;;
        --lsr-source)
            shift
            LSR_SOURCE=$1
            ;;
        -c | --channel)
            shift
            CHANNEL=$1
            ;;
        -i | --install-mode)
            shift
            INSTALL_MODE=$1
            ;;
        -s | --source)
            shift
            SOURCE=$1
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
    echo "Usage: ${script_name} --license-accept --operator-namespace <foundational-services-namespace> [OPTIONS]..."
    echo ""
    echo "Migrate Cloud Pak 2.0 Foundational services to Cloud Pak 3.0 Foundational services"
    echo "The --license-accept and --operator-namespace <operator-namespace> must be provided."
    echo "See https://www.ibm.com/docs/en/cloud-paks/foundational-services/4.0?topic=4x-in-place-migration for more information."
    echo ""
    echo "Options:"
    echo "   --oc string                        Optional. File path to oc CLI. Default uses oc in your PATH"
    echo "   --yq string                        Optional. File path to yq CLI. Default uses yq in your PATH"
    echo "   --operator-namespace string        Required. Namespace to migrate Foundational services operator"
    echo "   --services-namespace               Optional. Namespace to migrate operands of Foundational services, i.e. 'dataplane'. Default is the same as operator-namespace"
    echo "   --cert-manager-source string       Optional. CatalogSource name of ibm-cert-manager-operator. This assumes your CatalogSource is already created. Default is ibm-cert-manager-catalog"
    echo "   --licensing-source string          Optional. CatalogSource name of ibm-licensing. This assumes your CatalogSource is already created. Default is ibm-licensing-catalog"
    echo "   --enable-licensing                 Optional. Set this flag to migrate ibm-licensing-operator"
    echo "   --enable-private-catalog           Optional. Set this flag to use namespace scoped CatalogSource. Default is in openshift-marketplace namespace"
    echo "   --license-accept                   Required. Set this flag to accept the license agreement."
    echo "   --enable-license-service-reporter  Optional. Set this flag to migrate ibm-license-service-reporter"
    echo "   --lsr-source string                Optional. CatalogSource name of ibm-license-service-reporter-operator. This assumes your CatalogSource is already created. Default is ibm-license-service-reporter-bundle-catalog"
    echo "   --lsr-namespace                    Optional. Namespace to migrate License Service Reporter. Default is ibm-lsr"
    echo "   -c, --channel string               Optional. Channel for Subscription(s). Default is v4.0"   
    echo "   -i, --install-mode string          Optional. InstallPlan Approval Mode. Default is Automatic. Set to Manual for manual approval mode"
    echo "   -s, --source string                Optional. CatalogSource name. This assumes your CatalogSource is already created. Default is opencloud-operators"
    echo "   -v, --debug integer                Optional. Verbosity of logs. Default is 0. Set to 1 for debug logs."
    echo "   -h, --help                         Print usage information"
    echo ""
}

function pre_req() {
    # Check the value of DEBUG
    if [[ "$DEBUG" != "1" && "$DEBUG" != "0" ]]; then
        error "Invalid value for DEBUG. Expected 0 or 1."
    fi

    check_command "${OC}"
    check_command "${YQ}"

    # TODO: add more compatibility
    # # checking yq version is v4.30+
    # check_version "${YQ}" "--version" "mikefarah" "4\.([3-9][0-9])\.[0-9]+"

    # Checking oc command logged in
    user=$(${OC} whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi

    if [ $LICENSE_ACCEPT -ne 1 ]; then
        error "License not accepted. Rerun script with --license-accept flag set. See https://ibm.biz/integration-licenses for more details"
    fi

    if [ "$OPERATOR_NS" == "" ]; then
        error "Must provide operator namespace"
    fi

    if [ "$SERVICES_NS" == "" ]; then
        SERVICES_NS=$OPERATOR_NS
    fi

    if [ "$CONTROL_NS" == "" ]; then
        CONTROL_NS=$OPERATOR_NS
    fi

    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        SOURCE_NS=$OPERATOR_NS
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
    elif [[ $CHANNEL == "null" ]]; then
        warning "Channel is not set, default channel from operator bundle will be used"
    else
        error "Channel is not semantic vx.y"
    fi

    NS_LIST=$(${OC} get configmap namespace-scope -n ${OPERATOR_NS} -o jsonpath='{.data.namespaces}')
    if [[ -z "$NS_LIST" ]]; then
        error "Failed to get tenant scope from ConfigMap namespace-scope in namespace ${OPERATOR_NS}"
    fi
    
    get_and_validate_arguments
}

function isolate_license_service_reporter(){
    title "Isolating License Service Reporter"

    return_value=$( ("${OC}" get crd ibmlicenseservicereporters.operator.ibm.com > /dev/null && echo exists) || echo fail)
    if [[ $return_value == "exists" ]]; then

        return_value=$("${OC}" get ibmlicenseservicereporters -A --no-headers | wc -l)
        if [[ $return_value -gt 0 ]]; then

            # Change persistentVolumeReclaimPolicy to Retain
            status=$("${OC}" get pvc license-service-reporter-pvc -n $OPERATOR_NS)
            debug1 "LSR pvc status: $status"
            if [[ -z "$status" ]]; then
                error "PVC license-service-reporter-pvc not found in $OPERATOR_NS"
            fi

            VOL=$("${OC}" get pvc license-service-reporter-pvc -n $OPERATOR_NS  -o=jsonpath='{.spec.volumeName}')
            debug1 "LSR volume name: $VOL"
            if [[ -z "$VOL" ]]; then
                error "Volume for pvc license-service-reporter-pvc not found in $OPERATOR_NS"
            fi

            # label LSR PV as LSR PV for further LSR upgrade
            ${OC} label pv $VOL license-service-reporter-pv=true --overwrite 
            ${OC} patch pv $VOL -p '{"spec": { "persistentVolumeReclaimPolicy" : "Retain" }}'
        fi
    fi
    success "License Service Reporter isolated."
}

# TODO validate argument
function get_and_validate_arguments() {
    get_control_namespace
}

main $*
