##!/usr/bin/env bash

# Get all issuers in all namespaces and add foundationservices.cloudpak.ibm.com=cert-manager
CURRENT_ISSUERS=($(oc get Issuers --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True))
i=0
len=${#CURRENT_ISSUERS[@]}
while [ $i -lt $len ];
do
    NAME=${CURRENT_ISSUERS[$i]}
    let i++
    NAMESPACE=${CURRENT_ISSUERS[$i]}
    let i++
    echo $NAME
    echo $NAMESPACE
    echo "---"
    oc label issuer $NAME -n $NAMESPACE foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
done

CURRENT_ISSUERS=($(oc get issuers.cert-manager.io --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True))
i=0
len=${#CURRENT_ISSUERS[@]}
while [ $i -lt $len ];
do
    NAME=${CURRENT_ISSUERS[$i]}
    let i++
    NAMESPACE=${CURRENT_ISSUERS[$i]}
    let i++
    echo $NAME
    echo $NAMESPACE
    echo "---"
    oc label issuer $NAME -n $NAMESPACE foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
done

# Get all certificates in all namespaces and add foundationservices.cloudpak.ibm.com=cert-manager
CURRENT_CERTIFICATES=($(oc get certificates --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True))
i=0
len=${#CURRENT_CERTIFICATES[@]}
while [ $i -lt $len ];
do
    NAME=${CURRENT_CERTIFICATES[$i]}
    let i++
    NAMESPACE=${CURRENT_CERTIFICATES[$i]}
    let i++
    echo $NAME
    echo $NAMESPACE
    echo "---"
    oc label certificates $NAME -n $NAMESPACE foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
done

CURRENT_CERTIFICATES=($(oc get certificates.cert-manager.io --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True))
i=0
len=${#CURRENT_CERTIFICATES[@]}
while [ $i -lt $len ];
do
    NAME=${CURRENT_CERTIFICATES[$i]}
    let i++
    NAMESPACE=${CURRENT_CERTIFICATES[$i]}
    let i++
    echo $NAME
    echo $NAMESPACE
    echo "---"
    oc label certificates $NAME -n $NAMESPACE foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
done

# Get all secrets with label operator.ibm.com/watched-by-cert-manager="" and add foundationservices.cloudpak.ibm.com=cert-manager
CURRENT_SECRETS=($(oc get secrets -l operator.ibm.com/watched-by-cert-manager="" --all-namespaces -o custom-columns=NAME:.metadata.name,NAMESPACE:metadata.namespace --no-headers=True))
i=0
len=${#CURRENT_SECRETS[@]}
while [ $i -lt $len ];
do
    NAME=${CURRENT_SECRETS[$i]}
    let i++
    NAMESPACE=${CURRENT_SECRETS[$i]}
    let i++
    echo $NAME
    echo $NAMESPACE
    echo "---"
    oc label secret $NAME -n $NAMESPACE foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
done

CURRENT_CRD_ISSUERS=($(oc get crd | grep issuer | cut -d ' ' -f1))
i=0
len=${#CURRENT_CRD_ISSUERS[@]}
while [ $i -lt $len ];
do
    NAME=${CURRENT_CRD_ISSUERS[$i]}
    let i++
    echo $NAME
    echo "---"
    oc label crd $NAME foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
done

CURRENT_CRD_CERTIFICATES=($(oc get crd | grep certificates | cut -d ' ' -f1))
i=0
len=${#CURRENT_CRD_CERTIFICATES[@]}
while [ $i -lt $len ];
do
    NAME=${CURRENT_CRD_CERTIFICATES[$i]}
    let i++
    echo $NAME
    echo "---"
    oc label crd $NAME foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
done

#ensure zenservice custom route secrets are labeled
zen_namespace_list=$(oc get zenservice -A | awk '{if (NR!=1) {print $1}}' || echo "fail")
if [[ $zen_namespace_list != "fail" ]]; then 
    for zen_namespace in $zen_namespace_list
    do
        zenservice_list=$(oc get zenservice -n $zen_namespace | awk '{if (NR!=1) {print $1}}')
        for zenservice in $zenservice_list
        do
            zen_secret_name=$(oc get zenservice $zenservice -n $zen_namespace -o=jsonpath='{.spec.zenCustomRoute.route_secret}')
            echo $zen_secret_name
            echo $zen_namespace
            echo "---"
            oc label secret $zen_secret_name -n $zen_namespace foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
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
        echo $iam_secret_name
        echo $tenant_config_namespace
        echo "---"
        oc label secret $iam_secret_name -n $tenant_config_namespace foundationservices.cloudpak.ibm.com=cert-manager --overwrite=true
    done
else
    echo "[INFO] Configmap cs-onprem-tenant-config not found, skipping copying custom secrets..."
fi