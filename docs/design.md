# Design

IBM Common Service Operator is designed as the entrance of the IBM Common Services.

There are multiple individual servies in IBM Common Services, we need a method to unify the installation of these individual services.

A single cluster can have multiple IBM Common Service Operator installed, and all these operators have same function.

This Operator is a minimal operator that only have a single API and no controller, it will create/update the IBM Common Services related Kubernetes resources during the start of Operator process.

It will:

* create the `ibm-common-services` namespace, all the individual services will be installed into this namespace
* create the `operand-deployment-lifecycle-manager` subcription which will trigger the ODLM to be installed in all namespaces mode
* create the `OperandConfig` and `OperandRegistry` of IBM Common Services under `ibm-common-services` namespace which tells the individual service configuration and location, later users can manually create `OperandRequest` to trigger the installation
* create extra Kubernetes resources, there is a `build/resources/extra` directory, putting Kuberentes resource YAML files to this directory, the operator will automatically create them(note: one YAML file should only contain one Kubernetes resource)

The main logic is inside [pkg/bootstrap/init.go](/pkg/bootstrap/init.go) file.
