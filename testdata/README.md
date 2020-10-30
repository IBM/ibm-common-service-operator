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

oc -n common-service apply -f sa.yaml
oc -n common-service apply -f role.yaml
oc -n common-service apply -f role_binding.yaml
oc -n common-service apply -f deploy.yaml
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
* Verify the IAM status configmap can be created in `kube-public` namespace
* Verify the RBAC can be created in `kube-public` namespace

How:

```
oc create ns ibm-common-services

oc -n ibm-common-services apply -f sa.yaml
oc -n ibm-common-services apply -f role.yaml
oc -n ibm-common-services apply -f role_binding.yaml
oc -n ibm-common-services apply -f deploy.yaml
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
* Verify the IAM status configmap can be created in `kube-public` namespace
* Verify the RBAC can be created in `kube-public` namespace
* Verify the Cluster RBAC can be created for Namespace Scope Operator service account

How:

```
oc -n openshift-operators apply -f sa.yaml
oc -n openshift-operators apply -f cluster_rbac.yaml
oc -n openshift-operators apply -f deploy.yaml
```

Destroy:

```
oc -n openshift-operators delete -f sa.yaml
oc -n openshift-operators delete -f cluster_rbac.yaml
oc -n openshift-operators delete -f deploy.yaml

oc delete ns ibm-common-services
```
