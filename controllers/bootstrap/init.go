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

package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
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
	CsSubResource             = "csOperatorSubscription"
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
	*deploy.Manager
	SaasEnable           bool
	MultiInstancesEnable bool
	CSOperators          []CSOperator
	CSData               CSData
}
type CSData struct {
	Channel            string
	Version            string
	MasterNs           string
	ControlNs          string
	CatalogSourceName  string
	CatalogSourceNs    string
	IsolatedModeEnable string
	ApprovalMode       string
	OnPremMultiEnable  string
	CrossplaneProvider string
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

// NewBootstrap is the way to create a NewBootstrap struct
func NewBootstrap(mgr manager.Manager) (bs *Bootstrap, err error) {
	csWebhookDeployment := constant.CsWebhookOperator
	csSecretShareDeployment := constant.CsSecretshareOperator
	if _, err := util.GetCmOfMapCs(mgr.GetAPIReader()); err == nil {
		csWebhookDeployment = constant.CsWebhookOperatorEnableOpreqWebhook
	}
	var csOperators = []CSOperator{
		{"Webhook Operator", constant.WebhookCRD, constant.WebhookRBAC, constant.WebhookCR, csWebhookDeployment, constant.WebhookKind, constant.WebhookAPIVersion},
		{"Secretshare Operator", constant.SecretshareCRD, constant.SecretshareRBAC, constant.SecretshareCR, csSecretShareDeployment, constant.SecretshareKind, constant.SecretshareAPIVersion},
	}
	masterNs := util.GetMasterNs(mgr.GetAPIReader())
	operatorNs, err := util.GetOperatorNamespace()
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
	csData := CSData{
		MasterNs:          masterNs,
		ControlNs:         util.GetControlNs(mgr.GetAPIReader()),
		CatalogSourceName: catalogSourceName,
		CatalogSourceNs:   catalogSourceNs,
		ApprovalMode:      approvalMode,
	}

	bs = &Bootstrap{
		Client:               mgr.GetClient(),
		Reader:               mgr.GetAPIReader(),
		Config:               mgr.GetConfig(),
		Manager:              deploy.NewDeployManager(mgr),
		SaasEnable:           util.CheckSaas(mgr.GetAPIReader()),
		MultiInstancesEnable: util.CheckMultiInstances(mgr.GetAPIReader()),
		CSOperators:          csOperators,
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
	return
}

// CrossplaneCloudOperator install crossplane & cloud operator when bedrockshim is true
func (b *Bootstrap) CrossplaneCloudOperator(instance *apiv3.CommonService) error {

	// Install Crossplane Operator & Cloud Operator
	bedrockshim := false
	if instance.Spec.Features != nil {
		if instance.Spec.Features.Bedrockshim != nil {
			bedrockshim = instance.Spec.Features.Bedrockshim.Enabled
		}
	}

	if bedrockshim {
		b.CSData.CrossplaneProvider = "odlm"

		if b.SaasEnable {
			b.CSData.CrossplaneProvider = "ibmcloud"
			if err := b.installCloudOperator(); err != nil {
				return err
			}

		}

		if err := b.installCrossplaneOperator(); err != nil {
			return err
		}
	}

	return nil
}

// InitResources initialize resources at the bootstrap of operator
func (b *Bootstrap) InitResources(instance *apiv3.CommonService) error {
	installPlanApproval := instance.Spec.InstallPlanApproval
	manualManagement := instance.Spec.ManualManagement

	if installPlanApproval != "" {
		if installPlanApproval != olmv1alpha1.ApprovalAutomatic && installPlanApproval != olmv1alpha1.ApprovalManual {
			return fmt.Errorf("invalid value for installPlanApproval %v", installPlanApproval)
		}
		b.CSData.ApprovalMode = string(installPlanApproval)
	}

	// Check Saas or Multi instances Deployment
	if b.MultiInstancesEnable {
		klog.Infof("Creating IBM Common Services control namespace: %s", b.CSData.ControlNs)
		if err := b.CreateNamespace(b.CSData.ControlNs); err != nil {
			klog.Errorf("Failed to create control namespace: %v", err)
			return err
		}
	} else {
		b.CSData.ControlNs = b.CSData.MasterNs
	}

	operatorNs, err := util.GetOperatorNamespace()
	if err != nil {
		klog.Errorf("Getting operator namespace failed: %v", err)
		return err
	}

	// Grant cluster-admin to namespace scope operator
	if operatorNs == constant.ClusterOperatorNamespace {
		klog.Info("Creating cluster-admin permission RBAC")
		if err := b.renderTemplate(constant.ClusterAdminRBAC, b.CSData); err != nil {
			return err
		}
	}

	// Check storageClass
	if err := util.CheckStorageClass(b.Reader); err != nil {
		return err
	}

	// Install Namespace Scope Operator
	if err := b.installNssOperator(manualManagement); err != nil {
		return err
	}

	// Install CS Operators
	for _, operator := range b.CSOperators {
		if b.SaasEnable && operator.Name == "Secretshare Operator" {
			continue
		}
		klog.Infof("Installing %s", operator.Name)
		// Create Operator CRD
		if err := b.CreateOrUpdateFromYaml([]byte(operator.CRD)); err != nil {
			return err
		}
		// Create Operator RBAC
		if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(operator.RBAC, placeholder, b.CSData.ControlNs))); err != nil {
			return err
		}
		// Create Operator Deployment
		if err := b.CreateOrUpdateFromYaml([]byte(util.ReplaceImages(util.Namespacelize(operator.Deployment, placeholder, b.CSData.ControlNs)))); err != nil {
			return err
		}
		// Wait for CRD ready
		if err := b.waitResourceReady(operator.APIVersion, operator.Kind); err != nil {
			return err
		}
		// Create Operator CR
		if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(operator.CR, placeholder, b.CSData.ControlNs))); err != nil {
			return err
		}
	}

	// Create extra RBAC for ibmcloud-cluster-ca-cert and ibmcloud-cluster-info in kube-public
	klog.Info("Creating RBAC for ibmcloud-cluster-info & ibmcloud-cluster-ca-cert in kube-public")
	if err := b.CreateOrUpdateFromYaml([]byte(constant.ExtraRBAC)); err != nil {
		return err
	}

	// Install ODLM Operator
	if err := b.installODLM(operatorNs); err != nil {
		return err
	}

	// create and wait ODLM OperandRegistry and OperandConfig CR resources
	if err := b.waitResourceReady("operator.ibm.com/v1alpha1", "OperandRegistry"); err != nil {
		return err
	}
	if err := b.waitResourceReady("operator.ibm.com/v1alpha1", "OperandConfig"); err != nil {
		return err
	}

	if err := b.waitOperatorReady("operand-deployment-lifecycle-manager-app", b.CSData.MasterNs); err != nil {
		return err
	}

	klog.Info("Installing/Updating OperandRegistry")
	if installPlanApproval != "" || b.CSData.ApprovalMode == string(olmv1alpha1.ApprovalManual) {
		if err := b.updateApprovalMode(); err != nil {
			return err
		}
	}
	if b.SaasEnable {
		// OperandRegistry for SaaS deployment
		obj, err := b.GetObjs(constant.CSV3OperandRegistry, b.CSData)
		if err != nil {
			return err
		}
		for i := range obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{}) {
			if obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{})[i].(map[string]interface{})["sourceName"] != nil {
				continue
			}
			catalogsource, catalogsourceNs := util.GetCatalogSource(obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{})[i].(map[string]interface{})["packageName"].(string), b.CSData.MasterNs, b.Reader)
			if catalogsource != "" || catalogsourceNs != "" {
				obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{})[i].(map[string]interface{})["sourceName"], obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{})[i].(map[string]interface{})["sourceNamespace"] = catalogsource, catalogsourceNs
			}
		}
		objInCluster, err := b.GetObject(obj[0])
		if errors.IsNotFound(err) {
			klog.Infof("Creating resource with name: %s, namespace: %s, kind: %s, apiversion: %s\n", obj[0].GetName(), obj[0].GetNamespace(), obj[0].GetKind(), obj[0].GetAPIVersion())
			if err := b.CreateObject(obj[0]); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			klog.Infof("Updating resource with name: %s, namespace: %s, kind: %s, apiversion: %s\n", obj[0].GetName(), obj[0].GetNamespace(), obj[0].GetKind(), obj[0].GetAPIVersion())
			resourceVersion := objInCluster.GetResourceVersion()
			obj[0].SetResourceVersion(resourceVersion)
			if util.CompareVersion(obj[0].GetAnnotations()["version"], objInCluster.GetAnnotations()["version"]) {
				if err := b.UpdateObject(obj[0]); err != nil {
					return err
				}
			}
		}
	} else {
		// OperandRegistry for on-prem deployment
		obj, err := b.GetObjs(constant.CSV3OperandRegistry, b.CSData)
		if err != nil {
			klog.Error(err)
			return err
		}
		for i := range obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{}) {
			if obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{})[i].(map[string]interface{})["sourceName"] != nil {
				continue
			}
			catalogsource, catalogsourceNs := util.GetCatalogSource(obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{})[i].(map[string]interface{})["packageName"].(string), b.CSData.MasterNs, b.Reader)
			if catalogsource != "" || catalogsourceNs != "" {
				obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{})[i].(map[string]interface{})["sourceName"], obj[0].Object["spec"].(map[string]interface{})["operators"].([]interface{})[i].(map[string]interface{})["sourceNamespace"] = catalogsource, catalogsourceNs
			}
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
			if util.CompareVersion(obj[0].GetAnnotations()["version"], objInCluster.GetAnnotations()["version"]) {
				if err := b.UpdateObject(obj[0]); err != nil {
					klog.Error(err)

					return err
				}
			}
		}
	}

	if err := b.waitALLOperatorReady(b.CSData.MasterNs); err != nil {
		return err
	}

	klog.Info("Installing/Updating OperandConfig")
	if b.SaasEnable {
		// OperandConfig for SaaS deployment
		if err := b.renderTemplate(constant.CSV3SaasOperandConfig, b.CSData); err != nil {
			return err
		}
	} else {
		// OperandConfig for on-prem deployment
		b.CSData.OnPremMultiEnable = strconv.FormatBool(b.MultiInstancesEnable)
		if err := b.renderTemplate(constant.CSV3OperandConfig, b.CSData); err != nil {
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

func (b *Bootstrap) CreateCsSubscription() error {
	// Get all the resources from the deployment annotations
	if err := b.renderTemplate(constant.CSSubscription, b.CSData); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) CreateCsCR() error {
	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	cs.SetName("common-service")
	cs.SetNamespace(b.CSData.MasterNs)
	_, err := b.GetObject(cs)
	if errors.IsNotFound(err) { // Only if it's a fresh install or upgrade from 3.4
		odlm := util.NewUnstructured("operators.coreos.com", "Subscription", "v1alpha1")
		odlm.SetName("operand-deployment-lifecycle-manager-app")
		odlm.SetNamespace(constant.ClusterOperatorNamespace)
		_, err := b.GetObject(odlm)
		if errors.IsNotFound(err) {
			// Fresh Intall: No ODLM and NO CR
			return b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(constant.CsCR, placeholder, b.CSData.MasterNs)))
		} else if err != nil {
			return err
		}
		// Upgrade from 3.4.x: Have ODLM in openshift-operators and NO CR
		return b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(constant.CsNoSizeCR, placeholder, b.CSData.MasterNs)))
	} else if err != nil {
		return err
	}

	// Restart && Upgrade from 3.5+: Found existing CR
	return nil
}

