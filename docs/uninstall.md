<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Uninstall IBM Common Services](#uninstall-ibm-common-services)
  - [Delete OperandRequest Created by CloudPaks](#delete-operandrequest-created-by-cloudpaks)
  - [Clean up environment (Optional)](#clean-up-environment-optional)
    - [Clean up OperandConfig and OperandRegistry](#clean-up-operandconfig-and-operandregistry)
    - [Clean up `IBM Common Service Operator` and `Operand Deployment Lifecycle Manager`](#clean-up-ibm-common-service-operator-and-operand-deployment-lifecycle-manager)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Uninstall IBM Common Services

## Delete OperandRequest Created by CloudPaks

Users can use the following steps to remove the services deployed by `IBM Common Service Operator`.

1. Log in to your OpenShift cluster console.

2. Access the `OperandRequest` API. Click Operators > Installed Operators and switch to the namespace you create `OperandRequest`.

3. Click `Operand Deployment Lifecycle Manager` > `OperandRequest` and delete the OperandRequest instances you create. When an OperandRequest instance is deleted, the services that are requested by this OperandRequest instance are deleted unless the service is requested by other OperandRequest instances.

## Clean up environment (Optional)

**Note:** The following steps is for cleaning up `IBM Common Service Operator` and `Operand Deployment Lifecycle Manager`. You can only clean up them when you make sure, no other CloudPaks are using them.

### Clean up OperandConfig and OperandRegistry

Delete the OperandConfig API.

Click `Operators` > `Installed Operators` and switch to namespace `ibm-common-services`

Click `Operand Deployment Lifecycle Manager` > `OperandConfig` and delete all the OperandConfig instances.

Then Click `OperandRegistry` and delete all the `OperandRegistry` instances.

### Clean up `IBM Common Service Operator` and `Operand Deployment Lifecycle Manager`

- Uninstall the `Operand Deployment Lifecycle Manager`.

    Click Operators > Installed Operators and switch to namespace `openshift-operators`.

    Click the overflow menu icon and click Uninstall Operator.

- Uninstall the `IBM Common Service Operator` operator.

    Click Operators > Installed Operators and switch to namespace that CloudPaks deployed `IBM Common Service Operator`.

    Click the overflow menu icon and click Uninstall Operator.

- Remove `ibm-common-services` namespace.

    Click Projects and delete project `ibm-common-services`.
