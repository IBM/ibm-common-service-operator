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

package v3

import (
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ServiceConfig struct {
	Name               string                          `json:"name"`
	Spec               map[string]runtime.RawExtension `json:"spec"`
	ManagementStrategy string                          `json:"managementStrategy,omitempty"`
}

// CommonServiceSpec defines the desired state of CommonService
type CommonServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Features            *Features            `json:"features,omitempty"`
	InstallPlanApproval olmv1alpha1.Approval `json:"installPlanApproval,omitempty"`
	ManualManagement    bool                 `json:"manualManagement,omitempty"`
	RouteHost           string               `json:"routeHost,omitempty"`
	Size                string               `json:"size,omitempty"`
	Services            []ServiceConfig      `json:"services,omitempty"`
	StorageClass        string               `json:"storageClass,omitempty"`
	BYOCACertificate    bool                 `json:"BYOCACertificate,omitempty"`
	ProfileController   string               `json:"profileController,omitempty"`
	License             LicenseList          `json:"license"`
}

// LicenseList defines the license specification in CSV
type LicenseList struct {
	// Accepting the license - URL: https://ibm.biz/integration-licenses
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:hidden"
	Accept bool `json:"accept"`
	// The type of license being accepted.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:hidden"
	Use string `json:"use,omitempty"`
	// The license being accepted where the component has multiple.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:hidden"
	License string `json:"license,omitempty"`
	// The license key for this deployment.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:hidden"
	Key string `json:"key,omitempty"`
}

// Features defines the configurations of Cloud Pak Services
type Features struct {
	Bedrockshim *Bedrockshim `json:"bedrockshim,omitempty"`
	APICatalog  *APICatalog  `json:"apiCatalog,omitempty"`
}

// APICatalog defines the configuration of APICatalog
type APICatalog struct {
	StorageClass string `json:"storageClass,omitempty"`
}

// Bedrockshim defines the configuration of Bedrockshim
type Bedrockshim struct {
	Enabled                   bool `json:"enabled,omitempty"`
	CrossplaneProviderRemoval bool `json:"crossplaneProviderRemoval,omitempty"`
}

// BedrockOperator maintains a list of bedrock operators
type BedrockOperator struct {
	Name               string `json:"name,omitempty"`
	Version            string `json:"version,omitempty"`
	OperatorStatus     string `json:"operatorStatus,omitempty"`
	SubscriptionStatus string `json:"subscriptionStatus,omitempty"`
	InstallPlanName    string `json:"installPlanName,omitempty"`
	Troubleshooting    string `json:"troubleshooting,omitempty"`
}

// CommonServiceStatus defines the observed state of CommonService
type CommonServiceStatus struct {
	Phase            string            `json:"phase,omitempty"`
	BedrockOperators []BedrockOperator `json:"bedrockOperators,omitempty"`
	OverallStatus    string            `json:"overallStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="CommonService"

// CommonService is the Schema for the commonservices API
type CommonService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CommonServiceSpec   `json:"spec,omitempty"`
	Status CommonServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CommonServiceList contains a list of CommonService
type CommonServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CommonService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CommonService{}, &CommonServiceList{})
}