func (b *Bootstrap) CreateOperatorGroup() error {
	existOG := &olmv1.OperatorGroupList{}
	if err := b.Reader.List(context.TODO(), existOG, &client.ListOptions{Namespace: b.CSData.MasterNs}); err != nil {
		return err
	}
	if len(existOG.Items) == 0 {
		if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(constant.CsOperatorGroup, placeholder, b.CSData.MasterNs))); err != nil {
			return err
		}
	}
	return nil
}

// func (b *Bootstrap) createOrUpdateResource(annotations map[string]string, resName string, resNs string) error {
// 	if r, ok := annotations[resName]; ok {
// 		if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(r, placeholder, resNs))); err != nil {
// 			return err
// 		}
// 	} else {
// 		klog.Warningf("No resource %s found in annotations", resName)
// 	}
// 	return nil
// }

// func (b *Bootstrap) createOrUpdateResources(annotations map[string]string, resNames []string) error {
// 	for _, res := range resNames {
// 		if r, ok := annotations[res]; ok {
// 			if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(r, placeholder, b.MasterNamespace))); err != nil {
// 				return err
// 			}
// 		} else {
// 			klog.Warningf("no resource %s found in annotations", res)
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

		forceUpdate := false
		if len(alwaysUpdate) != 0 {
			forceUpdate = alwaysUpdate[0]
		}
		update := forceUpdate

		// do not compareVersion if the resource is subscription
		if gvk.Kind == "Subscription" {
			sub := b.GetSubscription(ctx, obj.GetName(), b.CSData.MasterNs)
			update = !equality.Semantic.DeepEqual(sub.Object["spec"], obj.Object["spec"])
		} else if util.CompareVersion(obj.GetAnnotations()["version"], objInCluster.GetAnnotations()["version"]) {
			update = true
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

// GetSubscription returns the subscription instance of "name" from "namespace" namespace
func (b *Bootstrap) GetSubscription(ctx context.Context, name, namespace string) *unstructured.Unstructured {
	klog.Infof("Fetch Subscription: %v/%v", namespace, name)
	sub := &unstructured.Unstructured{}
	sub.SetGroupVersionKind(olmv1alpha1.SchemeGroupVersion.WithKind("subscription"))
	subKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	if err := b.Client.Get(ctx, subKey, sub); err != nil {
		return nil
	}

	return sub
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

func (b *Bootstrap) installNssOperator(manualManagement bool) error {
	// Install Namespace Scope Operator
	klog.Info("Creating namespace-scope configmap")
	// Backward compatible upgrade from version 3.4.x
	if err := b.CreateNsScopeConfigmap(); err != nil {
		klog.Errorf("Failed to create Namespace Scope ConfigMap: %v", err)
		return err
	}

	klog.Info("Creating Namespace Scope Operator subscription")
	if err := b.createNsSubscription(manualManagement); err != nil {
		klog.Errorf("Failed to create Namespace Scope Operator subscription: %v", err)
		return err
	}

	if err := b.waitResourceReady("operator.ibm.com/v1", "NamespaceScope"); err != nil {
		return err
	}

	// Create General NSS CRs
	if err := b.renderTemplate(constant.NamespaceScopeCR, b.CSData); err != nil {
		return err
	}
	// Create NSS CRs managedby ODLM for Single CS instance case
	if !b.MultiInstancesEnable {
		if err := b.renderTemplate(constant.NamespaceScopeCRManagedbyODLM, b.CSData); err != nil {
			return err
		}
	}

	cm, err := util.GetCmOfMapCs(b.Reader)
	if err == nil {
		err := util.UpdateNSList(b.Reader, b.Client, cm, "common-service", b.CSData.MasterNs, false)
		if err != nil {
			return err
		}
	} else if !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (b *Bootstrap) installCrossplaneOperator() error {
	klog.Info("Creating Crossplane Operator subscription")
	if err := b.createCrossplaneSubscription(); err != nil {
		klog.Errorf("Failed to create or update Crossplane Operator subscription: %v", err)
		return err
	}

	if err := b.waitResourceReady("pkg.crossplane.io/v1", "Configuration"); err != nil {
		return err
	}

	if err := b.waitResourceReady("pkg.crossplane.io/v1alpha1", "Lock"); err != nil {
		return err
	}

	klog.Info("Creating Crossplane Configuration")
	if err := b.createCrossplaneConfiguration(); err != nil {
		klog.Errorf("Failed to create or update Crossplane Configuration: %v", err)
		return err
	}

	klog.Info("Creating Crossplane Lock")
	if err := b.createCrossplaneLock(); err != nil {
		klog.Errorf("Failed to create or update Crossplane Lock: %v", err)
		return err
	}

	return nil
}

func (b *Bootstrap) installCloudOperator() error {
	klog.Info("Creating IBM Cloud Operator subscription")
	if err := b.createCloudSubscription(); err != nil {
		klog.Errorf("Failed to create or update IBM Cloud Operator subscription: %v", err)
		return err
	}
	return nil
}

func (b *Bootstrap) installODLM(operatorNs string) error {
	// Delete the previous version ODLM operator
	klog.Info("Trying to delete ODLM operator in openshift-operators")
	if err := b.deleteSubscription("operand-deployment-lifecycle-manager-app", "openshift-operators"); err != nil {
		klog.Errorf("Failed to delete ODLM operator in openshift-operators: %v", err)
		return err
	}

	// Install ODLM Operator
	klog.Info("Installing ODLM Operator")
	if operatorNs == constant.ClusterOperatorNamespace {
		if err := b.renderTemplate(constant.ODLMClusterSubscription, b.CSData); err != nil {
			return err
		}
	} else {
		// SaaS or on-prem multi instances case, enable odlm-scope
		b.CSData.IsolatedModeEnable = strconv.FormatBool(b.MultiInstancesEnable)

		cm, err := util.GetCmOfMapCs(b.Client)
		if err == nil {
			err := util.UpdateNSList(b.Reader, b.Client, cm, "nss-odlm-scope", b.CSData.MasterNs, true)
			if err != nil {
				return err
			}
		} else if !errors.IsNotFound(err) {
			klog.Errorf("Failed to get common-service-maps: %v", err)
			return err
		}

		if err := b.renderTemplate(constant.ODLMNamespacedSubscription, b.CSData); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bootstrap) createNsSubscription(manualManagement bool) error {
	resourceName := constant.NSSubscription
	subNameToRemove := constant.NsRestrictedSubName
	if manualManagement {
		resourceName = constant.NSRestrictedSubscription
		subNameToRemove = constant.NsSubName
	}

	if err := b.deleteSubscription(subNameToRemove, b.CSData.MasterNs); err != nil {
		return err
	}

	if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
		return err
	}

	return nil
}

// CreateNsScopeConfigmap creates nss configmap for operators
func (b *Bootstrap) CreateNsScopeConfigmap() error {
	cmRes := constant.NamespaceScopeConfigMap
	if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cmRes, placeholder, b.CSData.MasterNs))); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) createCrossplaneSubscription() error {
	resourceName := constant.CrossSubscription
	if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
		return err
	}

	return nil
}

