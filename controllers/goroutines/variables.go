//
// Copyright 2022 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package goroutines

import "github.com/IBM/ibm-common-service-operator/controllers/constant"

var (
	OperatorAPIGroupVersion = "operator.ibm.com/v1"

	SecretShareAPIGroupVersion = "ibmcpcs.ibm.com/v1"
	SecretShareKind            = "SecretShare"
	SecretShareCppName         = constant.IBMCPPCONFIG

	IAMSaaSDeployNames = []string{"platform-identity-management", "platform-identity-provider", "platform-auth-service"}
	IAMDeployNames     = []string{"platform-identity-management", "platform-identity-provider", "platform-auth-service"}
	IAMJobNames        = []string{"oidc-client-registration"}

	NSSKinds    = []string{"NamespaceScope"}
	NSSCRList   = []string{"common-service", "nss-odlm-scope"}
	NSSSourceCR = "common-service"
	NSSTargetCR = "nss-odlm-scope"

	MasterNamespace   string
	ServicesNamespace string
)
