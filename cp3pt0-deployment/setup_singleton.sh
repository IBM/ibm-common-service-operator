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
ENABLE_LICENSING=0
ENABLE_PRIVATE_CATALOG=0
MIGRATE_SINGLETON=0
OPERATOR_NS=""
CONTROL_NS=""
CHANNEL="v4.1"
SOURCE_NS="openshift-marketplace"
INSTALL_MODE="Automatic"
CERT_MANAGER_SOURCE="ibm-cert-manager-catalog"
LICENSING_SOURCE="ibm-licensing-catalog"
CERT_MANAGER_NAMESPACE="ibm-cert-manager"
LICENSING_NAMESPACE="ibm-licensing"
LICENSE_ACCEPT=0
DEBUG=0

CUSTOMIZED_LICENSING_NAMESPACE=0
SKIP_INSTALL=0
CHECK_LICENSING_ONLY=0
CERT_MANAGER_V1_OWNER="operator.ibm.com/v1"
CERT_MANAGER_V1ALPHA1_OWNER="operator.ibm.com/v1alpha1"

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# log file
LOG_FILE="setup_singleton_log_$(date +'%Y%m%d%H%M%S').log"

# preview mode directory
PREVIEW_DIR="/tmp/preview"

# counter to keep track of installation steps
STEP=0

# ---------- Main functions ----------

. ${BASE_DIR}/common/utils.sh

function main() {
    parse_arguments "$@"
    save_log "logs" "setup_singleton_log" "$DEBUG"
    trap cleanup_log EXIT
    pre_req
    prepare_preview_mode

    is_migrate_licensing
    is_migrate_cert_manager

    if [ $MIGRATE_SINGLETON -eq 1 ]; then
        if [ $ENABLE_LICENSING -eq 1 ]; then
            ${BASE_DIR}/common/migrate_singleton.sh "--operator-namespace" "$OPERATOR_NS" --control-namespace "$CONTROL_NS" "--enable-licensing" --licensing-namespace "$LICENSING_NAMESPACE"
        else
            ${BASE_DIR}/common/migrate_singleton.sh "--operator-namespace" "$OPERATOR_NS" --control-namespace "$CONTROL_NS"
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
    echo "See https://www.ibm.com/docs/en/cloud-paks/foundational-services/4.0?topic=manager-installing-cert-licensing-by-script for more information."
    echo ""
    echo "Options:"
    echo "   --oc string                                    Optional. File path to oc CLI. Default uses oc in your PATH"
    echo "   --yq string                                    Optional. File path to yq CLI. Default uses yq in your PATH"
    echo "   --operator-namespace string                    Optional. Namespace to migrate Cloud Pak 2 Foundational services"
    echo "   --enable-licensing                             Optional. Set this flag to install ibm-licensing-operator"
    echo "   --enable-private-catalog                       Optional. Set this flag to use namespace scoped CatalogSource. Default is in openshift-marketplace namespace"
    echo "   --cert-manager-source string                   Optional. CatalogSource name of ibm-cert-manager-operator. This assumes your CatalogSource is already created. Default is ibm-cert-manager-catalog"
    echo "   --licensing-source string                      Optional. CatalogSource name of ibm-licensing. This assumes your CatalogSource is already created. Default is ibm-licensing-catalog"
    echo "   -cmNs, --cert-manager-namespace string         Optional. Set custom namespace for ibm-cert-manager-operator. Default is ibm-cert-manager"
    echo "   -licensingNs, --licensing-namespace string     Optional. Set custom namespace for ibm-licensing-operator. Default is ibm-licensing"
    echo "   --license-accept                               Required. Set this flag to accept the license agreement."
    echo "   -c, --channel string                           Optional. Channel for Subscription(s). Default is v4.0"   
    echo "   -i, --install-mode string                      Optional. InstallPlan Approval Mode. Default is Automatic. Set to Manual for manual approval mode"
    echo "   -v, --debug integer                            Optional. Verbosity of logs. Default is 0. Set to 1 for debug logs"
    echo "   -h, --help                                     Print usage information"
    echo ""
}

function is_migrate_cert_manager() {
    title "Check migrating and deactivating LTSR ibm-cert-manager-operator"
    local webhook_ns=$("$OC" get deployments -A | grep cert-manager-webhook | cut -d ' ' -f1)
    if [ -z "$webhook_ns" ]; then
        info "No cert-manager-webhook found, skipping migration"
        return 0
    fi
    local api_version=$("$OC" get deployments -n "$webhook_ns" cert-manager-webhook -o jsonpath='{.metadata.ownerReferences[*].apiVersion}')
    if [ "$api_version" != "$CERT_MANAGER_V1ALPHA1_OWNER" ]; then
        info "LTSR ibm-cert-manager-operator already deactivated, skipping"
        return 0
    fi
    MIGRATE_SINGLETON=1
    get_and_validate_arguments
}