func (b *Bootstrap) createCrossplaneConfiguration() error {
	resourceName := constant.CrossConfiguration
	if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) createCrossplaneLock() error {
	resourceName := constant.CrossLock
	if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) createCloudSubscription() error {
	resourceName := constant.IbmCloudSubscription
	if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
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

func (b *Bootstrap) waitOperatorReady(name, namespace string) error {
	time.Sleep(time.Second * 5)
	if err := utilwait.PollImmediate(time.Second*10, time.Minute*10, func() (done bool, err error) {
		klog.Infof("Waiting for Operator %s is ready...", name)
		key := types.NamespacedName{Name: name, Namespace: namespace}
		sub := &olmv1alpha1.Subscription{}
		if err := b.Reader.Get(context.TODO(), key, sub); err != nil {
			if errors.IsNotFound(err) {
				klog.V(3).Infof("NotFound subscription %s/%s", namespace, name)
			} else {
				klog.Errorf("Failed to get subscription %s/%s", namespace, name)
			}
			return false, client.IgnoreNotFound(err)
		}

		if version, ok := CSOperatorVersions[sub.Name]; ok {
			if sub.Status.InstalledCSV == "" {
				return false, nil
			}
			csvList := strings.Split(sub.Status.InstalledCSV, ".v")
			if len(csvList) != 2 {
				return false, nil
			}
			csvVersion := csvList[1]
			csvVersionSlice := strings.Split(csvVersion, ".")
			VersionSlice := strings.Split(version, ".")
			for index := range csvVersionSlice {
				csvVersion, err := strconv.Atoi(csvVersionSlice[index])
				if err != nil {
					return false, err
				}
				templateVersion, err := strconv.Atoi(VersionSlice[index])
				if err != nil {
					return false, err
				}
				if csvVersion > templateVersion {
					break
				} else if csvVersion == templateVersion {
					continue
				} else {
					return false, nil
				}
			}
		}

		// check csv
		csvName := sub.Status.InstalledCSV
		if csvName != "" {
			csv := &olmv1alpha1.ClusterServiceVersion{}
			if err := b.Reader.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: csvName}, csv); errors.IsNotFound(err) {
				klog.Errorf("Notfound Cluster Service Version: %v", err)
				return false, nil
			} else if err != nil {
				klog.Errorf("Failed to get Cluster Service Version: %v", err)
				return false, err
			}
			if csv.Status.Phase != olmv1alpha1.CSVPhaseSucceeded {
				return false, nil
			}
			if csv.Status.Reason != olmv1alpha1.CSVReasonInstallSuccessful {
				return false, nil
			}
			klog.Infof("Cluster Service Version %s/%s is ready", csv.Namespace, csv.Name)
			return true, nil
		}
		return false, nil
	}); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) waitALLOperatorReady(namespace string) error {
	subList := &olmv1alpha1.SubscriptionList{}

	if err := b.Reader.List(context.TODO(), subList, &client.ListOptions{Namespace: namespace, LabelSelector: labels.SelectorFromSet(map[string]string{
		"operator.ibm.com/opreq-control": "true",
	})}); err != nil {
		return err
	}

	var (
		errs []error
		mu   sync.Mutex
		wg   sync.WaitGroup
	)

	for _, sub := range subList.Items {
		var (
			// Copy variables into iteration scope
			name = sub.Name
			ns   = sub.Namespace
		)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.waitOperatorReady(name, ns); err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, err)
			}
		}()
	}
	wg.Wait()

	return utilerrors.NewAggregate(errs)

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

