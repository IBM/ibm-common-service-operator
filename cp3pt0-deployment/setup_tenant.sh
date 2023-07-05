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
CHANNEL="v4.0"
SOURCE="opencloud-operators"
SOURCE_NS="openshift-marketplace"
OPERATOR_NS=""
SERVICES_NS=""
TETHERED_NS=""
EXCLUDED_NS=""
SIZE_PROFILE="starterset"
INSTALL_MODE="Automatic"
PREVIEW_MODE=0
OC_CMD="oc"
DEBUG=0
LICENSE_ACCEPT=0
RETRY_CONFIG_CSCR=0

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

#log file
LOG_FILE="setup_tenant_log_$(date +'%Y%m%d%H%M%S').log"

# counter to keep track of installation steps
STEP=0

# preview mode directory
PREVIEW_DIR="/tmp/preview"

# ---------- Main functions ----------

. ${BASE_DIR}/common/utils.sh

function main() {
    parse_arguments "$@"
    save_log "logs" "setup_tenant_log" "$DEBUG"
    trap cleanup_log EXIT
    pre_req
    prepare_preview_mode
    setup_topology
    check_singleton_service
    setup_nss
    install_cs_operator
}

function parse_arguments() {
    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --oc)
            shift
            OC=$1
            ;;
        --enable-licensing)
            ENABLE_LICENSING=1
            ;;
        --operator-namespace)
            shift
            OPERATOR_NS=$1
            ;;
        --services-namespace)
            shift
            SERVICES_NS=$1
            ;;
        --tethered-namespaces)
            shift
            TETHERED_NS=$1
            ;;
        --excluded-namespaces)
            shift
            EXCLUDED_NS=$1
            ;;
        --license-accept)
            LICENSE_ACCEPT=1
            ;;
        -c | --channel)
            shift
            CHANNEL=$1
            ;;
        -s | --source)
            shift
            SOURCE=$1
            ;;
        -i | --install-mode)
            shift
            INSTALL_MODE=$1
            ;;
        -n | --source-namespace)
            shift
            SOURCE_NS=$1
            ;;
        -p | --size-profile)
            shift
            SIZE_PROFILE=$1
            ;;
        --preview)
            PREVIEW_MODE=1
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
    echo ""
}

function print_usage() {
    script_name=`basename ${0}`
    echo "Usage: ${script_name} --license-accept --operator-namespace <bedrock-namespace> [OPTIONS]..."
    echo ""
    echo "Set up an advanced topology tenant for Cloud Pak 3.0 Foundational services."
    echo "The --license-accept and --operator-namespace must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --enable-licensing             Set this flag to install ibm-licensing-operator"
    echo "   --operator-namespace string    Required. Namespace to install Foundational services operator"
    echo "   --services-namespace           Namespace to install operands of Foundational services, i.e. 'dataplane'. Default is the same as operator-namespace"
    echo "   --tethered-namespaces string   Add namespaces to this tenant, comma-delimited, e.g. 'ns1,ns2'"
    echo "   --excluded-namespaces string   Remove namespaces from this tenant, comma-delimited, e.g. 'ns1,ns2'"
    echo "   --license-accept               Set this flag to accept the license agreement"
    echo "   -c, --channel string           Channel for Subscription(s). Default is v4.0"
    echo "   -i, --install-mode string      InstallPlan Approval Mode. Default is Automatic. Set to Manual for manual approval mode"
    echo "   -s, --source string            CatalogSource name. This assumes your CatalogSource is already created. Default is opencloud-operators"
    echo "   -n, --namespace string         Namespace of CatalogSource. Default is openshift-marketplace"
    echo "   --preview                      Enable preview mode (dry run)"
    echo "   -v, --debug integer            Verbosity of logs. Default is 0. Set to 1 for debug logs"
    echo "   -h, --help                     Print usage information"
    echo "   -p, --size-profile             The default profile is starterset. Change the profile to starter, small, medium, or large, if required"
    echo ""
}

