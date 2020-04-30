# IBM Common Service Operator

This operator is a bridge to connect CloudPaks and ODLM/Common Services, and it can be also installed in standalone mode.


## Introduction

When you install this operator:

1. ODLM will be installed in all namespaces mode
1. `ibm-common-services` namespace will be created
1. OperandRegistry and OperandConfig of Common Services will be created under `ibm-common-services` namespace


## Design

* [IBM Common Service Operator Design](docs/design.md)


## Install Common Services

* [Install Common Services standalone](docs/install.md)
* [Install Common Services with CloudPaks](docs/cloudpak-integration.md)
