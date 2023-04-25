# How to test this operator

Put the CSV annotations to the `deploy.yaml` annotation.

Build your own image, publish to your own docker hub, and put it into `deploy.yaml` file.

## 1.Simulate in common-service namespace

Target:

* Verify the `ibm-common-servcies` namespace can be created
* Verify the `ibm-common-service-operator-operatorgroup` operatorgroup can be created in `ibm-common-servcies`
* Verify the `ibm-common-service-operator` subscription can be created in `ibm-common-servcies`

How:

```
oc create ns common-service

oc -n common-service apply -f deploy/sa.yaml
oc -n common-service apply -f deploy/role.yaml
oc -n common-service apply -f deploy/role_binding.yaml
oc -n common-service apply -f deploy/deploy.yaml
```

Destroy:

```
oc delete ns common-service ibm-common-services
```

## 2.Simulate in ibm-common-services namespace

Destroy the step 1 generated resources.

Target:

* Verify the ODLM Operator can be installed in `ibm-common-services` namespace
* Verify the Secretshare Operator can be installed in `ibm-common-services` namespace
* Verify the Webhook Operator can be installed in `ibm-common-services` namespace
* Verify the Namespace Scope Operator can be installed in `ibm-common-services` namespace
* Verify above operators and operands can work well
* Verify the `CommonService`, `OperandConfig`, `OperandRegistry` can be created in `ibm-common-services` namespace
* Verify the RBAC can be created in `kube-public` namespace

How:

```
oc create ns ibm-common-services

oc -n ibm-common-services apply -f deploy/sa.yaml
oc -n ibm-common-services apply -f deploy/role.yaml
oc -n ibm-common-services apply -f deploy/role_binding.yaml
oc -n ibm-common-services apply -f deploy/deploy.yaml
```

Destroy:

```
oc delete ns ibm-common-services
```

## 3.Simulate in openshift-operators namespace

Destroy the step 1&2 generated resources.

Target:

* Verify the ODLM Operator can be installed in `ibm-common-services` namespace
* Verify the Secretshare Operator can be installed in `ibm-common-services` namespace
* Verify the Webhook Operator can be installed in `ibm-common-services` namespace
* Verify the Namespace Scope Operator can be installed in `ibm-common-services` namespace
* Verify above operators and operands can work well
* Verify the `CommonService`, `OperandConfig`, `OperandRegistry` can be created in `ibm-common-services` namespace
* Verify the RBAC can be created in `kube-public` namespace
* Verify the Cluster RBAC can be created for Namespace Scope Operator service account

How:

```
oc -n openshift-operators apply -f deploy/sa.yaml
oc -n openshift-operators apply -f deploy/cluster_rbac.yaml
oc -n openshift-operators apply -f deploy/deploy.yaml
```

Destroy:

```
oc -n openshift-operators delete -f deploy/sa.yaml
oc -n openshift-operators delete -f deploy/cluster_rbac.yaml
oc -n openshift-operators delete -f deploy/deploy.yaml

oc delete ns ibm-common-services
```

## 4.Test sizing update

Put the CSV annotations to the `deploy.yaml` annotation.

Build your own image, publish to your own docker hub, and put it into `deploy.yaml` file
### shirt size update

Deploy cs operator in the `ibm-common-services` namespace

```
oc create ns ibm-common-services

oc -n ibm-common-services apply -f deploy/sa.yaml
oc -n ibm-common-services apply -f deploy/role.yaml
oc -n ibm-common-services apply -f deploy/role_binding.yaml
oc -n ibm-common-services apply -f deploy/deploy.yaml
```

```
oc create ns ibm-cloud-pak

oc -n ibm-cloud-pak apply -f sizing/small_size.yaml
```

* Verify OperandConfig doesn't update. It still uses the medium profile.

```
oc -n ibm-cloud-pak apply -f sizing/large_size.yaml
```

* Verify OperandConfig is update to the large profile.

```
oc -n ibm-cloud-pak delete -f sizing/large_size.yaml
```

* Verify OperandConfig is using the medium profile.

### resource size update

Apply the CR with large mongodb sizing
```
oc -n ibm-cloud-pak apply -f sizing/large_mongodb.yaml
```

* Verify OperandConfig. the mongodb should have a larger size.

```
oc -n ibm-cloud-pak delete -f sizing/large_mongodb.yaml
```

* Verify OperandConfig. the size of mongodb is reverted.

### StorageClass update

Apply storage class into the CR, replace the storage class to the existing storage class in the cluster.
```
oc -n ibm-cloud-pak apply -f sizing/storage_class.yaml
```

* Verify OperandConfig has the storage class setting for the mongodb

Delete the CR, storage class setting is reverted from the OperandConfig.

```
oc -n ibm-cloud-pak delete -f sizing/storage_class.yaml
```

* Verify OperandConfig. the storage class of mongodb is reverted.

Create the OperandRequest for the mongodb

```
oc -n ibm-common-services apply -f sizing/operandrequest.yaml
```

Then apply the storageclass settings

```
oc -n ibm-cloud-pak apply -f sizing/storage_class.yaml
```

* Verify the storage class of mongodb is not added in the OperandConfig.
