##!/usr/bin/env bash

NAMESPACES=""

function main(){
    parse_arguments "$@"
    # Checking oc command logged in
    user=$(oc whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
    fi
    label_all_resources
}

function print_usage(){
    script_name=`basename ${0}`
    echo "Usage: ${script_name} [OPTIONS]"
    echo ""
    echo "Label Cert Manager resources (ie certificates, secrets, and issuers) to prepare for Backup."
    echo "If specifying namespaces, make sure to include the operator, services, and all tethered namespaces. If no namespaces are specified, default is all namespaces."
    echo "This script assumes the following:"
    echo "    * An existing CPFS instance installed in the namespaces entered as parameters."
    echo "    * Existing permissions in the specified namespaces."
    echo ""
    echo "Options:"
    echo "   --namespaces                   Optional. Comma-delimited list of namespaces where cert manager resources will be labeled. Default is all namespaces."
    echo "   -h, --help                     Print usage information"
    echo ""
}

function parse_arguments(){
    script_name=`basename ${0}`
    echo "All arguments passed into the ${script_name}: $@"
    echo ""

    # process options
    while [[ "$@" != "" ]]; do
        case "$1" in
        --namespaces)
            shift
            NAMESPACES=$1
            ;;
        -h | --help)
            print_usage
            exit 1
            ;;
        *)
            echo "Entered option $1 not supported. Run ./${script_name} -h for script usage info."
            ;;
        esac
        shift
    done
    echo ""
}

