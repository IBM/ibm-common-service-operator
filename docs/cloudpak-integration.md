- [CloudPak Integration](#cloudpak-integration)
  * [1.Create OperatorSource](#1create-operatorsource)
  * [2.Make CloudPak depends on IBM Common Service Operator](#2make-cloudpak-depends-on-ibm-common-service-operator)
  * [3.Install Individual Common Services](#3install-individual-common-services)


# CloudPak Integration

Install IBM Common Services along with CloudPaks.

IBM Common Services is invisible to CloudPak users, when CloudPak users install the CloudPak, IBM Common Services will be seamlessly installed.


## 1.Create OperatorSource

The OperatorSource is used to define the external data store used to store Operator bundles.

By default, OpenShift has build-in three OperatorSources and all the released IBM Common Services operators are published to one of the build-in OperatorSources, so if you want to install a released version of IBM Common Services, you don't need to create the OperatorSource.

But if you want to install a development version of IBM Common Services, then you need to create following OperatorSource.

```yaml
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: opencloud-operators
  namespace: openshift-marketplace
spec:
  authorizationToken: {}
  displayName: IBMCS Operators
  endpoint: https://quay.io/cnr
  publisher: IBM
  registryNamespace: opencloudio
  type: appregistry
```

Open the OpenShift Web Console, click the plus button in top right corner, and then copy the above operator source into the editor.

![Create OperatorSource](./images/create-operator-source.png)


## 2.Make CloudPak depends on IBM Common Service Operator

In CloudPak Operator CSV file, add following content.

This can ensure when users install CloudPak Operator, the IBM Common Service Operator will be also installed by OLM.

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: cloudpak-operator.v0.0.1
  namespace: placeholder
spec:
  customresourcedefinitions:
    required:
    - description: CommonService is the Schema for the commonservices API
      displayName: CommonService
      kind: CommonService
      name: commonservices.operator.ibm.com
      version: v3
```

The logics inside IBM Common Service Operator is during the start the operator, it will:

1. Install ODLM operator in all namespaces mode
1. Create `ibm-common-services` namespace
1. Create IBM Common Services `OperandConfig` and `OperandRegistry`


## 3.Install Individual Common Services

Install individual common services by creating an OperandRequest.

```yaml
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRequest
metadata:
  name: cloudpak-required-common-service
  namespace: placeholder
spec:
  requests:
    - operands:
      - name: ibm-cert-manager-operator
      - name: ibm-mongodb-operator
      - name: ibm-iam-operator
      - name: ibm-monitoring-exporters-operator
      - name: ibm-monitoring-prometheusext-operator
      - name: ibm-monitoring-grafana-operator
      - name: ibm-healthcheck-operator
      - name: ibm-management-ingress-operator
      - name: ibm-licensing-operator
      - name: ibm-metering-operator
      - name: ibm-commonui-operator
      - name: ibm-elastic-stack-operator
      - name: ibm-ingress-nginx-operator
      - name: ibm-auditlogging-operator
      - name: ibm-platform-api-operator
      - name: ibm-helm-api-operator
      - name: ibm-helm-repo-operator
      - name: ibm-catalog-ui-operator
      registry: common-service
      registryNamespace: ibm-common-services
```

For development and testing purposes, ibm common service operator will deploy two pairs of the `OperandRegistry` and `OperandConfig`.

**Note:** Uses need to add the required services and all their dependencies into `OperandRequest`.

CloudPaks can create this `OperandRequest` during [the CloudPak Operator start](https://github.com/IBM/ibm-common-service-operator/blob/master/cmd/manager/main.go#L121-L126), or have their own method to create this `OperandRequest`.

After created this `OperandRequest`, ODLM Operator will use it to trigger the individual common services installation.
