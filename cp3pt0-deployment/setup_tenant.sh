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
SIZE_PROFILE="small"
INSTALL_MODE="Automatic"
DEBUG=0

# ---------- Command variables ----------

# script base directory
BASE_DIR=$(cd $(dirname "$0")/$(dirname "$(readlink $0)") && pwd -P)

# counter to keep track of installation steps
STEP=0

# ---------- Main functions ----------

. ${BASE_DIR}/common/utils.sh

function main() {
    parse_arguments "$@"
    pre_req
    setup_topology
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
    echo "Usage: ${script_name} --operator-namespace <bedrock-namespace> [OPTIONS]..."
    echo ""
    echo "Set up an advanced topology tenant for Cloud Pak 3.0 Foundational services."
    echo "The --operator-namespace must be provided."
    echo ""
    echo "Options:"
    echo "   --oc string                    File path to oc CLI. Default uses oc in your PATH"
    echo "   --enable-licensing             Set this flag to install ibm-licensing-operator"
    echo "   --operator-namespace string    Required. Namespace to install Foundational services operator"
    echo "   --services-namespace           Namespace to install operands of Foundational services, i.e. 'dataplane'. Default is the same as operator-namespace"
    echo "   --tethered-namespaces string   Additional namespaces for this tenant, comma-delimited, e.g. 'ns1,ns2'"
    echo "   -c, --channel string           Channel for Subscription(s). Default is v4.0"
    echo "   -i, --install-mode string      InstallPlan Approval Mode. Default is Automatic. Set to Manual for manual approval mode"
    echo "   -s, --source string            CatalogSource name. This assumes your CatalogSource is already created. Default is opencloud-operators"
    echo "   -n, --namespace string         Namespace of CatalogSource. Default is openshift-marketplace"
    echo "   --limited-access-mode string   Default is false, if set to true will throw error when require resources are not found"
    echo "   -v, --debug integer            Verbosity of logs. Default is 0. Set to 1 for debug logs."
    echo "   -h, --help                     Print usage information"
    echo ""
}

function pre_req() {
    check_command "${OC}"

    # checking oc command logged in
    user=$($OC whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi

    check_cert_manager "cert-manager"
    if [ $? -ne 0 ]; then
        error "Cert-manager is not found or having more than one\n"
    fi

    if [ $ENABLE_LICENSING -eq 1 ]; then
        check_licensing
        if [ $? -ne 0 ]; then
            error "ibm-licensing is not found or having more than one\n"
        fi
    fi

    if [ "$OPERATOR_NS" == "" ]; then
        error "Must provide operator namespace, please specify argument --operator-namespace"
    fi

    if [[ "$SERVICES_NS" == "" && "$TETHERED_NS" == "" ]]; then
        error "Must provide additional namespaces, either --services-namespace or --tethered-namespaces"
    fi

    if [[ "$SERVICES_NS" == "$OPERATOR_NS" && "$TETHERED_NS" == "" ]]; then
        error "Must provide additional namespaces for --tethered-namespaces when services-namespace is the same as operator-namespace"
    fi

    if [[ "$TETHERED_NS" == "$OPERATOR_NS" || "$TETHERED_NS" == "$SERVICES_NS" ]]; then
        error "Must provide additional namespaces for --tethered-namespaces, different from operator-namespace and services-namespace"
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

function setup_nss() {
    install_nss
    authorize_nss
}

function install_nss() {
    title "Installing Namespace Scope operator\n"

    is_sub_exist "ibm-namespace-scope-operator" "$OPERATOR_NS"
    if [ $? -eq 0 ]; then
        warning "There is an ibm-namespace-scope-operator subscription already deployed\n"
    else
        create_subscription "ibm-namespace-scope-operator" "$OPERATOR_NS" "$CHANNEL" "ibm-namespace-scope-operator" "${SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
    fi

    wait_for_operator "$OPERATOR_NS" "ibm-namespace-scope-operator"

    # namespaceMembers should at least have Bedrock operators' namespace
    local ns=$(cat <<EOF

    - $OPERATOR_NS
EOF
    )

    # add the tethered optional namespaces for a tenant to namespaceMembers
    # ${TETHERED_NS} is comma delimited, so need to replace commas with space
    for n in $SERVICES_NS ${TETHERED_NS//,/ }; do
        local ns=$ns$(cat <<EOF

    - $n
EOF
    )
    done

    configure_nss_kind "$ns"
    if [ $? -ne 0 ]; then
        error "Failed to create NSS CR in ${OPERATOR_NS}"
    fi
}

function authorize_nss() {

    local role=$(
        cat <<EOF
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
)

    local rb=$(
        cat <<EOF
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
)
    title "Checking and authorizing NSS to all namespaces in tenant\n"
    for ns in $SERVICES_NS ${TETHERED_NS//,/ }; do

        if [[ $($OC get RoleBinding nss-managed-role-from-$OPERATOR_NS -n $ns 2>/dev/null) != "" ]];then
            info "RoleBinding nss-managed-role-from-$OPERATOR_NS is already existed in $ns, skip creating"
        else
            debug1 "Creating following Role:\n"
            debug1 "${role//ns_to_replace/$ns}\n"
            echo "${role//ns_to_replace/$ns}" | ${OC} apply -f -
            if [[ $? -ne 0 ]]; then
                error "Failed to create Role for NSS in namespace $ns, please check if user has proper permission to create role"
            fi

            debug1 "Creating following RoleBinding:\n"
            debug1 "${rb//ns_to_replace/$ns}\n"
            echo "${rb//ns_to_replace/$ns}" | ${OC} apply -f -
            if [[ $? -ne 0 ]]; then
                error "Failed to create RoleBinding for NSS in namespace $ns, please check if user has proper permission to create rolebinding"
            fi
        fi
    done
}

function install_cs_operator() {
    msg "Installing IBM Foundational services operator into operator namespace - ${OPERATOR_NS}"

    is_sub_exist "ibm-common-service-operator" "$OPERATOR_NS"
    if [ $? -eq 0 ]; then
        info "There is an ibm-common-service-operator Subscription already\n"
    else
        create_subscription "ibm-common-service-operator" "$OPERATOR_NS" "$CHANNEL" "ibm-common-service-operator" "${SOURCE}" "${SOURCE_NS}" "${INSTALL_MODE}"
        sleep 120
    fi
    wait_for_operator "$OPERATOR_NS" "ibm-common-service-operator"
    configure_cs_kind
}

function configure_nss_kind() {
    local members=$1

    if [[ $($OC get NamespaceScope common-service -n $OPERATOR_NS 2>/dev/null) != "" ]];then
        title "NamespaceScope CR is already deployed in $OPERATOR_NS"
    else
        title "Creating the NamespaceScope object"
    fi
    local object=$(
    cat <<EOF
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
    )
    echo
    echo "$object"
    echo "$object" | ${OC} apply -f -
}

function configure_cs_kind() {
    local object=$(
        cat <<EOF
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
    )

    echo
    info "Configuring the CommonService object"
    echo "$object"
    echo "$object" | ${OC} apply -f -
    if [[ $? -ne 0 ]]; then
        error "Failed to create CommonService CR in ${OPERATOR_NS}"
    fi
}

function debug1() {
    if [ $DEBUG -eq 1 ]; then
        debug "${1}"
    fi
}

main $*