function label_resource(){
    resource=$1
    current_list=$2
    i=0
    len=${#current_list[@]}
    while [ $i -lt $len ];
    do
        NAME=${current_list[$i]}
        let i++
        NAMESPACE=${current_list[$i]}
        let i++
        echo $NAME
        echo $NAMESPACE
        oc label $resource $NAME -n $NAMESPACE foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
        echo "---"
    done
}

function label_specified_secret(){
    namespace=$1
    secret_name=$2
    echo $secret_name
    echo $namespace
    oc label secret $secret_name -n $namespace foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
    echo "---"
}

function label_all_resources(){
    #CRDS
    oc label crd certificates.cert-manager.io foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
    oc label crd issuers.cert-manager.io foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true

    # Get all issuers in all namespaces and add foundationservices.cloudpak.ibm.com=cert-manager
    if [[ $NAMESPACES != "" ]]; then
        NAMESPACES=$(echo "$NAMESPACES" | tr ',' ' ')
        for namespace in $NAMESPACES
        do
            CURRENT_ISSUERS=($(oc get Issuers -n $namespace -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True))
            label_resource Issuers $CURRENT_ISSUERS
            CURRENT_ISSUERS=($(oc get issuers.cert-manager.io -n $namespace -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True))
            label_resource issuers.cert-manager.io $CURRENT_ISSUERS
            CURRENT_CERTIFICATES=($(oc get certificates -n $namespace -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True | grep cs-ca-certificate))
            label_resource certificates $CURRENT_CERTIFICATES
            CURRENT_CERTIFICATES=($(oc get certificates.cert-manager.io -n $namespace -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True | grep cs-ca-certificate))
            label_resource certificates.cert-manager.io $CURRENT_CERTIFICATES
            CURRENT_SECRET=($(oc get secret -n $namespace -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True | grep cs-ca-certificate))
            label_specified_secret $namespace cs-ca-certificate-secret
            
            #zen custom secret and ca-cert-secret
            zen_in_namespace=$(oc get zenservice -n $namespace | awk '{if (NR!=1) {print $1}}' || echo "fail")
            if [[ $zen_in_namespace != "fail" ]]; then 
                zen_secret_name=$(oc get zenservice $zen_in_namespace -n $namespace -o=jsonpath='{.spec.zenCustomRoute.route_secret}')
                label_specified_secret $namespace $zen_secret_name
                label_specified_secret $namespace zen-ca-cert-secret
            else
                echo "[INFO] No zenservices found in namespace $namespace, skipping labeling zen custom route secrets..."
            fi

            #cs on prem config
            cm_namespace_list=$(oc get configmap -n $namespace | grep cs-onprem-tenant-config | awk '{if (NR!=1) {print $1}}' || echo "fail")
            if [[ $cm_namespace_list != "fail" ]]; then
                iam_secret_name=$(oc get configmap cs-onprem-tenant-config -n $namespace -o=jsonpath='{.data.custom_host_certificate_secret}')
                label_specified_secret $namespace $iam_secret_name
            else
                echo "[INFO] Configmap cs-onprem-tenant-config not found in namespace $namespace, skipping copying custom secrets..."
            fi

            #grab default admin credentials
            auth_namespace_list=$(oc get secret -n $namespace | grep platform-auth-idp-credentials | grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
            if [[ $auth_namespace_list != "none" ]]; then
                label_specified_secret $namespace platform-auth-idp-credentials
            else
                echo "[INFO] Secret platform-auth-idp-credentials not present in namespace $namespace. Skipping..."
            fi

            #grab default scim credentials
            scim_secret_namespace_list=$(oc get secret -n $namespace | grep platform-auth-scim-credentials | grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
            if [[ $scim_secret_namespace_list != "none" ]]; then
                label_specified_secret $namespace platform-auth-scim-credentials
            else
                echo "[INFO] Secret platform-auth-scim-credentials not present in namespace $scim_namespace. Skipping..."
            fi

            #grab LDAP TLS certificate
            ldaps_secret_namespace_list=$(oc get secret -n $namespace | grep platform-auth-ldaps-ca-cert | grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
            if [[ $ldaps_secret_namespace_list != "none" ]]; then
                label_specified_secret $namespace platform-auth-ldaps-ca-cert
            else
                echo "[INFO] Secret platform-auth-ldaps-ca-cert not present in namespace $ldaps_namespace. Skipping..."
            fi

            #grab icp service id apikey (if it exists)
            icp_serviceid_apikey_secret_namespace_list=$(oc get secret -n $namespace | grep icp-serviceid-apikey-secret | grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
            if [[ $icp_serviceid_apikey_secret_namespace_list != "none" ]]; then
                label_specified_secret $namespace icp-serviceid-apikey-secret
            else
                echo "[INFO] Secret icp-serviceid-apikey-secret not present in namespace $icp_serviceid_namespace. Skipping..."
            fi

            #grab zen service id apikey (if it exists)
            zen_serviceid_apikey_secret_namespace_list=$(oc get secret -n $namespace | grep zen-serviceid-apikey-secret| grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
            if [[ $zen_serviceid_apikey_secret_namespace_list != "none" ]]; then
                label_specified_secret $namespace zen-serviceid-apikey-secret
            else
                echo "[INFO] Secret zen-serviceid-apikey-secret not present in namespace $zen_serviceid_namespace. Skipping..."
            fi

            #add labels to iaf-system-automation-aui-zen-cert elasticsearch cert/secret
            auto_zen_cert_ns_list=$(oc get certificate -n $namespace --no-headers | grep iaf-system-automationui-aui-zen-cert | awk '{print $1}' | tr "\n" " " || echo "none")
            if [[ $auto_zen_cert_ns_list != "none" ]]; then
                secret_name=$(oc get certificate -n $ns iaf-system-automationui-aui-zen-cert -o jsonpath='{.spec.secretName}')
                label_specified_secret $namespace $secret_name
                oc label certificate iaf-system-automationui-aui-zen-cert -n $namespace foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
            fi

            #add labels to elasticsearch cert/secret
            elasticsearch_cert_ns_list=$(oc get certificate -n $namespace --no-headers | grep iaf-system-elasticsearch-es-client-cert | awk '{print $1}' | tr "\n" " " || echo "none")
            if [[ $elasticsearch_cert_ns_list != "none" ]]; then
                    oc label certificate iaf-system-elasticsearch-es-client-cert -n $ns foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
                    label_specified_secret $namespace iaf-system-elasticsearch-es-client-cert-kp
            fi

            #remove label from metastore-edb certificate and secret
            metastore_secret_ns_list=$(oc get secret -n $namespace --no-headers | grep  ibm-zen-metastore-edb-secret | awk '{print $1}' | tr "\n" " " || echo "none")
            if [[ $metastore_secret_ns_list != "none" ]]; then
                echo "[INFO] removing label from zen-metastore-edb-secret and certificate."
                for ns in $metastore_secret_ns_list
                do
                    oc label secret ibm-zen-metastore-edb-secret -n $namespace foundationservices.cloudpak.ibm.com-
                    oc label certificate ibm-zen-metastore-edb-certificate -n $namespace foundationservices.cloudpak.ibm.com-
                done
            fi
        done
    else
        CURRENT_ISSUERS=($(oc get Issuers --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True))
        label_resource Issuers $CURRENT_ISSUERS

        CURRENT_ISSUERS=($(oc get issuers.cert-manager.io --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True))
        label_resource Issuers $CURRENT_ISSUERS

        # Label all cs-ca-certificates
        CURRENT_CERTIFICATES=($(oc get certificates --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True | grep cs-ca-certificate))
        label_resource certificates $CURRENT_CERTIFICATES

        #cover the different api for certificates
        CURRENT_CERTIFICATES=($(oc get certificates.cert-manager.io --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True | grep cs-ca-certificate))
        label_resource certificates $CURRENT_CERTIFICATES

        CURRENT_SECRET=($(oc get secret --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True | grep cs-ca-certificate-secret))
        label_resource secret $CURRENT_SECRET

        #ensure zenservice custom route secrets are labeled
        zen_namespace_list=$(oc get zenservice -A | awk '{if (NR!=1) {print $1}}' || echo "fail")
        if [[ $zen_namespace_list != "fail" ]]; then 
            for zen_namespace in $zen_namespace_list
            do
                zenservice_list=$(oc get zenservice -n $zen_namespace | awk '{if (NR!=1) {print $1}}')
                for zenservice in $zenservice_list
                do
                    zen_secret_name=$(oc get zenservice $zenservice -n $zen_namespace -o=jsonpath='{.spec.zenCustomRoute.route_secret}')
                    label_specified_secret $zen_namespace $zen_secret_name
                    label_specified_secret $zen_namespace zen-ca-cert-secret
                done
            done
        else
            echo "[INFO] No zenservices found on cluster, skipping labeling zen custom route secrets..."
        fi

        #ensure iam custom route secrets are labeled
        cm_namespace_list=$(oc get configmap -A | grep cs-onprem-tenant-config | awk '{if (NR!=1) {print $1}}' || echo "fail")
        if [[ $cm_namespace_list != "fail" ]]; then
            for tenant_config_namespace in $cm_namespace_list
            do
                iam_secret_name=$(oc get configmap cs-onprem-tenant-config -n $tenant_config_namespace -o=jsonpath='{.data.custom_host_certificate_secret}')
                label_specified_secret $tenant_config_namespace $iam_secret_name
            done
        else
            echo "[INFO] Configmap cs-onprem-tenant-config not found, skipping copying custom secrets..."
        fi

        #grab default admin credentials
        auth_namespace_list=$(oc get secret -A | grep platform-auth-idp-credentials | grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
        if [[ $auth_namespace_list != "none" ]]; then
            for auth_namespace in $auth_namespace_list
            do
                label_specified_secret $auth_namespace platform-auth-idp-credentials
            done
        else
            echo "[INFO] Secret platform-auth-idp-credentials not present in namespace $auth_namespace. Skipping..."
        fi

        #grab default scim credentials
        scim_secret_namespace_list=$(oc get secret -A | grep platform-auth-scim-credentials | grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
        if [[ $scim_secret_namespace_list != "none" ]]; then
            for scim_namespace in $scim_secret_namespace_list
            do
                label_specified_secret $scim_namespace platform-auth-scim-credentials
            done
        else
            echo "[INFO] Secret platform-auth-scim-credentials not present in namespace $scim_namespace. Skipping..."
        fi

        #grab LDAP TLS certificate
        ldaps_secret_namespace_list=$(oc get secret -A | grep platform-auth-ldaps-ca-cert | grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
        if [[ $ldaps_secret_namespace_list != "none" ]]; then
            for ldaps_namespace in $ldaps_secret_namespace_list
            do
                label_specified_secret $ldaps_namespace platform-auth-ldaps-ca-cert
            done
        else
            echo "[INFO] Secret platform-auth-ldaps-ca-cert not present in namespace $ldaps_namespace. Skipping..."
        fi

        #grab icp service id apikey (if it exists)
        icp_serviceid_apikey_secret_namespace_list=$(oc get secret -A | grep icp-serviceid-apikey-secret | grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
        if [[ $icp_serviceid_apikey_secret_namespace_list != "none" ]]; then
            for icp_serviceid_namespace in $icp_serviceid_apikey_secret_namespace_list
            do
                label_specified_secret $icp_serviceid_namespace icp-serviceid-apikey
            done
        else
            echo "[INFO] Secret icp-serviceid-apikey-secret not present in namespace $icp_serviceid_namespace. Skipping..."
        fi

        #grab zen service id apikey (if it exists)
        zen_serviceid_apikey_secret_namespace_list=$(oc get secret -A | grep zen-serviceid-apikey-secret| grep -v "bindinfo" |  awk '{print $1}' | tr "\n" " " || echo "none")
        if [[ $zen_serviceid_apikey_secret_namespace_list != "none" ]]; then
            for zen_serviceid_namespace in $zen_serviceid_apikey_secret_namespace_list
            do
                label_specified_secret $zen_serviceid_namespace zen-serviceid-apikey-secret
            done
        else
            echo "[INFO] Secret zen-serviceid-apikey-secret not present in namespace $zen_serviceid_namespace. Skipping..."
        fi

        #add labels to iaf-system-automation-aui-zen-cert elasticsearch cert/secret
        auto_zen_cert_ns_list=$(oc get certificate -A --no-headers | grep iaf-system-automationui-aui-zen-cert | awk '{print $1}' | tr "\n" " " || echo "none")
        if [[ $auto_zen_cert_ns_list != "none" ]]; then
            for ns in $auto_zen_cert_ns_list
            do
                secret_name=$(oc get certificate -n $ns iaf-system-automationui-aui-zen-cert -o jsonpath='{.spec.secretName}')
                label_specified_secret $ns $secret_name
                oc label certificate iaf-system-automationui-aui-zen-cert -n $ns foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
            done
        fi

        #add labels to elasticsearch cert/secret
        elasticsearch_cert_ns_list=$(oc get certificate -A --no-headers | grep iaf-system-elasticsearch-es-client-cert | awk '{print $1}' | tr "\n" " " || echo "none")
        if [[ $elasticsearch_cert_ns_list != "none" ]]; then
            for ns in $elasticsearch_cert_ns_list
            do
                oc label certificate iaf-system-elasticsearch-es-client-cert -n $ns foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
                label_specified_secret $ns iaf-system-elasticsearch-es-client-cert-kp
            done
        fi

        #remove label from metastore-edb certificate and secret
        metastore_secret_ns_list=$(oc get secret -A --no-headers | grep  ibm-zen-metastore-edb-secret | awk '{print $1}' | tr "\n" " " || echo "none")
        if [[ $metastore_secret_ns_list != "none" ]]; then
            echo "[INFO] removing label from zen-metastore-edb-secret and certificate."
            for ns in $metastore_secret_ns_list
            do
                oc label secret ibm-zen-metastore-edb-secret -n $ns foundationservices.cloudpak.ibm.com-
                oc label certificate ibm-zen-metastore-edb-certificate -n $ns foundationservices.cloudpak.ibm.com-
            done
        fi
    fi

    echo "[SUCCESS] Certificates and secrets successfully labeled."
}

# ---------- Info functions ----------#

function msg() {
    printf '%b\n' "$1"
}

function success() {
    msg "\33[32m[✔] ${1}\33[0m"
}

function error() {
    msg "\33[31m[✘] ${1}\33[0m"
    exit 1
}

function title() {
    msg "\33[34m# ${1}\33[0m"
}

function info() {
    msg "[INFO] ${1}"
}

function warning() {
    msg "\33[33m[✗] ${1}\33[0m"
}

main $*