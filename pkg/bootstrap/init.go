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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	namespace, err := ioutil.ReadFile(resourcesDir + "/namespace.yaml")
	if err != nil {
		return err
	}

	ns, err := yamlToObject(namespace)
	if err != nil {
		return err
	}
	if err := createObject(ns, client); err != nil {
		return err
	}

	// install operator
	klog.Info("install ODLM operator")
	subscription, err := ioutil.ReadFile(resourcesDir + "/odlm-subscription.yaml")
	if err != nil {
		return err
	}

	if err := createOrUpdateFromYaml(subscription, client, reader); err != nil {
		return err
	}

	// create operandConfig and operandRegistry
	klog.Info("create OperandConfig and OperandRegistry")
	operandConfig, err := ioutil.ReadFile(resourcesDir + "/cs-operandconfig.yaml")
	if err != nil {
		return err
	}
	operandRegistry, err := ioutil.ReadFile(resourcesDir + "/cs-operandregistry.yaml")
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

func yamlToObject(yamlContent []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	jsonSpec, err := yaml.YAMLToJSON(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("could not convert yaml to json: %v", err)
	}

	if err := obj.UnmarshalJSON(jsonSpec); err != nil {
		return nil, fmt.Errorf("could not unmarshal resource: %v", err)
	}

	return obj, nil
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
	obj, err := yamlToObject(yamlContent)
	if err != nil {
		return err
	}

	objInCluster, err := getObject(obj, reader)
	if errors.IsNotFound(err) {
		return createObject(obj, client)
	} else if err != nil {
		return err
	}

	version, _ := strconv.Atoi(obj.GetAnnotations()["version"])
	versionInCluster, _ := strconv.Atoi(objInCluster.GetAnnotations()["version"])

	// TODO: deep merge and update
	if version > versionInCluster {
		return updateObject(obj, client)
	}

	return nil
}