function pre_req() {
    title "Start to validate the parameters passed into script... "
    # Check the value of DEBUG
    if [[ "$DEBUG" != "1" && "$DEBUG" != "0" ]]; then
        error "Invalid value for DEBUG. Expected 0 or 1."
    fi

    if [ $PREVIEW_MODE -eq 1 ]; then
        info "Running in preview mode. No actual changes will be made."
    fi

    check_command "${OC}"

    # Checking oc command logged in
    user=$($OC whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi
    
    if [ $? -ne 0 ]; then
        error "Cert-manager is not found or having more than one\n"
    fi

    if [ $LICENSE_ACCEPT -ne 1 ]; then
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

    # Check profile size
    case "$SIZE_PROFILE" in
    "starterset"|"starter"|"small"|"medium"|"large")
        success "Profile size is valid."
        ;;
    *)
        error " '$SIZE_PROFILE' is not a valid value for profile. Allowed values are 'starterset', 'starter', 'small', 'medium', and 'large'."
        ;;
    esac

    if [ "$OPERATOR_NS" == "" ]; then
        error "Must provide operator namespace, please specify argument --operator-namespace"
    fi

    if [[ "$SERVICES_NS" == "" && "$TETHERED_NS" == "" && "$EXCLUDED_NS" == "" ]]; then
        error "Must provide additional namespaces, either --services-namespace, --tethered-namespaces, or --excluded-namespaces"
    fi

    if [[ "$SERVICES_NS" == "$OPERATOR_NS" && "$TETHERED_NS" == "" && "$EXCLUDED_NS" == "" ]]; then
        error "Must provide additional namespaces for --tethered-namespaces or --excluded-namespaces when services-namespace is the same as operator-namespace"
    fi

    if [[ "$TETHERED_NS" == "$OPERATOR_NS" || "$TETHERED_NS" == "$SERVICES_NS" ]]; then
        error "Must provide additional namespaces for --tethered-namespaces, different from operator-namespace and services-namespace"
    fi
    echo ""
}

function prepare_preview_mode() {
    mkdir -p ${PREVIEW_DIR}
    if [ $PREVIEW_MODE -eq 1 ]; then
        OC_CMD="oc --dry-run=client" # a redirect to the file is needed too
    fi
}

