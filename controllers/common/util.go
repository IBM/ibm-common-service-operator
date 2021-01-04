//
// Copyright 2021 IBM Corporation
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

package common

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	utilyaml "github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/IBM/ibm-common-service-operator/controllers/constant"
)

type csMaps struct {
	NsMappingList []nsMapping `json:"namespaceMapping"`
	DefaultCsNs   string      `json:"defaultCsNs"`
}

type nsMapping struct {
	RequestNS string `json:"requested-from-namespace"`
	CsNs      string `json:"map-common-service-namespace"`
}

// YamlToObjects convert YAML content to unstructured objects
func YamlToObjects(yamlContent []byte) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured

	// This step is for converting large yaml file, we can remove it after using "apimachinery" v0.19.0
	if len(yamlContent) > 1024*64 {
		object, err := YamlToObject(yamlContent)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
		return objects, nil
	}

	yamlDecoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	reader := json.YAMLFramer.NewFrameReader(ioutil.NopCloser(bytes.NewReader(yamlContent)))
	decoder := streaming.NewDecoder(reader, yamlDecoder)
	for {
		obj, _, err := decoder.Decode(nil, nil)
		if err != nil {
			if err == io.EOF {
				break
			}
			klog.Infof("error convert object: %v", err)
			continue
		}

		switch t := obj.(type) {
		case *unstructured.Unstructured:
			objects = append(objects, t)
		default:
			return nil, fmt.Errorf("failed to convert object %s", reflect.TypeOf(obj))
		}
	}

	return objects, nil
}

// YamlToObject converting large yaml file, we can remove it after using "apimachinery" v0.19.0
func YamlToObject(yamlContent []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	jsonSpec, err := utilyaml.YAMLToJSON(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("could not convert yaml to json: %v", err)
	}

	if err := obj.UnmarshalJSON(jsonSpec); err != nil {
		return nil, fmt.Errorf("could not unmarshal resource: %v", err)
	}

	return obj, nil
}

// NewUnstructured return Unstructured object
func NewUnstructured(group, kind, version string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Kind:    kind,
		Version: version})
	return u
}

// NewUnstructuredList return UnstructuredList object
func NewUnstructuredList(group, kind, version string) *unstructured.UnstructuredList {
	ul := &unstructured.UnstructuredList{}
	ul.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Kind:    kind,
		Version: version})
	return ul
}

// GetOperatorName return the operator name
func GetOperatorName() (string, error) {
	operatorName, found := os.LookupEnv(constant.OperatorNameEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", constant.OperatorNameEnvVar)
	}
	if len(operatorName) == 0 {
		return "", fmt.Errorf("%s must not be empty", constant.OperatorNameEnvVar)
	}
	return operatorName, nil
}

// GetOperatorNamespace returns the namespace the operator should be running in.
func GetOperatorNamespace() (string, error) {
	ns, found := os.LookupEnv(constant.OperatorNamespaceEnvVar)
	if !found {
		nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("namespace not found for current environment")
			}
			return "", err
		}
		ns = strings.TrimSpace(string(nsBytes))
	}
	if len(ns) == 0 {
		return "", fmt.Errorf("operator namespace is empty")
	}
	klog.V(1).Info("Found namespace Namespace", ns)
	return ns, nil
}

// Contains returns whether the sub-string is contained
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Reverse resverses the string
func Reverse(original []string) []string {
	reversed := make([]string, 0, len(original))
	for i := len(original) - 1; i >= 0; i-- {
		reversed = append(reversed, original[i])
	}
	return reversed
}

// Namespacelize adds the namespace specified
func Namespacelize(resource, ns string) string {
	return strings.ReplaceAll(resource, "placeholder", ns)
}

// Get ConfigMap of Common Services Maps
func getCmOfMapCs(r client.Reader) *corev1.ConfigMap {
	cmName := constant.CsMapConfigMap
	cmNs := "kube-public"
	csConfigmap := &corev1.ConfigMap{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: cmNs}, csConfigmap)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("common service configmap kube-public/common-service-maps is not found: %v", err)
			return nil
		}
		klog.Errorf("failed to fetch configmap kube-public/common-service-maps: %v", err)
		return nil
	}
	return csConfigmap
}

// GetMasterNs gets MasterNamespaces of deploying Common Services
func GetMasterNs(r client.Reader) string {
	operatorNs, err := GetOperatorNamespace()
	if err != nil {
		klog.Errorf("Getting operator namespace failed: %v", err)
		return ""
	}

	csConfigmap := getCmOfMapCs(r)
	commonServiceMaps := csConfigmap.Data["common-service-maps.yaml"]
	var cmData csMaps
	if err := utilyaml.Unmarshal([]byte(commonServiceMaps), &cmData); err != nil {
		klog.Errorf("Failed to fetch data of configmap common-service-maps: %v", err)
		return ""
	}

	// default masternamepsace
	var masterNs string
	masterNs = constant.MasterNamespace

	for _, nsMapping := range cmData.NsMappingList {
		if nsMapping.RequestNS == operatorNs {
			masterNs = nsMapping.CsNs
			break
		}
		if nsMapping.CsNs == operatorNs {
			masterNs = operatorNs
			break
		}
	}

	return masterNs
}
