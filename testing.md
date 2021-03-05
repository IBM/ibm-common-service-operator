# How to test changes
This document assumes you will be using personal quay.io registry to push bundle and operator images to avoid accidentally changing production registries
```
export QUAY_REGISTRY=quay.io/<your_namespace>
```

Also create a CatalogSource
```
cat <<EOF | tee >(oc apply -f -) | cat
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: test-cs-operator
  namespace: openshift-marketplace
spec:
  displayName: test-cs-operator
  publisher: IBM
  sourceType: grpc
  image: quay.io/opencloudio/ibm-common-service-catalog:3.7.0-beta
  updateStrategy:
    registryPoll:
      interval: 45m
EOF
```

If pushing an image to your quay.io registry with its first tag, the image repository may be set to private by default, so you will need to login to quay.io and change the repository settings to public in order for your cluster to pull the image.

The actual testing consist of:
1. verify common-service operator pod is Running
2. verify namespace scope operator and ODLM operator are installed
3. verify secretshare and common-service-webhook pods are running
4. verify functionality of common-service operator such as:
    - size profile

## Test fresh installation with bundle
1. add features/fixes to repo
2. change `image` value in config/manager/manager.yaml to `quay.io/<your_namespace>/common-service-operator:dev`
   - and any of the operand image values if necessary
3. build operator with changes
```
make build-dev
```
4. build bundle containing changes and bundle image
```
make bundle-manifests VERSION=99.99.99
make build-bundle-image VERSION=dev
```
5. deploy operator using bundle format
```
make run-bundle VERSION=dev
```
6. run tests
7. clean up
```
make cleanup-bundle
```

## Test upgrade with bundle
Similar to fresh installation test except you will first deploy the operator/bundle of the most recent release without any changes, i.e. the operator/bundle code from the most recent commit in master branch

1. change `image` value in config/manager/manager.yaml to `quay.io/<your_namespace>/common-service-operator:3.7.1`
2. build bundle and bundle image without any changes
```
make bundle
make build-bundle-image
```
3. deploy unchanged operator using bundle format
```
make run-bundle
```
4. add features/fixes to repo
5. change `image` value in config/manager/manager.yaml to `quay.io/<your_namespace>/common-service-operator:dev`
   - and any of the operand image values if necessary
6. build operator with changes
```
make build-dev
```
7. build bundle containing changes and bundle image
```
make bundle-manifests VERSION=99.99.99
make build-bundle-image VERSION=dev
```
8. upgrade operator
```
make upgrade-bundle
```
9. run tests
10. clean up
```
make cleanup-bundle
```
