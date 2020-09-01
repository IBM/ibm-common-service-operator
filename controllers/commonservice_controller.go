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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	utilyaml "github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	"github.com/IBM/ibm-common-service-operator/controllers/deploy"
	"github.com/IBM/ibm-common-service-operator/controllers/size"
)

// CommonServiceReconciler reconciles a CommonService object
type CommonServiceReconciler struct {
	client.Client
	client.Reader
	*deploy.Manager
	*bootstrap.Bootstrap
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var ctx = context.Background()

// +kubebuilder:rbac:groups=*,resources=*,verbs=*

func (r *CommonServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	// Fetch the CommonService instance
	instance := &apiv3.CommonService{}

	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	klog.Infof("Reconciling CommonService: %s", req.NamespacedName)
	// Check if the CommonService instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if instance.GetDeletionTimestamp() != nil {
		if util.Contains(instance.GetFinalizers(), constant.CommonserviceFinalizer) {
			if err := r.createUninstallJob(); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(instance, constant.CommonserviceFinalizer)
			err := r.Update(ctx, instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !util.Contains(instance.GetFinalizers(), constant.CommonserviceFinalizer) {
		if err := r.addFinalizer(instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Init common servcie bootstrap resource
	if err := r.Bootstrap.InitResources(); err != nil {
		klog.Error("InitResources failed: ", err)
	}

	newConfigs, err := r.getNewConfigs(req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err = r.updateOpcon(newConfigs); err != nil {
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *CommonServiceReconciler) getNewConfigs(req ctrl.Request) ([]interface{}, error) {
	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	if err := r.Client.Get(ctx, req.NamespacedName, cs); err != nil {
		return nil, err
	}
	var newConfigs []interface{}
	switch cs.Object["spec"].(map[string]interface{})["size"] {
	case "small":
		if cs.Object["spec"].(map[string]interface{})["services"] == nil {
			newConfigs = deepMerge(newConfigs, size.Small)
		} else {
			newConfigs = deepMerge(cs.Object["spec"].(map[string]interface{})["services"].([]interface{}), size.Small)
		}
	case "medium":
		if cs.Object["spec"].(map[string]interface{})["services"] == nil {
			newConfigs = deepMerge(newConfigs, size.Medium)
		} else {
			newConfigs = deepMerge(cs.Object["spec"].(map[string]interface{})["services"].([]interface{}), size.Medium)
		}
	case "large":
		if cs.Object["spec"].(map[string]interface{})["services"] == nil {
			newConfigs = deepMerge(newConfigs, size.Large)
		} else {
			newConfigs = deepMerge(cs.Object["spec"].(map[string]interface{})["services"].([]interface{}), size.Large)
		}
	default:
		if cs.Object["spec"].(map[string]interface{})["services"] != nil {
			newConfigs = cs.Object["spec"].(map[string]interface{})["services"].([]interface{})
		}
	}

	return newConfigs, nil
}

func (r *CommonServiceReconciler) updateOpcon(newConfigs []interface{}) error {
	opcon := util.NewUnstructured("operator.ibm.com", "OperandConfig", "v1alpha1")
	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: "ibm-common-services",
	}
	if err := r.Reader.Get(ctx, opconKey, opcon); err != nil {
		klog.Error(err)
		return err
	}
	services := opcon.Object["spec"].(map[string]interface{})["services"].([]interface{})
	for _, service := range services {
		for _, size := range newConfigs {
			if service.(map[string]interface{})["name"].(string) == size.(map[string]interface{})["name"].(string) {
				for cr, spec := range service.(map[string]interface{})["spec"].(map[string]interface{}) {
					if size.(map[string]interface{})["spec"].(map[string]interface{})[cr] == nil {
						continue
					}
					service.(map[string]interface{})["spec"].(map[string]interface{})[cr] = mergeConfig(spec.(map[string]interface{}), size.(map[string]interface{})["spec"].(map[string]interface{})[cr].(map[string]interface{}))
				}
			}
		}
	}

	opcon.Object["spec"].(map[string]interface{})["services"] = services

	if err := r.Update(ctx, opcon); err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

func deepMerge(src []interface{}, dest string) []interface{} {

	jsonSpec, err := utilyaml.YAMLToJSON([]byte(dest))

	if err != nil {
		klog.Error(err)
	}

	// Create a slice for sizes
	var sizes []interface{}

	// Convert sizes string to slice
	err = json.Unmarshal(jsonSpec, &sizes)

	if err != nil {
		klog.Error(err)
	}

	for _, configSize := range sizes {
		for _, config := range src {
			if config.(map[string]interface{})["name"].(string) == configSize.(map[string]interface{})["name"].(string) {
				if config == nil {
					continue
				}
				if configSize == nil {
					configSize = config
					continue
				}
				for cr, size := range mergeConfig(configSize.(map[string]interface{})["spec"].(map[string]interface{}), config.(map[string]interface{})["spec"].(map[string]interface{})) {
					configSize.(map[string]interface{})["spec"].(map[string]interface{})[cr] = size
				}
			}
		}
	}
	return sizes
}

// mergeConfig deep merge two configs
func mergeConfig(defaultMap map[string]interface{}, changedMap map[string]interface{}) map[string]interface{} {
	for key := range defaultMap {
		checkKeyBeforeMerging(key, defaultMap[key], changedMap[key], changedMap)
	}
	return changedMap
}

func checkKeyBeforeMerging(key string, defaultMap interface{}, changedMap interface{}, finalMap map[string]interface{}) {
	if !reflect.DeepEqual(defaultMap, changedMap) {
		switch defaultMap := defaultMap.(type) {
		case map[string]interface{}:
			//Check that the changed map value doesn't contain this map at all and is nil
			if changedMap == nil {
				finalMap[key] = defaultMap
			} else if _, ok := changedMap.(map[string]interface{}); ok { //Check that the changed map value is also a map[string]interface
				defaultMapRef := defaultMap
				changedMapRef := changedMap.(map[string]interface{})
				for newKey := range defaultMapRef {
					checkKeyBeforeMerging(newKey, defaultMapRef[newKey], changedMapRef[newKey], finalMap[key].(map[string]interface{}))
				}
			}
		default:
			//Check if the value was set, otherwise set it
			if changedMap == nil {
				finalMap[key] = defaultMap
			}
		}
	}
}

func (r *CommonServiceReconciler) addFinalizer(cr *apiv3.CommonService) error {
	klog.Info("Adding Finalizer for the CommonService")
	controllerutil.AddFinalizer(cr, constant.CommonserviceFinalizer)
	if err := r.Update(ctx, cr); err != nil {
		klog.Errorf("Failed to update CommonService with finalizer: %s", err)
		return err
	}
	return nil
}

func (r *CommonServiceReconciler) createUninstallJob() error {
	klog.Info("Create job for uninstall common services")
	name := "uninstall-common-services"
	namespace := "ibm-common-services"
	job := &batchv1.Job{}
	jobKey := types.NamespacedName{Name: name, Namespace: namespace}
	jobImage, err := r.getOperatorImageName()
	if err != nil {
		klog.Errorf("Failed to get common service operator image: %s", err)
		return err
	}
	if err := r.Reader.Get(context.TODO(), jobKey, job); err != nil && errors.IsNotFound(err) {
		if err := r.Client.Create(context.TODO(), newJobObj(name, namespace, jobImage)); err != nil {
			klog.Errorf("Failed to create uninstall job: %s", err)
			return err
		}
		// Job created successfully
		klog.Infof("Job create successfully: %s", name)
		return nil
	}
	return nil
}

func newJobObj(name, namespace, jobImage string) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: func() *int32 { var s int32 = 100; return &s }(),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: "ibm-common-service-operator",
					RestartPolicy:      "Never",
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           jobImage,
							ImagePullPolicy: "Always",
							Command:         []string{"/uninstall.sh"},
						},
					},
				},
			},
		},
	}
	return job
}

