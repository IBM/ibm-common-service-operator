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

package deploy

import (
	"context"
	"fmt"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	util "github.com/IBM/ibm-common-service-operator/controllers/common"
)

type Manager struct {
	client.Client
	client.Reader
}

// NewDeployManager is the way to create a Manager struct
func NewDeployManager(mgr manager.Manager) *Manager {
	return &Manager{
		Client: mgr.GetClient(),
		Reader: mgr.GetAPIReader(),
	}
}

// GetObject get k8s resource with the unstructured object
func (d *Manager) GetObject(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())

	err := d.Reader.Get(context.TODO(), types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, found)

	return found, err
}

// CreateObject create k8s resource with the unstructured object
func (d *Manager) CreateObject(obj *unstructured.Unstructured) error {
	err := d.Client.Create(context.TODO(), obj)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("could not Create resource: %v", err)
	}
	return nil
}

// DeleteObject delete k8s resource with the unstructured object
func (d *Manager) DeleteObject(obj *unstructured.Unstructured) error {
	err := d.Client.Delete(context.TODO(), obj)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("could not Delete resource: %v", err)
	}
	return nil
}

// UpdateObject update k8s resource with the unstructured object
func (d *Manager) UpdateObject(obj *unstructured.Unstructured) error {
	if err := d.Client.Update(context.TODO(), obj); err != nil {
		return fmt.Errorf("could not update resource: %v", err)
	}
	return nil
}

// CreateFromYaml create k8s resource with the YAML content
func (d *Manager) CreateFromYaml(yamlContent []byte) error {
	objects, err := util.YamlToObjects(yamlContent)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		gvk := obj.GetObjectKind().GroupVersionKind()
		if _, err := d.GetObject(obj); err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("create resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
				if err := d.CreateObject(obj); err != nil {
					return err
				}
				continue
			}
			return err
		}
	}
	return nil
}

// DeleteFromYaml delete k8s resource with the YAML content
func (d *Manager) DeleteFromYaml(yamlContent []byte) error {
	objects, err := util.YamlToObjects(yamlContent)
	if err != nil {
		return err
	}

	for i := len(objects) - 1; i >= 0; i-- {
		obj := objects[i]
		gvk := obj.GetObjectKind().GroupVersionKind()
		klog.Infof("delete resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
		if err := d.DeleteObject(obj); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// GetAnnotations get the annotations from operator's deployment
func (d *Manager) GetAnnotations() (map[string]string, error) {
	deploy, err := d.GetDeployment()
	if err != nil {
		return nil, err
	}

	// Get all the resources from the deployment annotations
	return deploy.Spec.Template.GetAnnotations(), nil
}

func (d *Manager) GetDeployment() (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{}
	deployName, err := util.GetOperatorName()
	if err != nil {
		return nil, fmt.Errorf("could not find the operator name: %v", err)
	}
	deployNs, err := util.GetOperatorNamespace()
	if err != nil {
		return nil, fmt.Errorf("could not find the operator namespace: %v", err)
	}

	// Retrieve operator deployment, retry 3 times
	if err := utilwait.PollImmediate(time.Minute, time.Minute*3, func() (done bool, err error) {
		err = d.Reader.Get(context.TODO(), types.NamespacedName{Name: deployName, Namespace: deployNs}, deploy)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}); err != nil {
		return nil, err
	}
	return deploy, nil
}

// DeleteOperator delete operator's csv and subscription from specific namespace
func (d *Manager) DeleteOperator(name, namespace string) error {
	// Get existing operator's subscription
	subName := name
	subNs := namespace
	key := types.NamespacedName{Name: subName, Namespace: subNs}
	sub := &olmv1alpha1.Subscription{}
	if err := d.Reader.Get(context.TODO(), key, sub); err != nil {
		if errors.IsNotFound(err) {
			klog.V(3).Infof("NotFound Subscription %s from the namespace %s", subName, subNs)
		} else {
			klog.Errorf("failed to get Subscription %s from the namespace %s: %v", subName, subNs, err)
		}
		return client.IgnoreNotFound(err)
	}

	// Delete existing operator's csv
	csvName := sub.Status.InstalledCSV
	csvNs := namespace
	if csvName != "" {
		csv := &olmv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      csvName,
				Namespace: csvNs,
			},
		}
		if err := d.Client.Delete(context.TODO(), csv); err != nil && !errors.IsNotFound(err) {
			klog.Errorf("failed to delete Cluster Service Version %s from the namespace %s: %v", csvName, csvNs, err)
			return err
		}
	}

	// Delete existing operator's subscription
	if err := d.Client.Delete(context.TODO(), sub); err != nil && !errors.IsNotFound(err) {
		klog.Errorf("failed to delete Subscription %s from namespace %s: %v", subName, subNs, err)
		return err
	}
	return nil
}
