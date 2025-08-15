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

echo "Getting cluster domain..."
CLUSTER_DOMAIN=$(oc get route console -n openshift-console -o jsonpath='{.spec.host}' | sed 's/^console-openshift-console\.//')

if [[ -z $CLUSTER_DOMAIN ]]; then
    echo "Warning: Could not determine cluster domain from console route, trying alternative method..."
    CLUSTER_DOMAIN=$(oc get ingress.config.openshift.io cluster -o jsonpath='{.spec.domain}')
fi

if [[ -z $CLUSTER_DOMAIN ]]; then
    echo "Error: Could not determine cluster domain. Using example.com for testing."
    CLUSTER_DOMAIN="example.com"
fi

echo "Detected cluster domain: $CLUSTER_DOMAIN"

NEW_IDP_URL="https://cp-console.${CSDB_NAMESPACE}.${CLUSTER_DOMAIN}/idprovider/v1/auth"
echo "New IDP URL will be: $NEW_IDP_URL"

# Perform the update
echo "Updating IDP configuration..."
oc -n $CSDB_NAMESPACE exec -t $CNPG_PRIMARY_POD -c postgres -- psql -U postgres -d account_iam -c "
    UPDATE accountiam.idp_config 
    SET idp = '$NEW_IDP_URL', 
        modified_ts = NOW()
    WHERE idp LIKE '%/idprovider/v1/%';
"

echo "IDP configuration AFTER update:"
oc -n $CSDB_NAMESPACE exec -t $CNPG_PRIMARY_POD -c postgres -- psql -U postgres -d account_iam -c "
    SELECT uid, realm, idp, modified_ts 
    FROM accountiam.idp_config 
    ORDER BY modified_ts DESC;
"

echo "Test completed!"
