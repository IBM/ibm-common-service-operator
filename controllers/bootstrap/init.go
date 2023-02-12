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

package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"text/template"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	"github.com/IBM/ibm-common-service-operator/controllers/deploy"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
)

var (
	placeholder               = "placeholder"
	OdlmNamespacedSubResource = "odlmNamespacedSubscription"
	OdlmClusterSubResource    = "odlmClusterSubscription"
	RegistryCrResources       = "csV3OperandRegistry"
	RegistrySaasCrResources   = "csV3SaasOperandRegistry"
	ConfigCrResources         = "csV3OperandConfig"
	ConfigSaasCrResources     = "csV3SaasOperandConfig"
	CSOperatorVersions        = map[string]string{
		"operand-deployment-lifecycle-manager-app": "1.5.0",
		"ibm-cert-manager-operator":                "3.9.0",
	}
)

var ctx = context.Background()

type Bootstrap struct {
	client.Client
	client.Reader
	Config *rest.Config
	record.EventRecorder
	*deploy.Manager
	SaasEnable           bool
	MultiInstancesEnable bool
	CSOperators          []CSOperator
	CSData               apiv3.CSData
}

type CSOperator struct {
	Name       string
	CRD        string
	RBAC       string
	CR         string
	Deployment string
	Kind       string
	APIVersion string
}

type Resource struct {
	Name    string
	Version string
	Group   string
	Kind    string
	Scope   string
}

// NewBootstrap is the way to create a NewBootstrap struct
func NewBootstrap(mgr manager.Manager) (bs *Bootstrap, err error) {
	cpfsNs := util.GetCPFSNamespace(mgr.GetAPIReader())
	operatorNs, err := util.GetOperatorNamespace()
	if err != nil {
		return
	}
	isOCP, err := isOCP(mgr, cpfsNs)
	if err != nil {
		return
	}

	catalogSourceName, catalogSourceNs := util.GetCatalogSource(constant.IBMCSPackage, operatorNs, mgr.GetAPIReader())
	if catalogSourceName == "" || catalogSourceNs == "" {
		err = fmt.Errorf("failed to get catalogsource")
		return
	}
	approvalMode, err := util.GetApprovalModeinNs(mgr.GetAPIReader(), operatorNs)
	if err != nil {
		return
	}
	csData := apiv3.CSData{
		CPFSNs:            cpfsNs,
		ServicesNs:        util.GetServicesNamespace(mgr.GetAPIReader()),
		OperatorNs:        operatorNs,
		CatalogSourceName: catalogSourceName,
		CatalogSourceNs:   catalogSourceNs,
		ApprovalMode:      approvalMode,
		ZenOperatorImage:  util.GetImage("IBM_ZEN_OPERATOR_IMAGE"),
		IsOCP:             isOCP,
		WatchNamespaces:   util.GetWatchNamespace(),
	}

	bs = &Bootstrap{
		Client:               mgr.GetClient(),
		Reader:               mgr.GetAPIReader(),
		Config:               mgr.GetConfig(),
		EventRecorder:        mgr.GetEventRecorderFor("ibm-common-service-operator"),
		Manager:              deploy.NewDeployManager(mgr),
		SaasEnable:           util.CheckSaas(mgr.GetAPIReader()),
		MultiInstancesEnable: util.CheckMultiInstances(mgr.GetAPIReader()),
		CSData:               csData,
	}

	// Get all the resources from the deployment annotations
	annotations, err := bs.GetAnnotations()
	if err != nil {
		klog.Errorf("failed to get Annotations from csv: %v", err)
	}

	if r, ok := annotations["operatorChannel"]; ok {
		bs.CSData.Channel = r
	}

	if r, ok := annotations["operatorVersion"]; ok {
		bs.CSData.Version = r
	}
	klog.Infof("Single Deployment Status: %v, MultiInstance Deployment status: %v, SaaS Depolyment Status: %v", !bs.MultiInstancesEnable, bs.MultiInstancesEnable, bs.SaasEnable)
	return
}

func isOCP(mgr manager.Manager, ns string) (bool, error) {
	config := &corev1.ConfigMap{}
	if err := mgr.GetClient().Get(context.TODO(), types.NamespacedName{Name: "ibm-cpp-config", Namespace: ns}, config); err != nil && !errors.IsNotFound(err) {
		return false, err
	} else if errors.IsNotFound(err) {
		return true, nil
	} else {
		if config.Data["kubernetes_cluster_type"] == "" || config.Data["kubernetes_cluster_type"] == "ocp" {
			return true, nil
		}
		return false, nil
	}
}

