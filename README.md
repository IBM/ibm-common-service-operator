# ibm-common-service-operator

The `ibm-common-service-operator` is a bridge to connect IBM Cloud Paks and Operand Deployment Lifecycle Manager (ODLM) with IBM Cloud Platform Common Services. You can also install the `ibm-common-service-operator` in stand-alone mode.

When you install this operator, the operator completes the following tasks:

- Installs ODLM in all namespaces mode
- Creates the `ibm-common-services` namespace
- Creates the Common Services `OperandRegistry` and `OperandConfig` in the `ibm-common-services` namespace

For more information about installing this operator and other Common Services operators, see [Installer documentation](http://ibm.biz/cpcs_opinstall). If you are using this operator as part of an IBM Cloud Pak, see the documentation for that IBM Cloud Pak to learn more about how to install and use the operator service. For more information about IBM Cloud Paks, see [IBM Cloud Paks that use Common Services](http://ibm.biz/cpcs_cloudpaks).

For more information about the available IBM Cloud Platform Common Services, see the [IBM Knowledge Center](http://ibm.biz/cpcsdocs).

## Supported platforms

Red Hat OpenShift Container Platform 4.3 or newer installed on one of the following platforms:

   - Linux x86_64
   - Linux on Power (ppc64le)

## Operator versions

 - 4.3.0

## Prerequisites

Before you install this operator, you need to first install the operator prerequisites:

- For the list of prerequisites for installing the operator, see the IBM Knowledge Center [Preparing to install services documentation](http://ibm.biz/cpcs_opinstprereq).

## Documentation

- If you are using the operator as part of an IBM Cloud Pak, see the documentation for that IBM Cloud Pak. For a list of IBM Cloud Paks, see [IBM Cloud Paks that use Common Services](http://ibm.biz/cpcs_cloudpaks).
- If you are using the operator in stand-alone mode or with an IBM Containerized Software, see the IBM Cloud Platform Common Services Knowledge Center [Installer documentation](http://ibm.biz/cpcs_opinstall).

## SecurityContextConstraints Requirements

The IBM Common Service Operator supports running with the OpenShift Container Platform 4.3 default restricted Security Context Constraints (SCCs).

For more information about the OpenShift Container Platform Security Context Constraints, see [Managing Security Context Constraints](https://docs.openshift.com/container-platform/4.3/authentication/managing-security-context-constraints.html).

## Developer guide

If, as a developer, you are looking to build and test this operator to try out and learn more about the operator and its capabilities, you can use the following developer guide. This guide provides commands for a quick install and initial validation for running the operator.

  - [IBM Common Service Operator Design](docs/design.md)
  - [Install Common Services in stand-alone mode](docs/install.md)
  - [Install Common Services with CloudPaks](docs/cloudpak-integration.md)

### End-to-End testing

For more instructions on how to run end-to-end testing with the Operand Deployment Lifecycle Manager, see [ODLM guide](https://github.com/IBM/operand-deployment-lifecycle-manager/blob/master/docs/install/common-service-integration.md#end-to-end-test).