func (r *CommonServiceReconciler) getOperatorImageName() (string, error) {
	deploy, err := r.Manager.GetDeployment()
	if err != nil {
		return "", err
	}
	for _, c := range deploy.Spec.Template.Spec.Containers {
		if c.Name == "ibm-common-service-operator" {
			return c.Image, nil
		}
	}
	return "", fmt.Errorf("notfound ibm-common-service-operator image in deployment %s", deploy.GetName())
}

// Check if the request's NamespacedName is equal "ibm-common-services/common-service"
func checkNamespace(key string) bool {
	if key != "ibm-common-services/common-service" {
		klog.Infof("Ignore reconcile when commonservices.operator.ibm.com is NamespacedName '%s', only reconcile for NamespacedName 'ibm-common-services/common-service'", key)
		return false
	}
	return true
}

func filterNamespacePredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Only reconcle when NamespacedName equal "ibm-common-services/common-service"
			key := e.Meta.GetNamespace() + "/" + e.Meta.GetName()
			return checkNamespace(key)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Only reconcle when NamespacedName equal "ibm-common-services/common-service"
			key := e.MetaNew.GetNamespace() + "/" + e.MetaNew.GetName()
			return checkNamespace(key)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			// Only reconcle when NamespacedName equal "ibm-common-services/common-service"
			key := e.Meta.GetNamespace() + "/" + e.Meta.GetName()
			return !e.DeleteStateUnknown && checkNamespace(key)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			// Only reconcle when NamespacedName equal "ibm-common-services/common-service"
			key := e.Meta.GetNamespace() + "/" + e.Meta.GetName()
			return checkNamespace(key)
		},
	}
}

func (r *CommonServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv3.CommonService{}).
		WithEventFilter(filterNamespacePredicate()).
		Complete(r)
}