// func (b *Bootstrap) getResFromAnnotations(annotations map[string]string, resName string, resNs string) (*unstructured.Unstructured, error) {
// 	if r, ok := annotations[resName]; ok {
// 		yamlContent := util.Namespacelize(r, placeholder, resNs)
// 		obj, err := util.YamlToObject([]byte(yamlContent))
// 		if err != nil {
// 			return obj, err
// 		}
// 		return obj, nil
// 	} else {
// 		klog.Warningf("No resource %s found in annotations", resName)
// 	}
// 	return nil, nil
// }

// func (b *Bootstrap) getYamlFromAnnotations(annotations map[string]string, resName string) string {
// 	if r, ok := annotations[resName]; ok {
// 		return r
// 	}
// 	klog.Warningf("No yaml %s found in annotations", resName)
// 	return ""
// }

func (b *Bootstrap) UpdateCsOpApproval() error {
	sub := &olmv1alpha1.Subscription{}
	subKey := types.NamespacedName{
		Name:      "ibm-common-service-operator",
		Namespace: b.CSData.MasterNs,
	}

	if err := b.Reader.Get(ctx, subKey, sub); err != nil {
		return err
	}
	if b.CSData.ApprovalMode == string(olmv1alpha1.ApprovalManual) && sub.Spec.InstallPlanApproval != olmv1alpha1.ApprovalManual {
		sub.Spec.InstallPlanApproval = olmv1alpha1.ApprovalManual
		if err := b.Client.Update(ctx, sub); err != nil {
			return err
		}
		podList := &corev1.PodList{}
		opts := []client.ListOption{
			client.InNamespace(b.CSData.MasterNs),
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

	return nil
}

func (b *Bootstrap) updateApprovalMode() error {
	opreg := &odlm.OperandRegistry{}
	opregKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: b.CSData.MasterNs,
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

	return nil
}

// WaitResourceReady returns true only when the specific resource CRD is created
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
		err = b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cr, placeholder, b.CSData.MasterNs)))
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