function is_migrate_licensing() {
    if [ $ENABLE_LICENSING -ne 1 ] && [ $CHECK_LICENSING_ONLY -ne 1 ]; then
        return
    fi

    title "Check migrating LTSR ibm-licensing-operator"
    
    local version=$("$OC" get ibmlicensing instance -o jsonpath='{.spec.version}')
    if [ -z "$version" ]; then
        warning "No version field in ibmlicensing CR, skipping"
        return 0
    fi
    local major=$(echo "$version" | cut -d '.' -f1)
    if [ "$major" -ge 4 ]; then
        info "There is no LTSR ibm-licensing-operator to migrate, skipping"
        return 0
    fi

    local ns=$("$OC" get deployments -A | grep ibm-licensing-operator | cut -d ' ' -f1)
    if [ -z "$ns" ]; then
        info "No LTSR ibm-licensing-operator to migrate, skipping"
        return 0
    fi

    get_and_validate_arguments
    if [ ! -z "$CONTROL_NS" ]; then
        if [[ "$CUSTOMIZED_LICENSING_NAMESPACE" -eq 1 ]] && [[ "$CONTROL_NS" != "$LICENSING_NAMESPACE" ]]; then
            error "Licensing Migration could only be done in $CONTROL_NS, please do not set parameter '-licensingNs $LICENSING_NAMESPACE'"
        fi
        LICENSING_NAMESPACE="$CONTROL_NS"
    fi

    MIGRATE_SINGLETON=1
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

    local webhook_ns=$("$OC" get deployments -A | grep cert-manager-webhook | cut -d ' ' -f1)
    if [ ! -z "$webhook_ns" ]; then
        warning "There is a cert-manager-webhook pod Running, so most likely another cert-manager is already installed\n"
        info "Continue to upgrade check\n"
    elif [ $SKIP_INSTALL -eq 1 ]; then
        error "There is no cert-manager-webhook pod running\n"
    fi

    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        SOURCE_NS="${CERT_MANAGER_NAMESPACE}"
    fi

    local api_version=$("$OC" get deployments -n "$webhook_ns" cert-manager-webhook -o jsonpath='{.metadata.ownerReferences[*].apiVersion}')
    if [ ! -z "$api_version" ]; then
        if [ "$api_version" == "$CERT_MANAGER_V1ALPHA1_OWNER" ]; then
            error "Cluster has not deactivated LTSR ibm-cert-manager-operator yet, please re-run this script"
        fi

        if [ "$api_version" != "$CERT_MANAGER_V1_OWNER" ]; then
            warning "Cluster has a non ibm-cert-manager-operator already installed, skipping"
            return 0
        fi
        
        info "Upgrading ibm-cert-manager-operator to channel: $CHANNEL\n"
    fi
    
    create_namespace "${CERT_MANAGER_NAMESPACE}"
    create_operator_group "ibm-cert-manager-operator" "${CERT_MANAGER_NAMESPACE}" "{}"
    is_sub_exist "ibm-cert-manager-operator" "${CERT_MANAGER_NAMESPACE}" # this will catch the packagenames of all cert-manager-operators
    if [ $? -eq 0 ]; then
        update_operator "ibm-cert-manager-operator" "${CERT_MANAGER_NAMESPACE}" "$CHANNEL" "${CERT_MANAGER_SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    else
        create_subscription "ibm-cert-manager-operator" "${CERT_MANAGER_NAMESPACE}" "$CHANNEL" "ibm-cert-manager-operator" "${CERT_MANAGER_SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    fi
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
        warning "There is an ibm-licensing-operator-app Subscription already, so will upgrade it\n"
    elif [ $SKIP_INSTALL -eq 1 ]; then
        error "There is no ibm-licensing-operator-app Subscription installed\n"
    fi

    if [ $ENABLE_PRIVATE_CATALOG -eq 1 ]; then
        SOURCE_NS="${LICENSING_NAMESPACE}"
    fi

    local ns=$("$OC" get deployments -A | grep ibm-licensing-operator | cut -d ' ' -f1)
    if [ ! -z "$ns" ]; then
        if [ "$ns" != "$LICENSING_NAMESPACE" ]; then
            error "An ibm-licensing-operator already installed in namespace: $ns, expected namespace is: $LICENSING_NAMESPACE"
        fi
    fi

    create_namespace "${LICENSING_NAMESPACE}"

    target=$(cat <<EOF
        
  targetNamespaces:
    - ${LICENSING_NAMESPACE}
EOF
)
    create_operator_group "ibm-licensing-operator-app" "${LICENSING_NAMESPACE}" "$target"
    is_sub_exist "ibm-licensing-operator-app" # this will catch the packagenames of all ibm-licensing-operator-app
    if [ $? -eq 0 ]; then
        update_operator "ibm-licensing-operator-app" "${LICENSING_NAMESPACE}" "$CHANNEL" "${LICENSING_SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    else
        create_subscription "ibm-licensing-operator-app" "${LICENSING_NAMESPACE}" "$CHANNEL" "ibm-licensing-operator-app" "${LICENSING_SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    fi
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
    local wait_message="Waiting for ibmlicensing ${name} to be present."
    local success_message="ibmlicensing ${name} present"
    local error_message="Timeout after ${total_time_mins} minutes waiting for ibmlicensing ${name} to be present."
    wait_for_condition "${condition}" ${retries} ${sleep_time} "${wait_message}" "${success_message}" "${error_message}"
}

function pre_req() {
    # Check the value of DEBUG
    if [[ "$DEBUG" != "1" && "$DEBUG" != "0" ]]; then
        error "Invalid value for DEBUG. Expected 0 or 1."
    fi

    check_command "${OC}"
    check_command "${YQ}"

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

    # Check if all CS installations are above 3.19.9
    local csvs=$("$OC" get csv -A | grep ibm-common-service-operator | awk '{print $2}' | sort -V)
    local version=$(echo "$csvs" | head -n 1 | cut -d '.' -f2-)
    is_supports_delegation "$version"

    if [ -z "$OPERATOR_NS" ]; then
        OPERATOR_NS=$("$OC" project --short)
    fi
}

# TODO validate argument
function get_and_validate_arguments() {
    get_control_namespace
}

main "$@"