function create_ns_list() {
    for ns in $OPERATOR_NS $SERVICES_NS ${TETHERED_NS//,/ }; do
        create_namespace $ns
        if [ $? -ne 0 ]; then
            error "Namespace $ns cannot be created, please ensure user $user has proper permission to create namepace\n"
        fi
    done
}

function setup_topology() {
    create_ns_list
    target=$(cat <<EOF

  targetNamespaces:
    - $OPERATOR_NS
EOF
)
    create_operator_group "common-service" "$OPERATOR_NS" "$target"
    if [ $? -ne 0 ]; then
        error "Operatorgroup cannot be created in namespace $OPERATOR_NS, please ensure user $user has proper permission to create Operatorgroup\n"
    fi
}

function check_singleton_service() {
    check_cert_manager "cert-manager" "$OPERATOR_NS"
    if [ $ENABLE_LICENSING -eq 1 ]; then
        check_licensing
        if [ $? -ne 0 ]; then
            error "ibm-licensing is not found or having more than one\n"
        fi
    fi
}

function install_nss() {
    title "Checking whether Namespace Scope operator exist..."

    is_sub_exist "ibm-namespace-scope-operator" "$OPERATOR_NS"
    if [ $? -eq 0 ]; then
        warning "There is an ibm-namespace-scope-operator subscription already deployed\n"
    else
        create_subscription "ibm-namespace-scope-operator" "$OPERATOR_NS" "$CHANNEL" "ibm-namespace-scope-operator" "${SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    fi

    if [ $PREVIEW_MODE -eq 0 ]; then
        wait_for_operator "$OPERATOR_NS" "ibm-namespace-scope-operator" 
    fi
    
    # namespaceMembers should at least have Bedrock operators' namespace
    local ns=$(cat <<EOF

    - $OPERATOR_NS
EOF
    )

    title "Adding the tethered optional namespaces and removing excluded namespaces for a tenant to namespaceMembers..."
    # add the tethered optional namespaces for a tenant to namespaceMembers
    # ${TETHERED_NS} is comma delimited, so need to replace commas with space
    local nss_exists="fail"
    if [ $PREVIEW_MODE -eq 0 ]; then
        nss_exists=$(${OC} get nss common-service -n $OPERATOR_NS || echo "fail")
    fi 

    if [[ $nss_exists != "fail" ]]; then
        debug1 "NamspaceScope common-service exists in namespace $OPERATOR_NS."
        existing_ns=$(${OC} get nss common-service -n $OPERATOR_NS -o=jsonpath='{.spec.namespaceMembers}' | tr -d \" | tr -d [ | tr -d ])
        existing_ns="${existing_ns//,/ }"

        # remove the excluded namespaces from the list
        if [[ $EXCLUDED_NS != "" ]]; then
            info "Removing excluded namespaces from common-service NamespaceScope"
            remove_ns="${EXCLUDED_NS//,/ }"
            tmp_ns_list=""
            for namespace in $existing_ns
            do
                # check if namespace is in the list of excluded namespaces
                contains_ns=$([[ ${remove_ns[@]} =~ $namespace ]] || echo "false")
                if [[ $contains_ns == "false" ]]; then
                    if [[ $tmp_ns_list == "" ]]; then
                        tmp_ns_list="${namespace}"
                    else
                        tmp_ns_list="${tmp_ns_list} ${namespace}"
                    fi
                fi
            done
            existing_ns="${tmp_ns_list}"
        fi
        new_ns_list=$(echo ${existing_ns} ${TETHERED_NS//,/ } ${SERVICES_NS} | xargs -n1 | sort -u | xargs)
    else
        new_ns_list=$(echo ${TETHERED_NS//,/ } ${SERVICES_NS} | xargs -n1 | sort -u | xargs)
    fi
    debug1 "List of namespaces for common-service NSS ${new_ns_list}"
    
    # add the new namespaces from list of common-service NSS to namespaceMembers
    for n in ${new_ns_list}; do
        if [[ $n == $OPERATOR_NS ]]; then
            continue
        fi
        local ns=$ns$(cat <<EOF

    - $n
EOF
    )
    done

    debug1 "Format of namespaceMembers to be added: $ns\n"

    configure_nss_kind "$ns"
    if [ $? -ne 0 ]; then
        error "Failed to create NSS CR in ${OPERATOR_NS}"
    fi
    accept_license "namespacescope" "$OPERATOR_NS" "common-service"
}

function authorize_nss() {

    cat <<EOF > ${PREVIEW_DIR}/role.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: nss-managed-role-from-$OPERATOR_NS
  namespace: ns_to_replace
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

    cat <<EOF > ${PREVIEW_DIR}/rolebinding.yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: nss-managed-role-from-$OPERATOR_NS
  namespace: ns_to_replace
subjects:
- kind: ServiceAccount
  name: ibm-namespace-scope-operator
  namespace: $OPERATOR_NS
roleRef:
  kind: Role
  name: nss-managed-role-from-$OPERATOR_NS
  apiGroup: rbac.authorization.k8s.io
EOF

    title "Checking and authorizing NSS to all namespaces in tenant..."
    for ns in $SERVICES_NS ${TETHERED_NS//,/ }; do

        if [[ $($OC get RoleBinding nss-managed-role-from-$OPERATOR_NS -n $ns 2>/dev/null) != "" ]];then
            info "RoleBinding nss-managed-role-from-$OPERATOR_NS is already existed in $ns, skip creating\n"
        else
            debug1 "Creating following Role:\n"
            cat ${PREVIEW_DIR}/role.yaml | sed "s/ns_to_replace/$ns/g"
            echo ""
            cat ${PREVIEW_DIR}/role.yaml | sed "s/ns_to_replace/$ns/g" | ${OC_CMD} apply -f -
            if [[ $? -ne 0 ]]; then
                error "Failed to create Role for NSS in namespace $ns, please check if user has proper permission to create role\n"
            fi

            debug1 "Creating following RoleBinding:\n"
            cat ${PREVIEW_DIR}/rolebinding.yaml | sed "s/ns_to_replace/$ns/g"
            echo ""
            cat ${PREVIEW_DIR}/rolebinding.yaml | sed "s/ns_to_replace/$ns/g" | ${OC_CMD} apply -f -
            if [[ $? -ne 0 ]]; then
                error "Failed to create RoleBinding for NSS in namespace $ns, please check if user has proper permission to create rolebinding\n"
            fi
        fi
    done
}

function setup_nss() {
    install_nss
    authorize_nss
}

function install_cs_operator() {
    title "Installing IBM Foundational services operator into operator namespace ${OPERATOR_NS}..."

    title "checking if CommonService CRD exists in the cluster..."
    local is_CS_CRD_exist=$(($OC get commonservice -n "$OPERATOR_NS" --ignore-not-found > /dev/null && echo exists) || echo fail)

    if [ "$is_CS_CRD_exist" == "exists" ]; then
        info "CommonService CRD exist\n"
        configure_cs_kind
    else
        info "CommonService CRD does not exist, installing ibm-common-service-operator first\n"
    fi

    title "Checking whether IBM Common Service operator exist..."
    is_sub_exist "ibm-common-service-operator" "$OPERATOR_NS"
    if [ $? -eq 0 ]; then
        info "There is an ibm-common-service-operator Subscription already\n"
    else
        create_subscription "ibm-common-service-operator" "$OPERATOR_NS" "$CHANNEL" "ibm-common-service-operator" "${SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
        # sleep 120
    fi

    if [ $PREVIEW_MODE -eq 0 ]; then
        wait_for_operator "$OPERATOR_NS" "ibm-common-service-operator"
        accept_license "commonservice" "$OPERATOR_NS" "common-service"
        wait_for_nss_patch "$OPERATOR_NS" 
        wait_for_cs_webhook "$OPERATOR_NS" "ibm-common-service-webhook"
    else
        info "Preview mode is on, skip waiting for operator and webhook being ready\n"
    fi
    
    if [ "$is_CS_CRD_exist" == "fail" ] || [ $RETRY_CONFIG_CSCR -eq 1 ]; then
        RETRY_CONFIG_CSCR=1
        configure_cs_kind
    fi
}

function configure_nss_kind() {
    local members=$1

    title "Configuring NamespaceScope CR in $OPERATOR_NS..."
    if [[ $($OC get NamespaceScope common-service -n $OPERATOR_NS 2>/dev/null) != "" ]];then
        info "NamespaceScope CR is already deployed in $OPERATOR_NS\n"
    else
        info "Creating the NamespaceScope object:\n"
    fi

    cat <<EOF > ${PREVIEW_DIR}/namespacescope.yaml
apiVersion: operator.ibm.com/v1
kind: NamespaceScope
metadata:
  name: common-service
  namespace: $OPERATOR_NS
spec:
  csvInjector:
    enable: true
  namespaceMembers: $members
  restartLabels:
    intent: projected
EOF

    cat ${PREVIEW_DIR}/namespacescope.yaml
    echo ""
    cat "${PREVIEW_DIR}/namespacescope.yaml" | ${OC_CMD} apply -f -
}

function configure_cs_kind() {
    local retries=5
    local delay=30

    title "Configuring CommonService CR in $OPERATOR_NS..."
    if [[ $($OC get CommonService common-service -n $OPERATOR_NS 2>/dev/null) != "" ]];then
        info "CommonService CR is already deployed in $OPERATOR_NS\n"
    else
        info "Creating the CommonService object:\n"
    fi

    cat <<EOF > ${PREVIEW_DIR}/commonservice.yaml
apiVersion: operator.ibm.com/v3
kind: CommonService
metadata:
  name: common-service
  namespace: $OPERATOR_NS
spec:
  operatorNamespace: $OPERATOR_NS
  servicesNamespace: $SERVICES_NS
  size: $SIZE_PROFILE
EOF

    cat ${PREVIEW_DIR}/commonservice.yaml
    echo ""

    while [ $retries -gt 0 ]; do
        
        cat "${PREVIEW_DIR}/commonservice.yaml" | ${OC_CMD} apply -f -
    
        # Check if the patch was successful
        if [[ $? -eq 0 ]]; then
            success "Successfully patched CommonService CR in ${OPERATOR_NS}\n"
            break
        else
            warning "Failed to patch CommonService CR in ${OPERATOR_NS}, retry it in ${delay} seconds..."
            sleep ${delay}
            retries=$((retries-1))
        fi
    done

    if [ $retries -eq 0 ] && [ $RETRY_CONFIG_CSCR -eq 1 ]; then
        error "Fail to patch CommonService CR in ${OPERATOR_NS}\n"
    fi

    if [ $retries -eq 0 ] && [ $RETRY_CONFIG_CSCR -eq 0 ]; then
        warning "Fail to patch CommonService CR in ${OPERATOR_NS}, try to install cs-operator first"
        RETRY_CONFIG_CSCR=1
    fi
}

main $*