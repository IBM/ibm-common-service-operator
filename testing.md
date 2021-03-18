# How to test changes
The testing commands rely on having access to the following registries:
```
quay.io/opencloudio
hyc-cloud-private-scratch-docker-local.artifactory.swg-devops.com/ibmcom
```

If you do not have access or would prefer to use different registries then you can overwrite them with:
```
export QUAY_REGISTRY=<your_registry>
export REGISTRY=<your_registry>
```

The QUAY_REGISTRY value must point to a public registry because `operator-sdk run bundle` does not work with private registries currently (as of v1.5.0).

If pushing an image to your quay.io registry with its first tag, the image repository may be set to private by default, so you will need to login to quay.io and change the repository settings to public.

Also create a CatalogSource
```
cat <<EOF | tee >(oc apply -f -) | cat
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: opencloud-operators
  namespace: openshift-marketplace
spec:
  displayName: opencloud-operators
  publisher: IBM
  sourceType: grpc
  image: quay.io/opencloudio/ibm-common-service-catalog:3.7.1
  updateStrategy:
    registryPoll:
      interval: 45m
EOF
```

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
make build-dev-image
```
4. build bundle containing changes and bundle image
```
make bundle-manifests RELEASE_VERSION=99.99.99
make build-bundle-image RELEASE_VERSION=dev
```
5. deploy operator using bundle format
```
make run-bundle RELEASE_VERSION=dev
```
6. run tests
7. clean up
```
make cleanup-bundle
```

## Test upgrade with bundle
Similar to fresh installation test except you will first deploy the operator/bundle of the most recent release without any changes, i.e. the operator/bundle code from the most recent commit in master branch

1. deploy unchanged operator with official bundle image
```
make run-bundle BUNDLE_IMAGE_NAME=ibm-common-service-operator-bundle
```

If using your own registries, you can pull the official bundle image, and then push it to your registry
```
docker pull quay.io/opencloudio/ibm-common-service-operator-bundle:<RELEASE_VERSION>
docker tag quay.io/opencloudio/ibm-common-service-operator-bundle:<RELEASE_VERSION> <your_registry>/dev-common-service-operator-bundle:<RELEASE_VERSION>
docker push <your_registry>/dev-common-service-operator-bundle:<RELEASE_VERSION>
make run-bundle
```
2. add features/fixes to repo
3. change `image` value in config/manager/manager.yaml to `quay.io/<your_namespace>/common-service-operator:dev`
   - and any of the operand image values if necessary
4. build operator with changes
```
make build-dev-image
```
5. build bundle containing changes and bundle image
```
make bundle-manifests RELEASE_VERSION=99.99.99
make build-bundle-image RELEASE_VERSION=dev
```
6. upgrade operator
```
make upgrade-bundle
```
7. run tests
8. clean up
```
make cleanup-bundle
```
