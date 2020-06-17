<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Install IBM Common Services with CLI](#install-ibm-common-services-with-cli)
  - [1. Create OperatorSource](#1-create-operatorsource)
  - [2. Create Operator NS, Group, Subscription](#2-create-operator-ns-group-subscription)
  - [3. Waiting for Operator CSV is ready](#3-waiting-for-operator-csv-is-ready)
  - [4. Create OperandRequest instance](#4-create-operandrequest-instance)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Install IBM Common Services with CLI

## 1. Create OperatorSource

The OperatorSource is used to define the external data store used to store Operator bundles.

But if you want to install a development version of IBM Common Services, then you need to create following OperatorSource.

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: opencloud-operators
  namespace: openshift-marketplace
spec:
  displayName: IBMCS Operators
  publisher: IBM
  sourceType: grpc
  image: quay.io/opencloudio/ibm-common-service-catalog:dev-latest
  updateStrategy:
    registryPoll:
      interval: 45m
```

## 2. Create Namespace, OperatorGroup and Subscription

**Note:** For CloudPak users, you need to replace the namespace `common-service` to the namespace of the CloudPak.
**Note:** The `dev` channel in the subscription is for testing purposes. For the product, we need to use the `stable` channel.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: common-service

---
apiVersion: operators.coreos.com/v1alpha2
kind: OperatorGroup
metadata:
  name: operatorgroup
  namespace: common-service
spec:
  targetNamespaces:
  - common-service

---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ibm-common-service-operator
  namespace: common-service
spec:
  channel: dev # dev channel is for development purpose only
  installPlanApproval: Automatic
  name: ibm-common-service-operator
  source: opencloud-operators
  sourceNamespace: openshift-marketplace
```

The ibm common service operator supports the all namespaces mode. If the CloudPak is deployed in the namespace `openshift-operators` in the all namespaces mode, then the ibm common service operator need to be installed in the namespace `openshift-operators` as well.

```yaml

apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ibm-common-service-operator
  namespace: openshift-operators
spec:
  channel: dev # dev channel is for development purpose only
  installPlanApproval: Automatic
  name: ibm-common-service-operator
  source: opencloud-operators
  sourceNamespace: openshift-marketplace
```

## 3. Waiting for Operator CSV is ready

Check if ibm-common-service-operator and operands-deployment-lifecycle-manager are ready.

```bash
# Check ibm-common-service-operator
oc -n ibm-common-services get csv
# Check operands-deployment-lifecycle-manager
oc -n openshift-operators get csv
```

Check if the CRDs are created

```bash
oc get crd operandrequest
```

## 4. Create OperandRequest instance

Add the required services into `operands:` and create the `OperandRequest` in the namespace you create.

```yaml
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRequest
metadata:
  name: common-service
spec:
  requests:
  - registry: common-service
    registryNamespace: ibm-common-services
    operands:
    - name: ibm-licensing-operator
```

The list of operators you can add:

```bash
    - name: ibm-mongodb-operator
    - name: ibm-iam-operator
    - name: ibm-monitoring-exporters-operator
    - name: ibm-monitoring-prometheusext-operator
    - name: ibm-healthcheck-operator
    - name: ibm-management-ingress-operator
    - name: ibm-ingress-nginx-operator
    - name: ibm-metering-operator
    - name: ibm-licensing-operator
    - name: ibm-commonui-operator
    - name: ibm-auditlogging-operator
    - name: ibm-catalog-ui-operator
    - name: ibm-platform-api-operator
```

**Note:** `ibm-cert-manager`, `ibm-helm-api-operator` and `ibm-helm-repo-operator` are private operator for common services and only be requested within `ibm-common-services` namespace.

After the `OperandRequest` created, we can check if common services are installed successfully.

```bash
oc -n ibm-common-services get csv
```
