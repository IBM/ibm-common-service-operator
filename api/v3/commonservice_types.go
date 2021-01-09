//
// Copyright 2020 IBM Corporation
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ServiceConfig struct {
	Name string                          `json:"name"`
	Spec map[string]runtime.RawExtension `json:"spec"`
}

// CommonServiceSpec defines the desired state of CommonService
type CommonServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ManualManagement bool            `json:"manualManagement,omitempty"`
	Size             string          `json:"size,omitempty"`
	Services         []ServiceConfig `json:"services,omitempty"`
	StorageClass     string          `json:"storageClass,omitempty"`
}

// CommonServiceStatus defines the observed state of CommonService
type CommonServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
