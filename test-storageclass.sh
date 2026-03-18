#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="cs5"
CS_NAME="common-service"
DEPLOYMENT_NAME="ibm-common-service-operator"
STORAGE_CLASS="nfs-storage"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}StorageClass Configuration Test Script${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Step 1: Remove storageClass from CommonService CR
echo -e "${YELLOW}[Step 1/8]${NC} Removing storageClass from CommonService CR..."
# Use kubectl patch with JSON patch to remove the field
oc patch commonservice ${CS_NAME} -n ${NAMESPACE} --type=json -p='[{"op": "remove", "path": "/spec/storageClass"}]' 2>/dev/null || echo "storageClass field may not exist, continuing..."

# Verify it's removed
STORAGE_CLASS_VALUE=$(oc get commonservice ${CS_NAME} -n ${NAMESPACE} -o jsonpath='{.spec.storageClass}' 2>/dev/null || echo "")
if [ -z "$STORAGE_CLASS_VALUE" ]; then
    echo -e "${GREEN}✓ StorageClass removed from CommonService CR${NC}"
else
    echo -e "${RED}✗ Failed to remove storageClass, current value: ${STORAGE_CLASS_VALUE}${NC}"
    echo "Trying alternative method..."
    oc get commonservice ${CS_NAME} -n ${NAMESPACE} -o json | \
      jq 'del(.spec.storageClass)' | \
      oc replace -f -
    echo -e "${GREEN}✓ StorageClass removed using replace${NC}"
fi
echo ""

# Step 2: Build dev image
echo -e "${YELLOW}[Step 2/8]${NC} Building dev image..."
cd /root/project/ibm-common-service-operator
make build-dev-image
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Image built successfully${NC}"
else
    echo -e "${RED}✗ Image build failed${NC}"
    exit 1
fi
echo ""

# Step 3: Wait for build to complete (already done in step 2)
echo -e "${YELLOW}[Step 3/8]${NC} Build completed"
echo ""

# Step 4: Restart operator pod
echo -e "${YELLOW}[Step 4/8]${NC} Restarting operator pod..."
oc delete pod -n ${NAMESPACE} -l name=${DEPLOYMENT_NAME}
echo "Waiting for new pod to be ready..."
oc wait --for=condition=ready pod -n ${NAMESPACE} -l name=${DEPLOYMENT_NAME} --timeout=300s
echo -e "${GREEN}✓ Operator pod restarted${NC}"
echo ""

# Step 5: Wait for first reconcile and CS ready
echo -e "${YELLOW}[Step 5/8]${NC} Waiting for first reconcile to complete..."
sleep 10
for i in {1..30}; do
    STATUS=$(oc get commonservice ${CS_NAME} -n ${NAMESPACE} -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    if [ "$STATUS" == "Succeeded" ]; then
        echo -e "${GREEN}✓ CommonService is ready (status: $STATUS)${NC}"
        break
    fi
    echo "Waiting for CommonService to be ready... (attempt $i/30, current status: $STATUS)"
    sleep 10
done
echo ""

sleep 60

# Step 6: Add storageClass to CommonService CR
echo -e "${YELLOW}[Step 6/8]${NC} Adding storageClass to CommonService CR..."
oc patch commonservice ${CS_NAME} -n ${NAMESPACE} --type=merge -p "{\"spec\":{\"storageClass\":\"${STORAGE_CLASS}\"}}"
echo -e "${GREEN}✓ StorageClass added: ${STORAGE_CLASS}${NC}"
echo ""

# Step 7: Wait for CS ready after update
echo -e "${YELLOW}[Step 7/8]${NC} Waiting for reconcile after storageClass update..."
sleep 15
for i in {1..30}; do
    STATUS=$(oc get commonservice ${CS_NAME} -n ${NAMESPACE} -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    if [ "$STATUS" == "Succeeded" ]; then
        echo -e "${GREEN}✓ CommonService is ready after update (status: $STATUS)${NC}"
        break
    fi
    echo "Waiting for CommonService to be ready... (attempt $i/30, current status: $STATUS)"
    sleep 10
done
echo ""

# Step 8: Verify OperandConfig contains storageClass
echo -e "${YELLOW}[Step 8/8]${NC} Verifying OperandConfig contains storageClass..."
echo ""

# Check edb-keycloak service
echo "Checking edb-keycloak Cluster resource..."
STORAGE_JSON=$(oc get operandconfig common-service -n ${NAMESPACE} -o jsonpath='{.spec.services[?(@.name=="edb-keycloak")].resources[0].data.spec.storage}' 2>/dev/null || echo "{}")
echo "Storage config: $STORAGE_JSON"

if echo "$STORAGE_JSON" | grep -q "storageClass"; then
    STORAGE_CLASS_VALUE=$(echo "$STORAGE_JSON" | jq -r '.storageClass' 2>/dev/null || echo "")
    if [ "$STORAGE_CLASS_VALUE" == "${STORAGE_CLASS}" ]; then
        echo -e "${GREEN}✓ storage.storageClass found: ${STORAGE_CLASS_VALUE}${NC}"
    else
        echo -e "${RED}✗ storage.storageClass has wrong value: ${STORAGE_CLASS_VALUE} (expected: ${STORAGE_CLASS})${NC}"
    fi
else
    echo -e "${RED}✗ storage.storageClass NOT found in OperandConfig!${NC}"
fi

WAL_STORAGE_JSON=$(oc get operandconfig common-service -n ${NAMESPACE} -o jsonpath='{.spec.services[?(@.name=="edb-keycloak")].resources[0].data.spec.walStorage}' 2>/dev/null || echo "{}")
echo "WAL Storage config: $WAL_STORAGE_JSON"

if echo "$WAL_STORAGE_JSON" | grep -q "storageClass"; then
    WAL_STORAGE_CLASS_VALUE=$(echo "$WAL_STORAGE_JSON" | jq -r '.storageClass' 2>/dev/null || echo "")
    if [ "$WAL_STORAGE_CLASS_VALUE" == "${STORAGE_CLASS}" ]; then
        echo -e "${GREEN}✓ walStorage.storageClass found: ${WAL_STORAGE_CLASS_VALUE}${NC}"
    else
        echo -e "${RED}✗ walStorage.storageClass has wrong value: ${WAL_STORAGE_CLASS_VALUE} (expected: ${STORAGE_CLASS})${NC}"
    fi
else
    echo -e "${RED}✗ walStorage.storageClass NOT found in OperandConfig!${NC}"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Checking operator logs for debug info...${NC}"
echo -e "${BLUE}========================================${NC}"

echo ""
echo -e "${YELLOW}=== Merge and YAML Generation Logs ===${NC}"
oc logs -n ${NAMESPACE} deployment/${DEPLOYMENT_NAME} --tail=200 | grep -E "(mergeResourceArrays|mergedResource storage|Final YAML|renderTemplate)" | tail -20

echo ""
echo -e "${YELLOW}=== Create/Update Operation Logs ===${NC}"
oc logs -n ${NAMESPACE} deployment/${DEPLOYMENT_NAME} --tail=200 | grep -E "(Creating resource.*OperandConfig|Updating resource.*OperandConfig|🟢|🟠|🔵|⚠️|CreateOrUpdateFromYaml)" | tail -20

echo ""
echo -e "${YELLOW}=== StorageClass Related Logs ===${NC}"
oc logs -n ${NAMESPACE} deployment/${DEPLOYMENT_NAME} --tail=200 | grep -i "storageclass" | tail -20

echo ""
echo -e "${YELLOW}=== Success/Failure Indicators ===${NC}"
oc logs -n ${NAMESPACE} deployment/${DEPLOYMENT_NAME} --tail=200 | grep -E "(✅|❌)" | tail -10

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Full Debug Log (last 300 lines)${NC}"
echo -e "${BLUE}========================================${NC}"
oc logs -n ${NAMESPACE} deployment/${DEPLOYMENT_NAME} --tail=300 > /tmp/operator-full-log.txt
echo "Full log saved to /tmp/operator-full-log.txt"
echo "You can view it with: cat /tmp/operator-full-log.txt"

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Complete${NC}"
echo -e "${BLUE}========================================${NC}"

# Made with Bob
