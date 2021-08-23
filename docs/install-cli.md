<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Install IBM Common Services with CLI](#install-ibm-common-services-with-cli)
  - [1. Create CatalogSource](#1-create-catalogsource)
  - [2. Create Namespace, OperatorGroup and Subscription](#2-create-namespace-operatorgroup-and-subscription)
  - [3. Waiting for Operator CSV is ready](#3-waiting-for-operator-csv-is-ready)
  - [4.Configure IBM Common Services](#4configure-ibm-common-services)
    - [Configure Size](#configure-size)
    - [Configure general parameters](#configure-general-parameters)
  - [5. Create OperandRequest instance](#5-create-operandrequest-instance)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Install IBM Common Services with CLI

## 1. Create CatalogSource

The OLM uses CatalogSources, which use the Operator Registry API, to query for available Operators as well as upgrades for installed Operators.

You need to create the CatalogSource as a prerequisite for the IBM common services installation.

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
  image: quay.io/opencloudio/ibm-common-service-catalog:latest
  updateStrategy:
    registryPoll:
      interval: 45m
```

## 2. Create Namespace, OperatorGroup and Subscription

**Note:** For CloudPak users, you need to replace the namespace `common-service` to the namespace of the CloudPak.
**Note:** You can use the `stable-v1` channel for installing the common service in the last release or use the `beta` channel to install the latest beta release.

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
  channel: beta
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
  channel: beta
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

## 4.Configure IBM Common Services

### Configure Size

The IBM Common Service Operator will bootstrap a `commonservice` CR in the `ibm-common-services` namespace. The initialized size is `starterset`.

```yaml
apiVersion: operator.ibm.com/v3
kind: CommonService
metadata:
  name: common-service
  namespace: ibm-common-services
spec:
  size: starterset
```

The supported sizes are: `starterset`, `small`, `medium` and `large`.
**NOTE** In the post installation, once the size is changed from `starterset` to other size profile, we could not roll back to `starterset` anymore. We are able to switch between `small`, `medium` and `large`.

### Configure general parameters

Take MongoDB as an example, following is configure MongoDB storage class:

```yaml
apiVersion: operator.ibm.com/v3
kind: CommonService
metadata:
  name: common-service
  namespace: ibm-common-services
spec:
  size: starterset
  services:
  - name: ibm-mongodb-operator
    spec:
      mongoDB:
        storageClass: cephfs
```

## 5. Create OperandRequest instance

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
    - name: ibm-licensing-operator
    - name: ibm-commonui-operator
    - name: ibm-auditlogging-operator
    - name: ibm-platform-api-operator
```

**Note:** `ibm-cert-manager` is a private operator for common services and can only be requested within `ibm-common-services` namespace.

After the `OperandRequest` created, we can check if common services are installed successfully.

```bash
oc -n ibm-common-services get csv
```
