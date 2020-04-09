package bootstrap

import (
	"context"
	"fmt"
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

	// create namespace
	klog.Info("create ibm-common-services namespace")
	ns, err := yamlToObject([]byte(namespace))
	if err != nil {
		return err
	}
	if err := createObject(ns, client); err != nil {
		return err
	}

	// install operator
	klog.Info("install ODLM operator")
	if err := createOrUpdateFromYaml([]byte(subscription), client, reader); err != nil {
		return err
	}

	timeout := time.After(300 * time.Second)
	ticker := time.Tick(30 * time.Second)
	for {
		klog.Info("try to create IBM Common Services OperandConfig and OperandRegistry")
		select {
		case <-timeout:
			return fmt.Errorf("Timeout to create the ODLM resource")
		case <-ticker:
			// create OperandConfig
			errConfig := createOrUpdateFromYaml([]byte(operandConfig), client, reader)
			if errConfig != nil {
				klog.Error("create OperandConfig error with: ", errConfig)
			}

			// create OperandRegistry
			errRegistry := createOrUpdateFromYaml([]byte(operandRegistry), client, reader)
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

	version := obj.GetAnnotations()["version"]
	versionInCluster := objInCluster.GetAnnotations()["version"]

	// TODO: deep merge and update
	if version > versionInCluster {
		return updateObject(obj, client)
	}

	return nil
}
