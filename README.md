# IBM Common Service Operator

This operator is a bridge to connect CloudPaks and ODLM/Common Services.


## Introduction

When you install this operator:

1. ODLM will be installed in all namespaces mode
1. `ibm-common-services` namespace will be created
1. OperandRegistry and OperandConfig of Common Services will be created under `ibm-common-services` namespace


## Install Common Services

1. Install `ibm-common-service-operator` in any namespace
1. Navigate to `ibm-common-services` namespce and create OperandRequest to trigger common services installation
