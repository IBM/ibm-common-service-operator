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
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/IBM/ibm-common-service-operator/v4/controllers/constant"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type CSData struct {
	Channel                 string
	Version                 string
	CPFSNs                  string
	ServicesNs              string
	OperatorNs              string
	CatalogSourceName       string
	CatalogSourceNs         string
	IsolatedModeEnable      string
	ApprovalMode            string
	OnPremMultiEnable       string
	WatchNamespaces         string
	CloudPakThemes          string
	CloudPakThemesVersion   string
	ExcludedCatalog         string
	StatusMonitoredServices string
}

// +kubebuilder:pruning:PreserveUnknownFields
type ExtensionWithMarker struct {
	runtime.RawExtension `json:",inline"`
}

// +kubebuilder:pruning:PreserveUnknownFields
type ServiceConfig struct {
	Name               string                         `json:"name"`
	Spec               map[string]ExtensionWithMarker `json:"spec,omitempty"`
	ManagementStrategy string                         `json:"managementStrategy,omitempty"`
	Resources          []ExtensionWithMarker          `json:"resources,omitempty"`
}

// CommonServiceSpec defines the desired state of CommonService
type CommonServiceSpec struct {
	Features *Features `json:"features,omitempty"`
	// InstallPlanApproval sets the approval mode for ODLM and other
	// foundational services: Manual or Automatic
	InstallPlanApproval olmv1alpha1.Approval `json:"installPlanApproval,omitempty"`
	ManualManagement    bool                 `json:"manualManagement,omitempty"`
	// FipsEnabled enables FIPS mode for foundational services
	FipsEnabled bool `json:"fipsEnabled,omitempty"`
	// RouteHost describes the hostname for the foundational services route,
	// and can only be configured pre-installation of IM
	RouteHost string `json:"routeHost,omitempty"`
	// Size describes the T-shirt size of foundational services: starterset,
	// small, medium, or large
	Size string `json:"size,omitempty"`
	// Services describes the CPU, memory, and replica configuration for
	// individual services in foundational services
	Services []ServiceConfig `json:"services,omitempty"`
	// StorageClass describes the storage class to use for the foundational
	// services PVCs
	StorageClass string `json:"storageClass,omitempty"`
	// BYOCACertificate enables the option to replace the cs-ca-certificate with
	// your own CA certificate
	BYOCACertificate bool `json:"BYOCACertificate,omitempty"`
	// ProfileController enables turbonomic to automatically handle sizing of
	// foundational services. Default value is 'default'
	ProfileController string `json:"profileController,omitempty"`
	// ServicesNamespace describes the namespace where operands will be created
	// in such as OperandRegistry and OperandConfig. This will also apply to all
	// services in foundational services, e.g. IM will create operands in this
	// namespace
	ServicesNamespace ServicesNamespace `json:"servicesNamespace,omitempty"`
	// OperatorNamespace describes the namespace where operators will be
	// installed in such as ODLM. This will also apply to all services in
	// foundational services, e.g. ODLM will install IM operator in this
	// namespace
	OperatorNamespace OperatorNamespace `json:"operatorNamespace,omitempty"`
	// CatalogName is the name of the CatalogSource that will be used for ODLM
	// and other services in foundational services
	CatalogName CatalogName `json:"catalogName,omitempty"`
	// CatalogNamespace is the namespace of the CatalogSource that will be used
	// for ODLM and other services in foundational services
	CatalogNamespace CatalogNamespace `json:"catalogNamespace,omitempty"`
	// DefalutAdminUser is the name of the default admin user for foundational
	// services IM, default is cpadmin
	DefaultAdminUser string `json:"defaultAdminUser,omitempty"`
	// Labels describes foundational services will use this
	// labels to labels their corresponding resources
	Labels map[string]string `json:"labels,omitempty"`
	// +optional
	// HugePages describes the hugepages settings for foundational services
	// +kubebuilder:pruning:PreserveUnknownFields
	HugePages *HugePages `json:"hugepages,omitempty"`
	// OperatorConfigs is a list of configurations to be applied to operators via CSV updates
	// +kubebuilder:pruning:PreserveUnknownFields
	OperatorConfigs []OperatorConfig `json:"operatorConfigs,omitempty"`
	// +optional
	License LicenseList `json:"license"`
}