// InitResources initialize resources at the bootstrap of operator
func (b *Bootstrap) InitResources(instance *apiv3.CommonService, forceUpdateODLMCRs bool) error {
	installPlanApproval := instance.Spec.InstallPlanApproval

	if installPlanApproval != "" {
		if installPlanApproval != olmv1alpha1.ApprovalAutomatic && installPlanApproval != olmv1alpha1.ApprovalManual {
			return fmt.Errorf("invalid value for installPlanApproval %v", installPlanApproval)
		}
		b.CSData.ApprovalMode = string(installPlanApproval)
	}

	// Check storageClass
	if err := util.CheckStorageClass(b.Reader); err != nil {
		return err
	}

	// Backward compatible upgrade from version 3.x.x
	if err := b.CreateNsScopeConfigmap(); err != nil {
		klog.Errorf("Failed to create Namespace Scope ConfigMap: %v", err)
		return err
	}

	klog.Info("Installing ODLM Operator")
	if err := b.renderTemplate(constant.ODLMSubscription, b.CSData); err != nil {
		return err
	}

	// create and wait ODLM OperandRegistry and OperandConfig CR resources
	if err := b.waitResourceReady("operator.ibm.com/v1alpha1", "OperandRegistry"); err != nil {
		return err
	}
	if err := b.waitResourceReady("operator.ibm.com/v1alpha1", "OperandConfig"); err != nil {
		return err
	}

	klog.Info("Checking OperandRegistry and OperandConfig deployment status")
	if err := b.ConfigODLMOperandManagedByOperator(ctx); err != nil {
		return err
	}

	klog.Info("Installing/Updating OperandRegistry")
	if installPlanApproval != "" || b.CSData.ApprovalMode == string(olmv1alpha1.ApprovalManual) {
		if err := b.updateApprovalMode(); err != nil {
			return err
		}
	}

	var obj []*unstructured.Unstructured
	var err error
	if b.SaasEnable {
		// OperandRegistry for SaaS deployment
		obj, err = b.GetObjs(constant.CSV3SaasOperandRegistry, b.CSData)
	} else {
		// OperandRegistry for on-prem deployment
		obj, err = b.GetObjs(constant.CSV3OperandRegistry, b.CSData)
	}
	if err != nil {
		klog.Error(err)
		return err
	}

	objInCluster, err := b.GetObject(obj[0])
	if errors.IsNotFound(err) {
		klog.Infof("Creating resource with name: %s, namespace: %s, kind: %s, apiversion: %s\n", obj[0].GetName(), obj[0].GetNamespace(), obj[0].GetKind(), obj[0].GetAPIVersion())
		if err := b.CreateObject(obj[0]); err != nil {
			klog.Error(err)
			return err

		}
	} else if err != nil {
		klog.Error(err)

		return err
	} else {
		klog.Infof("Updating resource with name: %s, namespace: %s, kind: %s, apiversion: %s\n", obj[0].GetName(), obj[0].GetNamespace(), obj[0].GetKind(), obj[0].GetAPIVersion())
		resourceVersion := objInCluster.GetResourceVersion()
		obj[0].SetResourceVersion(resourceVersion)
		v1IsLarger, convertErr := util.CompareVersion(obj[0].GetAnnotations()["version"], objInCluster.GetAnnotations()["version"])
		if convertErr != nil {
			return convertErr
		}
		if v1IsLarger || forceUpdateODLMCRs {
			if err := b.UpdateObject(obj[0]); err != nil {
				klog.Error(err)

				return err
			}
		}
	}

	klog.Info("Installing/Updating OperandConfig")
	if b.SaasEnable {
		// OperandConfig for SaaS deployment
		if err := b.renderTemplate(constant.CSV3SaasOperandConfig, b.CSData, forceUpdateODLMCRs); err != nil {
			return err
		}
	} else {
		// OperandConfig for on-prem deployment
		b.CSData.OnPremMultiEnable = strconv.FormatBool(b.MultiInstancesEnable)
		if err := b.renderTemplate(constant.CSV3OperandConfig, b.CSData, forceUpdateODLMCRs); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bootstrap) CreateNamespace(name string) error {
	nsObj := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	if err := b.Client.Create(ctx, nsObj); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (b *Bootstrap) CheckCsSubscription() error {
	subs, err := b.ListSubscriptions(ctx, b.CSData.OperatorNs, client.ListOptions{Namespace: b.CSData.OperatorNs, LabelSelector: labels.SelectorFromSet(map[string]string{
		"operators.coreos.com/ibm-common-service-operator." + b.CSData.OperatorNs: "",
	})})

	if err != nil {
		return err
	}
	// check all the CS subscrtipions and delete the operator not deployed by ibm-common-service-operator
	for _, sub := range subs.Items {
		if sub.GetName() != "ibm-common-service-operator" {
			if err := b.deleteSubscription(sub.GetName(), sub.GetNamespace()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *Bootstrap) CreateCsCR() error {
	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	cs.SetName("common-service")
	cs.SetNamespace(b.CSData.OperatorNs)

	if len(b.CSData.WatchNamespaces) == 0 {
		// All Namespaces Mode:
		// using `ibm-common-services` ns as ServicesNs if it exists
		// Otherwise, do not create default CR
		// Tolerate if someone manually create the default CR in operator NS
		defaultCRReady := false
		for !defaultCRReady {
			_, err := b.GetObject(cs)
			if errors.IsNotFound(err) {
				ctx := context.Background()
				ns := &corev1.Namespace{}
				if err := b.Reader.Get(ctx, types.NamespacedName{Name: constant.MasterNamespace}, ns); err != nil {
					if errors.IsNotFound(err) {
						klog.Warningf("Not found well-known default namespace %v, please manually create the namespace", constant.MasterNamespace)
						time.Sleep(10 * time.Second)
						continue
					}
					return err
				}
				b.CSData.ServicesNs = constant.MasterNamespace
				return b.renderTemplate(constant.CsCR, b.CSData)
			} else if err != nil {
				return err
			}
			defaultCRReady = true
		}
	} else {
		_, err := b.GetObject(cs)
		if errors.IsNotFound(err) { // Only if it's a fresh install
			// Fresh Intall: No ODLM and NO CR
			return b.renderTemplate(constant.CsCR, b.CSData)
		} else if err != nil {
			return err
		}
	}

	// Restart && Upgrade from 3.5+: Found existing CR
	return nil
}

// func (b *Bootstrap) CreateOperatorGroup(namespace string) error {
// 	existOG := &olmv1.OperatorGroupList{}
// 	if err := b.Reader.List(context.TODO(), existOG, &client.ListOptions{Namespace: namespace}); err != nil {
// 		return err
// 	}
// 	if len(existOG.Items) == 0 {
// 		if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(constant.CsOperatorGroup, placeholder, namespace))); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

func (b *Bootstrap) CreateOrUpdateFromYaml(yamlContent []byte, alwaysUpdate ...bool) error {
	objects, err := util.YamlToObjects(yamlContent)
	if err != nil {
		return err
	}

	var errMsg error

	for _, obj := range objects {
		gvk := obj.GetObjectKind().GroupVersionKind()

		objInCluster, err := b.GetObject(obj)
		if errors.IsNotFound(err) {
			klog.Infof("Creating resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
			if err := b.CreateObject(obj); err != nil {
				errMsg = err
			}
			continue
		} else if err != nil {
			errMsg = err
			continue
		}

		if objInCluster.GetDeletionTimestamp() != nil {
			errMsg = fmt.Errorf("resource %s/%s is being deleted, retry later, kind: %s, apiversion: %s/%s", obj.GetNamespace(), obj.GetName(), gvk.Kind, gvk.Group, gvk.Version)
			continue
		}

		forceUpdate := false
		if len(alwaysUpdate) != 0 {
			forceUpdate = alwaysUpdate[0]
		}
		update := forceUpdate

		// do not compareVersion if the resource is subscription
		if gvk.Kind == "Subscription" {
			sub, err := b.GetSubscription(ctx, obj.GetName(), obj.GetNamespace())
			if err != nil {
				if obj.GetNamespace() == "" {
					klog.Errorf("Failed to get subscription for %s. Namespace not found.", obj.GetName())
				} else {
					klog.Errorf("Failed to get subscription %s/%s", obj.GetNamespace(), obj.GetName())
				}
				return err
			}
			if sub.Object["spec"].(map[string]interface{})["config"] != nil {
				obj.Object["spec"].(map[string]interface{})["config"] = sub.Object["spec"].(map[string]interface{})["config"]
			}
			update = !equality.Semantic.DeepEqual(sub.Object["spec"], obj.Object["spec"])
		} else {
			v1IsLarger, convertErr := util.CompareVersion(obj.GetAnnotations()["version"], objInCluster.GetAnnotations()["version"])
			if convertErr != nil {
				return convertErr
			}
			if v1IsLarger {
				update = true
			}
		}

		if update {
			klog.Infof("Updating resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
			resourceVersion := objInCluster.GetResourceVersion()
			obj.SetResourceVersion(resourceVersion)
			if err := b.UpdateObject(obj); err != nil {
				errMsg = err
			}
		}
	}

	return errMsg
}

// DeleteFromYaml takes [objectTemplate, b.CSData] and delete the object according to the objectTemplate
func (b *Bootstrap) DeleteFromYaml(objectTemplate string, data interface{}) error {
	var buffer bytes.Buffer
	t := template.Must(template.New("newTemplate").Parse(objectTemplate))
	if err := t.Execute(&buffer, data); err != nil {
		return err
	}

	yamlContent := buffer.Bytes()
	objects, err := util.YamlToObjects(yamlContent)
	if err != nil {
		return err
	}

	var errMsg error

	for _, obj := range objects {
		gvk := obj.GetObjectKind().GroupVersionKind()

		_, err := b.GetObject(obj)
		if errors.IsNotFound(err) {
			klog.Infof("Not Found name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n, skipping", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
			continue
		} else if err != nil {
			errMsg = err
			continue
		}

		klog.Infof("Deleting object with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
		if err := b.DeleteObject(obj); err != nil {
			errMsg = err
		}

		// waiting for the object be deleted
		if err := utilwait.PollImmediate(time.Second*10, time.Minute*5, func() (done bool, err error) {
			_, errNotFound := b.GetObject(obj)
			if errors.IsNotFound(errNotFound) {
				return true, nil
			}
			klog.Infof("waiting for object with name: %s, namespace: %s, kind: %s, apiversion: %s/%s to delete\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
			return false, nil
		}); err != nil {
			return err
		}

	}

	return errMsg
}

// GetSubscription returns the subscription instance of "name" from "namespace" namespace
func (b *Bootstrap) GetSubscription(ctx context.Context, name, namespace string) (*unstructured.Unstructured, error) {
	klog.Infof("Fetch Subscription: %v/%v", namespace, name)
	sub := &unstructured.Unstructured{}
	sub.SetGroupVersionKind(olmv1alpha1.SchemeGroupVersion.WithKind("subscription"))
	subKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	if err := b.Client.Get(ctx, subKey, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// GetSubscription returns the subscription instances from a  namespace
func (b *Bootstrap) ListSubscriptions(ctx context.Context, namespace string, listOptions client.ListOptions) (*unstructured.UnstructuredList, error) {
	klog.Infof("List Subscriptions in namespace %v", namespace)
	subs := &unstructured.UnstructuredList{}
	subs.SetGroupVersionKind(olmv1alpha1.SchemeGroupVersion.WithKind("SubscriptionList"))
	if err := b.Client.List(ctx, subs, &listOptions); err != nil {
		return nil, err
	}
	return subs, nil
}

// GetOperandRegistry returns the OperandRegistry instance of "name" from "namespace" namespace
func (b *Bootstrap) GetOperandRegistry(ctx context.Context, name, namespace string) (*odlm.OperandRegistry, error) {
	klog.V(2).Infof("Fetch OperandRegistry: %v/%v", namespace, name)
	opreg := &odlm.OperandRegistry{}
	opregKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	if err := b.Reader.Get(ctx, opregKey, opreg); err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	return opreg, nil
}

// GetOperandConfig returns the OperandConfig instance of "name" from "namespace" namespace
func (b *Bootstrap) GetOperandConfig(ctx context.Context, name, namespace string) (*odlm.OperandConfig, error) {
	klog.V(2).Infof("Fetch OperandConfig: %v/%v", namespace, name)
	opconfig := &odlm.OperandConfig{}
	opconfigKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	if err := b.Reader.Get(ctx, opconfigKey, opconfig); err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	return opconfig, nil
}

// ListOperandRegistry returns the OperandRegistry instance with "options"
func (b *Bootstrap) ListOperandRegistry(ctx context.Context, opts ...client.ListOption) *odlm.OperandRegistryList {
	opregList := &odlm.OperandRegistryList{}
	if err := b.Client.List(ctx, opregList, opts...); err != nil {
		klog.Errorf("failed to List OperandRegistry: %v", err)
		return nil
	}

	return opregList
}

// ListOperandConfig returns the OperandConfig instance with "options"
func (b *Bootstrap) ListOperandConfig(ctx context.Context, opts ...client.ListOption) *odlm.OperandConfigList {
	opconfigList := &odlm.OperandConfigList{}
	if err := b.Client.List(ctx, opconfigList, opts...); err != nil {
		klog.Errorf("failed to List OperandConfig: %v", err)
		return nil
	}

	return opconfigList
}

func (b *Bootstrap) CheckOperatorCatalog(ns string) error {

	err := utilwait.PollImmediate(time.Second*10, time.Minute*3, func() (done bool, err error) {
		subList := &olmv1alpha1.SubscriptionList{}

		if err := b.Reader.List(context.TODO(), subList, &client.ListOptions{Namespace: ns}); err != nil {
			return false, err
		}

		var csSub []olmv1alpha1.Subscription
		for _, sub := range subList.Items {
			if sub.Spec.Package == constant.IBMCSPackage {
				csSub = append(csSub, sub)
			}
		}

		if len(csSub) != 1 {
			klog.Errorf("Fail to find ibm-common-service-operator subscription in the namespace %s", ns)
			return false, nil
		}

		if csSub[0].Spec.CatalogSource != b.CSData.CatalogSourceName || subList.Items[0].Spec.CatalogSourceNamespace != b.CSData.CatalogSourceNs {
			csSub[0].Spec.CatalogSource = b.CSData.CatalogSourceName
			csSub[0].Spec.CatalogSourceNamespace = b.CSData.CatalogSourceNs
			if err := b.Client.Update(context.TODO(), &csSub[0]); err != nil {
				return false, err
			}
		}
		return true, nil

	})

	return err
}

func (b *Bootstrap) waitResourceReady(apiGroupVersion, kind string) error {
	dc := discovery.NewDiscoveryClientForConfigOrDie(b.Config)
	if err := utilwait.PollImmediate(time.Second*10, time.Minute*5, func() (done bool, err error) {
		exist, err := b.ResourceExists(dc, apiGroupVersion, kind)
		if err != nil {
			return exist, err
		}
		if !exist {
			klog.Infof("waiting for resource ready with kind: %s, apiGroupVersion: %s", kind, apiGroupVersion)
		}
		return exist, nil
	}); err != nil {
		return err
	}
	return nil
}

// ResourceExists returns true if the given resource kind exists
// in the given api groupversion
func (b *Bootstrap) ResourceExists(dc discovery.DiscoveryInterface, apiGroupVersion, kind string) (bool, error) {
	_, apiLists, err := dc.ServerGroupsAndResources()
	if err != nil {
		return false, err
	}
	for _, apiList := range apiLists {
		if apiList.GroupVersion == apiGroupVersion {
			for _, r := range apiList.APIResources {
				if r.Kind == kind {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// func (b *Bootstrap) createNsSubscription(manualManagement bool) error {
// 	resourceName := constant.NSSubscription
// 	subNameToRemove := constant.NsRestrictedSubName
// 	if manualManagement {
// 		resourceName = constant.NSRestrictedSubscription
// 		subNameToRemove = constant.NsSubName
// 	}

// 	if err := b.deleteSubscription(subNameToRemove, b.CSData.CPFSNs); err != nil {
// 		return err
// 	}

// 	if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
// 		return err
// 	}

// 	return nil
// }

// CreateNsScopeConfigmap creates nss configmap for operators
func (b *Bootstrap) CreateNsScopeConfigmap() error {
	cmRes := constant.NamespaceScopeConfigMap
	if err := b.renderTemplate(cmRes, b.CSData, false); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) deleteSubscription(name, namespace string) error {
	key := types.NamespacedName{Name: name, Namespace: namespace}
	sub := &olmv1alpha1.Subscription{}
	if err := b.Reader.Get(context.TODO(), key, sub); err != nil {
		if errors.IsNotFound(err) {
			klog.V(3).Infof("NotFound subscription %s/%s", namespace, name)
		} else {
			klog.Errorf("Failed to get subscription %s/%s", namespace, name)
		}
		return client.IgnoreNotFound(err)
	}

	klog.Infof("Deleting subscription %s/%s", namespace, name)

	// Delete csv
	csvName := sub.Status.InstalledCSV
	if csvName != "" {
		csv := &olmv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      csvName,
				Namespace: namespace,
			},
		}
		if err := b.Client.Delete(context.TODO(), csv); err != nil && !errors.IsNotFound(err) {
			klog.Errorf("Failed to delete Cluster Service Version: %v", err)
			return err
		}
	}

	// Delete subscription
	if err := b.Client.Delete(context.TODO(), sub); err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Failed to delete subscription: %s", err)
		return err
	}

	return nil
}

func (b *Bootstrap) renderTemplate(objectTemplate string, data interface{}, alwaysUpdate ...bool) error {
	var buffer bytes.Buffer
	t := template.Must(template.New("newTemplate").Parse(objectTemplate))
	if err := t.Execute(&buffer, data); err != nil {
		return err
	}

	forceUpdate := false
	if len(alwaysUpdate) != 0 {
		forceUpdate = alwaysUpdate[0]
	}

	if err := b.CreateOrUpdateFromYaml(buffer.Bytes(), forceUpdate); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) GetObjs(objectTemplate string, data interface{}, alwaysUpdate ...bool) ([]*unstructured.Unstructured, error) {
	var buffer bytes.Buffer
	t := template.Must(template.New("newTemplate").Parse(objectTemplate))
	if err := t.Execute(&buffer, data); err != nil {
		return nil, err
	}

	objects, err := util.YamlToObjects(buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return objects, nil
}

// update approval mode for the common service operator
// use label to find the subscription
// need this function because common service operator is not in operandRegistry
func (b *Bootstrap) UpdateCsOpApproval() error {
	var commonserviceNS string
	operatorNs, err := util.GetOperatorNamespace()
	if err != nil {
		klog.Errorf("Getting operator namespace failed: %v", err)
		return err
	}

	if operatorNs == constant.ClusterOperatorNamespace {
		commonserviceNS = constant.ClusterOperatorNamespace
	} else {
		commonserviceNS = b.CSData.OperatorNs
	}

	subList := &olmv1alpha1.SubscriptionList{}
	opts := []client.ListOption{
		client.InNamespace(commonserviceNS),
		client.MatchingLabels(
			map[string]string{"operators.coreos.com/ibm-common-service-operator." + commonserviceNS: ""}),
	}

	if err := b.Reader.List(ctx, subList, opts...); err != nil {
		return err
	}

	if len(subList.Items) == 0 {
		return fmt.Errorf("not found ibm-common-service-operator subscription in namespace: %v or %v", b.CSData.OperatorNs, constant.ClusterOperatorNamespace)
	}

	if len(subList.Items) > 1 {
		return fmt.Errorf("found more than one ibm-common-service-operator subscription in namespace: %v or %v, skip this", b.CSData.OperatorNs, constant.ClusterOperatorNamespace)
	}

	for _, sub := range subList.Items {
		if b.CSData.ApprovalMode == string(olmv1alpha1.ApprovalManual) && sub.Spec.InstallPlanApproval != olmv1alpha1.ApprovalManual {
			sub.Spec.InstallPlanApproval = olmv1alpha1.ApprovalManual
			if err := b.Client.Update(ctx, &sub); err != nil {
				return err
			}
			podList := &corev1.PodList{}
			opts := []client.ListOption{
				client.InNamespace(commonserviceNS),
				client.MatchingLabels(map[string]string{"name": "ibm-common-service-operator"}),
			}
			if err := b.Reader.List(ctx, podList, opts...); err != nil {
				return err
			}
			for _, pod := range podList.Items {
				if err := b.Client.Delete(ctx, &pod); err != nil {
					return err
				}
			}

		}
	}
	return nil
}

func (b *Bootstrap) updateApprovalMode() error {
	opreg := &odlm.OperandRegistry{}
	opregKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: b.CSData.ServicesNs,
	}

	err := b.Reader.Get(ctx, opregKey, opreg)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		klog.Errorf("failed to get OperandRegistry %s: %v", opregKey.String(), err)
		return err

	}

	for i := range opreg.Spec.Operators {
		opreg.Spec.Operators[i].InstallPlanApproval = olmv1alpha1.Approval(b.CSData.ApprovalMode)
	}
	if err := b.Update(ctx, opreg); err != nil {
		klog.Errorf("failed to update OperandRegistry %s: %v", opregKey.String(), err)
		return err
	}

	if err = b.UpdateCsOpApproval(); err != nil {
		klog.Errorf("Failed to update %s subscription: %v", constant.IBMCSPackage, err)
		return err
	}

	return nil
}

// WaitResourceReady returns true only when the specific resource CRD is created and wait for infinite time
func (b *Bootstrap) WaitResourceReady(apiGroupVersion string, kind string) error {
	dc := discovery.NewDiscoveryClientForConfigOrDie(b.Config)
	if err := utilwait.PollImmediateInfinite(time.Second*10, func() (done bool, err error) {
		exist, err := b.ResourceExists(dc, apiGroupVersion, kind)
		if err != nil {
			return exist, err
		}
		if !exist {
			klog.V(2).Infof("waiting for resource ready with kind: %s, apiGroupVersion: %s", kind, apiGroupVersion)
		}
		return exist, nil
	}); err != nil {
		return err
	}
	return nil
}

// deployResource deploys the given resource CR
func (b *Bootstrap) DeployResource(cr, placeholder string) bool {
	if err := utilwait.PollImmediateInfinite(time.Second*10, func() (done bool, err error) {
		err = b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cr, placeholder, b.CSData.ServicesNs)))
		if err != nil {
			return false, err
		}
		return true, nil
	}); err != nil {
		klog.Errorf("Failed to create Certmanager resource: %v, retry in 10 seconds", err)
		return false
	}
	return true
}

func CheckClusterType(mgr manager.Manager, ns string) (bool, error) {
	var isOCP bool
	dc := discovery.NewDiscoveryClientForConfigOrDie(mgr.GetConfig())
	_, apiLists, err := dc.ServerGroupsAndResources()
	if err != nil {
		return false, err
	}
	for _, apiList := range apiLists {
		if apiList.GroupVersion == "machineconfiguration.openshift.io/v1" {
			for _, r := range apiList.APIResources {
				if r.Kind == "MachineConfig" {
					isOCP = true
				}
			}
		}
	}

	config := &corev1.ConfigMap{}
	if err := mgr.GetClient().Get(context.TODO(), types.NamespacedName{Name: "ibm-cpp-config", Namespace: ns}, config); err != nil && !errors.IsNotFound(err) {
		return false, err
	} else if errors.IsNotFound(err) {
		if isOCP {
			return true, nil
		}
		klog.Errorf("Configmap %s/ibm-cpp-config is required", ns)
		return false, nil
	} else {
		if config.Data["kubernetes_cluster_type"] == "" {
			return true, nil
		}
		if config.Data["kubernetes_cluster_type"] == "ocp" && !isOCP || config.Data["kubernetes_cluster_type"] != "ocp" && isOCP {
			ocpCluster := "a non-OCP"
			if isOCP {
				ocpCluster = "an OCP"
			}
			klog.Errorf("cluster type isn't correct, kubernetes_cluster_type in configmap %s/ibm-cpp-config is %s, but the cluster is %s environment", ns, config.Data["kubernetes_cluster_type"], ocpCluster)
			return false, nil
		}

		klog.Info("cluster type is correct")
		return true, nil
	}
}

func (b *Bootstrap) DeployCertManagerCR() error {
	klog.V(2).Info("Fetch all the CommonService instances")
	csObjectList := &apiv3.CommonServiceList{}
	if err := b.Client.List(ctx, csObjectList); err != nil {
		return err
	}
	csList, err := util.ObjectListToNewUnstructuredList(csObjectList)
	if err != nil {
		return err
	}
	deployRootCert := true
	var crWithBYOCert string
	for _, cs := range csList.Items {
		if cs.GetDeletionTimestamp() != nil {
			continue
		}
		if cs.Object["spec"].(map[string]interface{})["BYOCACertificate"] == true {
			deployRootCert = false
			crWithBYOCert = cs.GetNamespace() + "/" + cs.GetName()
			break
		}
	}
	klog.Info("Deploying Cert Manager CRs")
	for _, kind := range constant.CertManagerKinds {
		// wait for v1 crd ready
		if err := b.waitResourceReady(constant.CertManagerAPIGroupVersionV1, kind); err != nil {
			klog.Errorf("Failed to wait for resource ready with kind: %s, apiGroupVersion: %s", kind, constant.CertManagerAPIGroupVersionV1)
		}
	}
	// will use v1 cert instead of v1alpha cert
	// delete v1alpha1 cert if it exist
	var resourceList = []*Resource{
		{
			Name:    "cs-ca-issuer",
			Version: "v1alpha1",
			Group:   "certmanager.k8s.io",
			Kind:    "issuer",
			Scope:   "namespaceScope",
		},
		{
			Name:    "cs-ss-issuer",
			Version: "v1alpha1",
			Group:   "certmanager.k8s.io",
			Kind:    "issuer",
			Scope:   "namespaceScope",
		},
		{
			Name:    "cs-ca-certificate",
			Version: "v1alpha1",
			Group:   "certmanager.k8s.io",
			Kind:    "certificate",
			Scope:   "namespaceScope",
		},
	}

	for _, resource := range resourceList {
		if err := b.Cleanup(b.CSData.ServicesNs, resource); err != nil {
			return err
		}
	}

	for _, cr := range constant.CertManagerIssuers {
		if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cr, placeholder, b.CSData.ServicesNs))); err != nil {
			return err
		}
	}
	if deployRootCert {
		for _, cr := range constant.CertManagerCerts {
			if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cr, placeholder, b.CSData.ServicesNs))); err != nil {
				return err
			}
		}
	} else {
		klog.Infof("Skipped deploying %s, BYOCertififcate feature is enabled in %s", constant.CSCACertificate, crWithBYOCert)
	}

	return nil
}

func (b *Bootstrap) Cleanup(operatorNs string, resource *Resource) error {
	// check if crd exist
	dc := discovery.NewDiscoveryClientForConfigOrDie(b.Config)
	APIGroupVersion := resource.Group + "/" + resource.Version
	exist, err := b.ResourceExists(dc, APIGroupVersion, resource.Kind)
	if err != nil {
		klog.Errorf("Failed to check resource with kind: %s, apiGroupVersion: %s", resource.Kind, APIGroupVersion)
	}
	if !exist {
		return nil
	}

	deprecated := &unstructured.Unstructured{}
	deprecated.SetGroupVersionKind(schema.GroupVersionKind{Group: resource.Group, Version: resource.Version, Kind: resource.Kind})
	deprecated.SetName(resource.Name)
	if resource.Scope == "namespaceScope" {
		deprecated.SetNamespace(operatorNs)
	}
	if err := b.Client.Delete(context.TODO(), deprecated); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	klog.Infof("Deleting resource %s/%s", operatorNs, resource.Name)
	return nil
}

func (b *Bootstrap) CheckDeployStatus(ctx context.Context) (operatorDeployed bool, servicesDeployed bool, err error) {
	opreg, err := b.GetOperandRegistry(ctx, "common-service", b.CSData.ServicesNs)
	if err != nil {
		return true, true, err
	} else if opreg != nil && opreg.Status.Phase == odlm.RegistryRunning {
		operatorDeployed = true
	}

	opconfig, err := b.GetOperandConfig(ctx, "common-service", b.CSData.ServicesNs)
	if err != nil {
		return true, true, err
	} else if opreg != nil && opconfig.Status.Phase == odlm.ServiceRunning {
		servicesDeployed = true
	}
	return
}

// ConfigODLMOperandManagedByOperator gets all OperandRegistry and OperandConfig which are managed by CS operator
// To confirm that the deployment of CRs are in the correct ServicesNamespace
func (b *Bootstrap) ConfigODLMOperandManagedByOperator(ctx context.Context) error {
	opts := []client.ListOption{
		client.MatchingLabels(
			map[string]string{constant.CsManagedLabel: "true"}),
	}
	opregList := b.ListOperandRegistry(ctx, opts...)
	if opregList != nil {
		for _, opreg := range opregList.Items {
			if opreg.Namespace != b.CSData.ServicesNs && opreg.Status.Phase == odlm.RegistryReady {
				if err := b.Client.Delete(ctx, &opreg); err != nil {
					klog.Errorf("Failed to delete idle OperandRegistry %s/%s which is managed by CS operator, but not in ServicesNamespace %s", opreg.GetNamespace(), opreg.GetName(), b.CSData.ServicesNs)
					return err
				}
				klog.Infof("Delete idle OperandRegistry %s/%s which is managed by CS operator, but not in ServicesNamespace %s", opreg.GetNamespace(), opreg.GetName(), b.CSData.ServicesNs)
			} else if opreg.Namespace != b.CSData.ServicesNs && opreg.Status.Phase != odlm.RegistryReady {
				klog.Warningf("Skipped deleting OperandRegistry %s/%s, its status is %s", opreg.GetNamespace(), opreg.GetName(), opreg.Status.Phase)
				return fmt.Errorf("please configure the correct ServicesNamespace or uninstall the existing foundational services to configure the correct OperandRegistry")
			}
		}
	}

	opconfigList := b.ListOperandConfig(ctx, opts...)
	if opconfigList != nil {
		for _, opconfig := range opconfigList.Items {
			if opconfig.Namespace != b.CSData.ServicesNs && opconfig.Status.Phase == odlm.ServiceInit {
				if err := b.Client.Delete(ctx, &opconfig); err != nil {
					klog.Errorf("Failed to delete idle OperandConfig %s/%s which is managed by CS operator, but not in ServicesNamespace %s", opconfig.GetNamespace(), opconfig.GetName(), b.CSData.ServicesNs)
					return err
				}
				klog.Infof("Delete idle OperandConfig %s/%s which is managed by CS operator, but not in ServicesNamespace %s", opconfig.GetNamespace(), opconfig.GetName(), b.CSData.ServicesNs)
			} else if opconfig.Namespace != b.CSData.ServicesNs && opconfig.Status.Phase != odlm.ServiceInit {
				klog.Warningf("Skipped deleting OperandConfig %s/%s, its status is %s", opconfig.GetNamespace(), opconfig.GetName(), opconfig.Status.Phase)
				return fmt.Errorf("please configure the correct ServicesNamespace or uninstall the existing foundational services to configure the correct OperandConfig")
			}
		}
	}

	return nil
}
