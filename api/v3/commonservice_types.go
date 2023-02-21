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

type CSData struct {
	Channel            string
	Version            string
	CPFSNs             string
	ServicesNs         string
	OperatorNs         string
	CatalogSourceName  string
	CatalogSourceNs    string
	IsolatedModeEnable string
	ApprovalMode       string
	OnPremMultiEnable  string
	ZenOperatorImage   string
	IsOCP              bool
	WatchNamespaces    string
}

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
	FipsEnabled         bool                 `json:"fipsEnabled,omitempty"`
	RouteHost           string               `json:"routeHost,omitempty"`
	Size                string               `json:"size,omitempty"`
	Services            []ServiceConfig      `json:"services,omitempty"`
	StorageClass        string               `json:"storageClass,omitempty"`
	BYOCACertificate    bool                 `json:"BYOCACertificate,omitempty"`
	ProfileController   string               `json:"profileController,omitempty"`
	ServicesNamespace   ServicesNamespace    `json:"servicesNamespace,omitempty"`
	OperatorNamespace   OperatorNamespace    `json:"operatorNamespace,omitempty"`
	CatalogName         CatalogName          `json:"catalogName,omitempty"`
	CatalogNamespace    CatalogNamespace     `json:"catalogNamespace,omitempty"`

	// +optional
	License LicenseList `json:"license"`
}

// LicenseList defines the license specification in CSV
type LicenseList struct {
	// Accepting the license - URL: https://ibm.biz/integration-licenses
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:hidden"
	// +optional
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

type ServicesNamespace string

type OperatorNamespace string

type CatalogName string

type CatalogNamespace string

type ConfigStatus struct {
	ServicesPlane ServicesPlane `json:"servicesPlane,omitempty"`
	OperatorPlane OperatorPlane `json:"operatorPlane,omitempty"`
	CatalogPlane  CatalogPlane  `json:"catalogPlane,omitempty"`
}

type ServicesPlane struct {
	ServicesNamespace  ServicesNamespace            `json:"servicesNamespace,omitempty"`
	ServicesDeployed   bool                         `json:"servicesDeployed,omitempty"`
	NonDefaultCRstatus map[string]ServicesNamespace `json:"nonDefaultCRstatus,omitempty"`
}

type OperatorPlane struct {
	OperatorNamespace  OperatorNamespace            `json:"operatorNamespace,omitempty"`
	OperatorDeployed   bool                         `json:"operatorDeployed,omitempty"`
	NonDefaultCRstatus map[string]OperatorNamespace `json:"nonDefaultCRstatus,omitempty"`
}

type CatalogPlane struct {
	CatalogName        CatalogName       `json:"catalogName,omitempty"`
	CatalogNamespace   CatalogNamespace  `json:"catalogNamespace,omitempty"`
	NonDefaultCRstatus map[string]string `json:"nonDefaultCRstatus,omitempty"`
}

// CommonServiceStatus defines the observed state of CommonService
type CommonServiceStatus struct {
	Phase            string            `json:"phase,omitempty"`
	BedrockOperators []BedrockOperator `json:"bedrockOperators,omitempty"`
	OverallStatus    string            `json:"overallStatus,omitempty"`
	ConfigStatus     ConfigStatus      `json:"configStatus,omitempty"`
	Configurable     bool              `json:"configurable"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="CommonService"

// CommonService is the Schema for the commonservices API
type CommonService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
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

func (r *CommonService) UpdateConfigStatus(CSData *CSData, operatorDeployed, serviceDeployed bool) {
	r.Status.ConfigStatus.OperatorPlane.OperatorDeployed = operatorDeployed
	if r.Spec.OperatorNamespace != "" && !operatorDeployed {
		r.Status.ConfigStatus.OperatorPlane.OperatorNamespace = r.Spec.OperatorNamespace
	} else {
		r.Status.ConfigStatus.OperatorPlane.OperatorNamespace = OperatorNamespace(CSData.CPFSNs)
	}

	r.Status.ConfigStatus.ServicesPlane.ServicesDeployed = serviceDeployed
	if r.Spec.ServicesNamespace != "" && !serviceDeployed {
		r.Status.ConfigStatus.ServicesPlane.ServicesNamespace = r.Spec.ServicesNamespace
	} else {
		r.Status.ConfigStatus.ServicesPlane.ServicesNamespace = ServicesNamespace(CSData.ServicesNs)
	}

	if r.Spec.CatalogName != "" {
		r.Status.ConfigStatus.CatalogPlane.CatalogName = r.Spec.CatalogName
	} else {
		r.Status.ConfigStatus.CatalogPlane.CatalogName = CatalogName(CSData.CatalogSourceName)
	}

	if r.Spec.CatalogNamespace != "" {
		r.Status.ConfigStatus.CatalogPlane.CatalogNamespace = r.Spec.CatalogNamespace
	} else {
		r.Status.ConfigStatus.CatalogPlane.CatalogNamespace = CatalogNamespace(CSData.CatalogSourceNs)
	}
	r.Status.Configurable = true

}

func init() {
	SchemeBuilder.Register(&CommonService{}, &CommonServiceList{})
}
