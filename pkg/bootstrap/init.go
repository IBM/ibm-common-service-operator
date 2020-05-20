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

package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	utilyaml "github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// InitResources initialize resources at the bootstrap of operator
func InitResources(mgr manager.Manager) error {
	client := mgr.GetClient()
	reader := mgr.GetAPIReader()

	resourcesDir := os.Getenv("RESOURCES_DIR")

	// create namespace
	klog.Info("create ibm-common-services namespace")
	namespace, err := ioutil.ReadFile(filepath.Join(resourcesDir, "namespace.yaml"))
	if err != nil {
		return err
	}

	objects, err := yamlToObjects(namespace)
	if err != nil {
		return err
	}
	if err := createObject(objects[0], client); err != nil {
		return err
	}

	klog.Info("check existing ODLM operator")
	err = deleteExistingODLM(client)
	if err != nil {
		return err
	}
	// install operator
	klog.Info("install ODLM operator")
	subscription, err := ioutil.ReadFile(filepath.Join(resourcesDir, "odlm-subscription.yaml"))
	if err != nil {
		return err
	}

	if err := createOrUpdateFromYaml(subscription, client, reader); err != nil {
		return err
	}

	// create extra yamls
	klog.Info("create extra yaml resources")
	if err := createOrUpdateResourcesFromDir(filepath.Join(resourcesDir, "extra"), client, reader); err != nil {
		return err
	}

	// create operandConfig and operandRegistry
	klog.Info("create OperandConfig and OperandRegistry")
	operandConfig, err := ioutil.ReadFile(filepath.Join(resourcesDir, "cs-operandconfig.yaml"))
	if err != nil {
		return err
	}
	operandRegistry, err := ioutil.ReadFile(filepath.Join(resourcesDir, "cs-operandregistry.yaml"))
	if err != nil {
		return err
	}

	timeout := time.After(300 * time.Second)
	ticker := time.NewTicker(30 * time.Second)
	for {
		klog.Info("try to create IBM Common Services OperandConfig and OperandRegistry")
		select {
		case <-timeout:
			return fmt.Errorf("timeout to create the ODLM resource")
		case <-ticker.C:
			// create OperandConfig
			errConfig := createOrUpdateFromYaml(operandConfig, client, reader)
			if errConfig != nil {
				klog.Error("create OperandConfig error with: ", errConfig)
			}

			// create OperandRegistry
			errRegistry := createOrUpdateFromYaml(operandRegistry, client, reader)
			if errRegistry != nil {
				klog.Error("create OperandRegistry error with: ", errRegistry)
			}

			if errConfig == nil && errRegistry == nil {
				return nil
			}
		}
	}
}

func getObject(obj *unstructured.Unstructured, reader client.Reader) (*unstructured.Unstructured, error) {
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())

	err := reader.Get(context.TODO(), types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, found)

	return found, err
}

func createObject(obj *unstructured.Unstructured, client client.Client) error {
	err := client.Create(context.TODO(), obj)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("could not Create resource: %v", err)
	}

	return nil
}

func updateObject(obj *unstructured.Unstructured, client client.Client) error {
	if err := client.Update(context.TODO(), obj); err != nil {
		return fmt.Errorf("could not update resource: %v", err)
	}

	return nil
}

func createOrUpdateFromYaml(yamlContent []byte, client client.Client, reader client.Reader) error {
	objects, err := yamlToObjects(yamlContent)
	if err != nil {
		return err
	}

	var errMsg error

	for _, obj := range objects {
		gvk := obj.GetObjectKind().GroupVersionKind()

		objInCluster, err := getObject(obj, reader)
		if errors.IsNotFound(err) {
			klog.Infof("create resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
			if err := createObject(obj, client); err != nil {
				errMsg = err
			}
			continue
		} else if err != nil {
			errMsg = err
			continue
		}

		annoVersion := obj.GetAnnotations()["version"]
		if annoVersion == "" {
			annoVersion = "0"
		}
		annoVersionInCluster := objInCluster.GetAnnotations()["version"]
		if annoVersionInCluster == "" {
			annoVersionInCluster = "0"
		}

		version, _ := strconv.Atoi(annoVersion)
		versionInCluster, _ := strconv.Atoi(annoVersionInCluster)

		// TODO: deep merge and update
		if version > versionInCluster {
			klog.Infof("update resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
			resourceVersion := objInCluster.GetResourceVersion()
			obj.SetResourceVersion(resourceVersion)
			if err := updateObject(obj, client); err != nil {
				errMsg = err
			}
		}
	}

	return errMsg
}

func yamlToObjects(yamlContent []byte) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured

	// This step is for converting large yaml file, we can remove it after using "apimachinery" v0.19.0
	if len(yamlContent) > 1024*64 {
		object, err := yamlToObject(yamlContent)
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

// This function is for converting large yaml file, we can remove it after using "apimachinery" v0.19.0
func yamlToObject(yamlContent []byte) (*unstructured.Unstructured, error) {
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

func createOrUpdateResourcesFromDir(dir string, client client.Client, reader client.Reader) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	var yamlFiles []string

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yaml" {
			yamlFiles = append(yamlFiles, file.Name())
		}
	}

	for _, file := range yamlFiles {
		yamlContent, err := ioutil.ReadFile(filepath.Join(dir, file))
		if err != nil {
			return err
		}
		if err := createOrUpdateFromYaml(yamlContent, client, reader); err != nil {
			return err
		}
	}

	return nil
}

func deleteExistingODLM(client client.Client) error {
	// delete subscription
	objSub := &unstructured.Unstructured{}
	objSub.SetGroupVersionKind(schema.GroupVersionKind{Group: "operators.coreos.com", Kind: "Subscription", Version: "v1alpha1"})
	objSub.SetName("operand-deployment-lifecycle-manager-app")
	objSub.SetNamespace("ibm-common-services")
	err := client.Delete(context.TODO(), objSub)
	if err != nil && !errors.IsNotFound(err) {
		klog.Error("Failed to delete ODLM subscription in the ibm-common-services namespace")
		return err
	}

	// delete csv v1.1.0
	objCSV := &unstructured.Unstructured{}
	objCSV.SetGroupVersionKind(schema.GroupVersionKind{Group: "operators.coreos.com", Kind: "ClusterServiceVersion", Version: "v1alpha1"})
	objCSV.SetNamespace("ibm-common-services")
	objCSV.SetName("operand-deployment-lifecycle-manager.v1.1.0")
	err = client.Delete(context.TODO(), objCSV)
	if err != nil && !errors.IsNotFound(err) {
		klog.Error("Failed to delete ODLM Cluster Service Version v1.1.0 in the ibm-common-services namespace")
		return err
	}

	// delete csv v1.2.0
	objCSV.SetName("operand-deployment-lifecycle-manager.v1.2.0")
	err = client.Delete(context.TODO(), objCSV)
	if err != nil && !errors.IsNotFound(err) {
		klog.Error("Failed to delete ODLM Cluster Service Version v1.2.0 in the ibm-common-services namespace")
		return err
	}
	return nil
}
