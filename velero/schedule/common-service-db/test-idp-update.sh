#!/bin/bash

# Test script to check IDP update functionality
CSDB_NAMESPACE=$1

if [[ -z $CSDB_NAMESPACE ]]; then
    echo "No namespace provided. please provide the namespace as an argument."
    exit 1
fi

echo "Getting primary PostgreSQL pod..."
CNPG_PRIMARY_POD=$(oc get cluster.postgresql.k8s.enterprisedb.io common-service-db -o jsonpath="{.status.currentPrimary}" -n $CSDB_NAMESPACE)

if [[ -z $CNPG_PRIMARY_POD ]]; then
    echo "Error: Could not find primary PostgreSQL pod"
    exit 1
fi

echo "Primary pod: $CNPG_PRIMARY_POD"

echo "Checking if account_iam database exists..."
ACCOUNT_IAM_EXISTS=$(oc -n $CSDB_NAMESPACE exec -t $CNPG_PRIMARY_POD -c postgres -- psql -U postgres -c "\list" | grep "account_iam" || echo False)

if [[ $ACCOUNT_IAM_EXISTS == "False" ]]; then
    echo "account_iam database not found. Creating test data..."
    exit 1
fi

echo "Current IDP configuration BEFORE update:"
oc -n $CSDB_NAMESPACE exec -t $CNPG_PRIMARY_POD -c postgres -- psql -U postgres -d account_iam -c "
    SELECT uid, realm, idp, modified_ts 
    FROM accountiam.idp_config 
    ORDER BY modified_ts DESC;
"

echo "Getting cluster domain from ibmcloud-cluster-info configmap..."
CLUSTER_DOMAIN=$(oc get cm ibmcloud-cluster-info -n $CSDB_NAMESPACE -o jsonpath='{.data.cluster_address}' 2>/dev/null || echo "")

if [[ -z $CLUSTER_DOMAIN ]]; then
    echo "Error: Could not determine cluster domain from ibmcloud-cluster-info configmap."
    echo "Please ensure the ibmcloud-cluster-info configmap exists in namespace $CSDB_NAMESPACE"
    exit 1
fi

echo "Detected cluster domain: $CLUSTER_DOMAIN"

NEW_IDP_URL="https://${CLUSTER_DOMAIN}/idprovider/v1/auth"
echo "Target IDP URL: $NEW_IDP_URL"

# Check current IDP configuration first
echo "Checking current IDP configuration..."
CURRENT_IDP=$(oc -n $CSDB_NAMESPACE exec -t $CNPG_PRIMARY_POD -c postgres -- psql -U postgres -d account_iam -t -c "SELECT DISTINCT idp FROM accountiam.idp_config WHERE idp LIKE '%/idprovider/v1/%' LIMIT 1;" | xargs || echo "")

if [[ -n $CURRENT_IDP ]] && [[ $CURRENT_IDP != $NEW_IDP_URL ]]; then
    echo "Current IDP URL: $CURRENT_IDP"
    echo "Updating IDP configuration..."
    
    # Perform the update
    oc -n $CSDB_NAMESPACE exec -t $CNPG_PRIMARY_POD -c postgres -- psql -U postgres -d account_iam -c "
        UPDATE accountiam.idp_config 
        SET idp = '$NEW_IDP_URL', 
            modified_ts = NOW()
        WHERE idp LIKE '%/idprovider/v1/%';
    "
elif [[ $CURRENT_IDP == $NEW_IDP_URL ]]; then
    echo "IDP configuration already matches target URL, no update needed."
    echo "Current IDP URL: $CURRENT_IDP"
else
    echo "No IDP configuration found in database, skipping update."
fi

echo "IDP configuration AFTER update:"
oc -n $CSDB_NAMESPACE exec -t $CNPG_PRIMARY_POD -c postgres -- psql -U postgres -d account_iam -c "
    SELECT uid, realm, idp, modified_ts 
    FROM accountiam.idp_config 
    ORDER BY modified_ts DESC;
"

echo "Test completed!"