// OperatorConfig is configuration composed of key-value pairs to be injected into specified CSVs
type OperatorConfig struct {
	// Name is the name of the operator as requested in an OperandRequest
	Name string `json:"name,omitempty"`
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	// UserManaged is a flag that indicates whether the operator is managed by
	// user or not. If set the value will propagate down to UserManaged field
	// in the OperandRegistry
	UserManaged bool `json:"userManaged,omitempty"`
}

// LicenseList defines the license specification in CSV
type LicenseList struct {
	// Accepting the license - URL: https://ibm.biz/icpfs39license
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

// BedrockOperator describes a list of foundational services' operators currently installed for this tenant.
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

type ConfigurableCR struct {
	// ObjectName is the name of the configurable CommonService CR
	ObjectName string `json:"objectName,omitempty"`
	// ApiVersion is the api version of the configurable CommonService CR
	APIVersion string `json:"apiVersion,omitempty"`
	// Namespace is the namespace of the configurable CommonService CR
	Namespace string `json:"namespace,omitempty"`
	// Kind is the kind of the configurable CommonService CR
	Kind string `json:"kind,omitempty"`
}

// ConfigStatus describes various configuration currently applied onto the foundational services installer.
type ConfigStatus struct {
	// CatalogName is the name of the CatalogSource foundational services is using
	CatalogName CatalogName `json:"catalogName,omitempty"`
	// CatalogNamespace is the namesapce of the CatalogSource
	CatalogNamespace CatalogNamespace `json:"catalogNamespace,omitempty"`
	// OperatorNamespace is the namespace of where the foundational services'
	// operators will be installed in.
	OperatorNamespace OperatorNamespace `json:"operatorNamespace,omitempty"`
	// OperatorDeployed indicates whether the OperandRegistry has been created
	// or not.
	OperatorDeployed bool `json:"operatorDeployed,omitempty"`
	// ServicesNamespace is the namespace where the foundational services'
	// operands will be created in.
	ServicesNamespace ServicesNamespace `json:"servicesNamespace,omitempty"`
	// ServicesDeployed indicates whether the OperandConfig has been created or
	// not.
	ServicesDeployed bool `json:"servicesDeployed,omitempty"`
	// Configurable indicates whether this CommonService CR is the one
	// that can be used to configure the foundational services' installer. Other
	// CommonService CRs configurations will not take effect, except for sizing
	Configurable bool `json:"configurable,omitempty"`
	// TopologyConfigurableCRs describes the configurable CommonService CR
	TopologyConfigurableCRs []ConfigurableCR `json:"topologyConfigurableCRs,omitempty"`
}

// HugePages defines the various hugepages settings applied to foundational services
type HugePages struct {
	// HugePagesEnabled enables hugepages settings for foundational services
	Enable bool `json:"enable,omitempty"`
	// HugePagesSizes describes the size of the hugepages
	HugePagesSizes map[string]string `json:"-"`
}

// CommonServiceStatus defines the observed state of CommonService
type CommonServiceStatus struct {
	// Phase describes the phase of the overall installation
	Phase            string            `json:"phase,omitempty"`
	BedrockOperators []BedrockOperator `json:"bedrockOperators,omitempty"`
	// OverallStatus describes whether the Installation for the foundational services has succeeded or not
	OverallStatus string       `json:"overallStatus,omitempty"`
	ConfigStatus  ConfigStatus `json:"configStatus,omitempty"`
	// Conditions represents the current state of CommonService
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Conditions",xDescriptors="urn:alm:descriptor:io.kubernetes.conditions"
	Conditions []CommonServiceCondition `json:"conditions,omitempty"`
}

// CommonServiceCondition defines the observed condition of CommonService
type CommonServiceCondition struct {
	// Type of condition.
	// +optional
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	// +optional
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	// +optional
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// ConditionType is the condition of a service.
type ConditionType string

// ResourceType is the type of condition use.
type ResourceType string

const (
	ConditionTypeBlocked     ConditionType = "Blocked"
	ConditionTypeReady       ConditionType = "Ready"
	ConditionTypeWarning     ConditionType = "Warning"
	ConditionTypeError       ConditionType = "Error"
	ConditionTypePending     ConditionType = "Pending"
	ConditionTypeReconciling ConditionType = "Reconciling"
)

const (
	ConditionReasonReconcile = "StartReconciling"
	ConditionReasonInit      = "UpdatingResources"
	ConditionReasonConfig    = "Configuring"
	ConditionReasonWarning   = "WarningOccurred"
	ConditionReasonError     = "ReconcileError"
	ConditionReasonReady     = "ReconcileSucceeded"
)

const (
	ConditionMessageReconcile = "reconciling CommonService CR."
	ConditionMessageInit      = "initializing/updating: waiting for OperandRegistry and OperandConfig to become ready."
	ConditionMessageConfig    = "configuring CommonService CR."
	ConditionMessageMissSC    = "warning: StorageClass is not configured in CommonService CR, if KeyCloak or IBM IM service will be deployed, please configure StorageClass in the CS CR. Refer to the documentation for more information: https://www.ibm.com/docs/en/cloud-paks/foundational-services/4.6?topic=options-configuring-foundational-services#storage-class"
	ConditionMessageReady     = "CommonService CR is ready."
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="foundationservices.cloudpak.ibm.com=crd"
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="CommonService"

// CommonService is the Schema for the commonservices API. This API is used to
// configure general foundational services configurations, such as sizing,
// catalogsource, etc. See description of fields for more details. An instance
// of this CRD is automatically created by the ibm-common-service-operator upon
// installation to trigger the installation of critical installer components
// such as operand-deployment-lifecycle-manager (ODLM), which is required to
// further install other services from foundational services, such as IM.
type CommonService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	Spec CommonServiceSpec `json:"spec,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
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
	r.Status.ConfigStatus.OperatorDeployed = operatorDeployed
	if r.Spec.OperatorNamespace != "" && !operatorDeployed {
		r.Status.ConfigStatus.OperatorNamespace = r.Spec.OperatorNamespace
	} else {
		r.Status.ConfigStatus.OperatorNamespace = OperatorNamespace(CSData.CPFSNs)
	}

	r.Status.ConfigStatus.ServicesDeployed = serviceDeployed
	if r.Spec.ServicesNamespace != "" && !serviceDeployed {
		r.Status.ConfigStatus.ServicesNamespace = r.Spec.ServicesNamespace
	} else {
		r.Status.ConfigStatus.ServicesNamespace = ServicesNamespace(CSData.ServicesNs)
	}

	if r.Spec.CatalogName != "" {
		r.Status.ConfigStatus.CatalogName = r.Spec.CatalogName
	} else {
		r.Status.ConfigStatus.CatalogName = CatalogName(CSData.CatalogSourceName)
	}

	if r.Spec.CatalogNamespace != "" {
		r.Status.ConfigStatus.CatalogNamespace = r.Spec.CatalogNamespace
	} else {
		r.Status.ConfigStatus.CatalogNamespace = CatalogNamespace(CSData.CatalogSourceNs)
	}
	r.Status.ConfigStatus.OperatorDeployed = true
	r.Status.ConfigStatus.ServicesDeployed = true
	r.Status.ConfigStatus.Configurable = true

	r.UpdateTopologyCR(CSData)
}

func (r *CommonService) UpdateNonMasterConfigStatus(CSData *CSData) {
	r.Status.ConfigStatus.OperatorNamespace = OperatorNamespace(CSData.OperatorNs)
	r.Status.ConfigStatus.ServicesNamespace = ServicesNamespace(CSData.ServicesNs)
	r.Status.ConfigStatus.CatalogName = CatalogName(CSData.CatalogSourceName)
	r.Status.ConfigStatus.CatalogNamespace = CatalogNamespace(CSData.CatalogSourceNs)
	r.Status.ConfigStatus.OperatorDeployed = true
	r.Status.ConfigStatus.ServicesDeployed = true
	r.Status.ConfigStatus.Configurable = false

	r.UpdateTopologyCR(CSData)
}

// SetPendingCondition sets a condition to claim Pending.
func (r *CommonService) SetPendingCondition(name string, ct ConditionType, cs corev1.ConditionStatus, reason, message string) {
	c := newCondition(ct, cs, reason, message)
	r.setCondition(*c)
}

// SetReadyCondition creates a Condition to claim Ready.
func (r *CommonService) SetReadyCondition(name string, ct ConditionType, cs corev1.ConditionStatus) {
	r.UpdateConditionList(corev1.ConditionFalse)
	c := newCondition(ConditionTypeReady, cs, ConditionReasonReady, ConditionMessageReady)
	r.setCondition(*c)
}

// SetWarningCondition creates a Condition to claim Warning.
func (r *CommonService) SetWarningCondition(name string, ct ConditionType, cs corev1.ConditionStatus, reason, message string) {
	c := newCondition(ct, cs, reason, message)
	r.setCondition(*c)
}

// SetErrorCondition creates a Condition to claim Error.
func (r *CommonService) SetErrorCondition(name string, ct ConditionType, cs corev1.ConditionStatus, reason, message string) {
	r.UpdateConditionList(corev1.ConditionFalse)
	c := newCondition(ct, cs, reason, message)
	r.setCondition(*c)
}

// UpdateConditionList updates the condition list of the CommonService CR
func (r *CommonService) UpdateConditionList(ct corev1.ConditionStatus) {
	// check all the conditions
	for i := range r.Status.Conditions {
		if r.Status.Conditions[i].Type == ConditionTypePending || r.Status.Conditions[i].Type == ConditionTypeReconciling {
			r.Status.Conditions[i].Status = ct
		}
	}
}

// NewCondition Returns a new condition with the given values for CommonService
func newCondition(condType ConditionType, status corev1.ConditionStatus, reason, message string) *CommonServiceCondition {
	now := time.Now().Format(time.RFC3339)
	return &CommonServiceCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     now,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}
}

// GetCondition returns the condition status by given condition type and message
func getCondition(conds *[]CommonServiceCondition, condtype ConditionType, msg string) (int, *CommonServiceCondition) {
	for i, condition := range *conds {
		if condtype == condition.Type && msg == condition.Message {
			return i, &condition
		}
	}
	return -1, nil
}

// SetCondition append a new condition status
func (r *CommonService) setCondition(condition CommonServiceCondition) {
	position, cp := getCondition(&r.Status.Conditions, condition.Type, condition.Message)
	if cp != nil {
		r.Status.Conditions[position] = condition
	} else {
		r.Status.Conditions = append(r.Status.Conditions, condition)
	}
}

func (r *CommonService) UpdateTopologyCR(CSData *CSData) {
	var masterCRSlice []ConfigurableCR
	var csCR ConfigurableCR
	csCR.ObjectName = constant.MasterCR
	csCR.APIVersion = constant.APIVersion
	csCR.Namespace = CSData.OperatorNs
	csCR.Kind = constant.KindCR
	masterCRSlice = append(masterCRSlice, csCR)
	r.Status.ConfigStatus.TopologyConfigurableCRs = masterCRSlice
}

func init() {
	SchemeBuilder.Register(&CommonService{}, &CommonServiceList{})
}
