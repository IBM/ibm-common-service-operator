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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
	record.EventRecorder
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
	ZenOperatorImage   string
	ICPPKOperator      string
	ICPPICOperator     string
	ICPOperator        string
	IsOCP              bool
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
	isOCP, err := isOCP(mgr, masterNs)
	if err != nil {
		return
	}

	if !isOCP {
		csOperators = []CSOperator{
			{"Secretshare Operator", constant.SecretshareCRD, constant.SecretshareRBAC, constant.SecretshareCR, csSecretShareDeployment, constant.SecretshareKind, constant.SecretshareAPIVersion},
		}
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
		ZenOperatorImage:  util.GetImage("IBM_ZEN_OPERATOR_IMAGE"),
		ICPPKOperator:     constant.ICPPKOperator,
		ICPPICOperator:    constant.ICPPICOperator,
		ICPOperator:       constant.ICPOperator,
		IsOCP:             isOCP,
	}

	bs = &Bootstrap{
		Client:               mgr.GetClient(),
		Reader:               mgr.GetAPIReader(),
		Config:               mgr.GetConfig(),
		EventRecorder:        mgr.GetEventRecorderFor("ibm-common-service-operator"),
		Manager:              deploy.NewDeployManager(mgr),
		SaasEnable:           util.CheckSaas(mgr.GetAPIReader()),
		MultiInstancesEnable: util.CheckMultiInstances(mgr.GetAPIReader()),
		CSOperators:          csOperators,
		CSData:               csData,
	}

	if !bs.MultiInstancesEnable {
		bs.CSData.ControlNs = bs.CSData.MasterNs
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

// CrossplaneOperatorProviderOperator installs Crossplane & Provider when bedrockshim is true
func (b *Bootstrap) CrossplaneOperatorProviderOperator(instance *apiv3.CommonService) error {

	// Install Crossplane Operator & Provider Operator
	bedrockshim := false
	removeCrossplane := false
	if instance.Spec.Features != nil {
		if instance.Spec.Features.Bedrockshim != nil {
			bedrockshim = instance.Spec.Features.Bedrockshim.Enabled
			removeCrossplane = instance.Spec.Features.Bedrockshim.CrossplaneProviderRemoval
		}
	}

	var isLater bool
	var err error

	if !removeCrossplane {
		if bedrockshim {
			b.CSData.CrossplaneProvider = "odlm"

			if b.SaasEnable {
				b.CSData.CrossplaneProvider = "ibmcloud"
			}
			if b.MultiInstancesEnable {
				resourceName := constant.CrossSubscription
				isLater, err = b.CompareChannel(resourceName)
				if err != nil {
					return err
				}
			} else {
				isLater = false
			}
			//isLater value of false means we install, true means we do not install
			if !isLater {
				if err := b.installCrossplaneOperator(); err != nil {
					return err
				}
				switch b.CSData.CrossplaneProvider {
				case "odlm":
					if err := b.installKubernetesProvider(); err != nil {
						return err
					}
				case "ibmcloud":
					if err := b.installIBMCloudProvider(); err != nil {
						return err
					}
				}
				installPlanApproval := instance.Spec.InstallPlanApproval
				if installPlanApproval != "" || b.CSData.ApprovalMode == string(olmv1alpha1.ApprovalManual) {
					if err := b.updateICPApprovalMode(); err != nil {
						klog.Errorf("Failed to update approval mode for %s in namespace %s: %v", instance.Name, instance.Namespace, err)
					}
				}
			} else {
				klog.Infof("Crossplane operator already exists at a later version in control namespace. Skipping.")
			}
		} else {
			if err := b.DeleteCrossplaneAndProviderSubscription(b.CSData.ControlNs); err != nil {
				return err
			}
		}
	}

	if removeCrossplane {
		if err := b.DeleteCrossplaneProviderSubscription(b.CSData.ControlNs); err != nil {
			return err
		}
	}

	return nil
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

// DeleteCrossplaneAndProviderSubscription deletes Crossplane & Provider subscription when bedrockshim set to false or CS CR is removed
func (b *Bootstrap) DeleteCrossplaneAndProviderSubscription(namespace string) error {
	// Fetch all the CommonService instances
	klog.V(2).Info("Fetch all the CommonService instances")
	csList := util.NewUnstructuredList("operator.ibm.com", "CommonService", "v3")
	if err := b.Client.List(ctx, csList); err != nil {
		return err
	}
	uninstallCrossplane := true
	for _, cs := range csList.Items {
		if cs.GetDeletionTimestamp() != nil {
			continue
		}
		if cs.Object["spec"].(map[string]interface{})["features"] != nil &&
			cs.Object["spec"].(map[string]interface{})["features"].(map[string]interface{})["bedrockshim"] != nil &&
			cs.Object["spec"].(map[string]interface{})["features"].(map[string]interface{})["bedrockshim"].(map[string]interface{})["enabled"] != nil {
			if cs.Object["spec"].(map[string]interface{})["features"].(map[string]interface{})["bedrockshim"].(map[string]interface{})["enabled"].(bool) {
				uninstallCrossplane = false
			}
		}
	}

	if uninstallCrossplane {
		crossplaneInstalled := false
		_, crossplaneErr := b.GetSubscription(ctx, constant.ICPOperator, namespace)
		if errors.IsNotFound(crossplaneErr) {
			klog.Infof("Skipped the uninstallation, %s not installed", constant.ICPOperator)
		} else if crossplaneErr != nil {
			klog.Errorf("Failed to get subscription %s/%s", namespace, constant.ICPOperator)
		} else {
			crossplaneInstalled = true
			// delete crossplane cr
			klog.Infof("Trying to delete %s CR in %s", constant.ICPOperator, namespace)
			resourceCrossConfiguration := constant.CrossConfiguration
			if err := b.DeleteFromYaml(resourceCrossConfiguration, b.CSData); err != nil {
				return err
			}
			resourceCrossLock := constant.CrossLock
			if err := b.DeleteFromYaml(resourceCrossLock, b.CSData); err != nil {
				return err
			}
		}

		_, providerErr := b.GetSubscription(ctx, constant.ICPPKOperator, namespace)
		if errors.IsNotFound(providerErr) {
			klog.Infof("%s not installed, skipping", constant.ICPPKOperator)
		} else if providerErr != nil {
			klog.Errorf("Failed to get subscription %s/%s", namespace, constant.ICPPKOperator)
		} else {
			klog.Infof("Trying to delete Kubernetes Provider in %s", namespace)
			// delete ProviderConfig cr
			resourceCrossKubernetesProviderConfig := constant.CrossKubernetesProviderConfig
			if err := b.DeleteFromYaml(resourceCrossKubernetesProviderConfig, b.CSData); err != nil {
				return err
			}

			// delete Kubernetes Provider subscription
			klog.Infof("Trying to delete %s in %s", constant.ICPPKOperator, namespace)
			if err := b.deleteSubscription(constant.ICPPKOperator, namespace); err != nil {
				klog.Errorf("Failed to delete %s in %s: %v", constant.ICPPKOperator, namespace, err)
				return err
			}
		}

		_, providerErr = b.GetSubscription(ctx, constant.ICPPICOperator, namespace)
		if errors.IsNotFound(providerErr) {
			klog.Infof("Skipped the uninstallation, %s not installed", constant.ICPPICOperator)
		} else if providerErr != nil {
			klog.Errorf("Failed to get subscription %s/%s", namespace, constant.ICPPICOperator)
		} else {
			klog.Infof("Trying to delete IBM Cloud Provider in %s", namespace)
			// delete ProviderConfig cr
			resourceCrossIBMCloudProviderConfig := constant.CrossIBMCloudProviderConfig
			if err := b.DeleteFromYaml(resourceCrossIBMCloudProviderConfig, b.CSData); err != nil {
				return err
			}

			// delete IBM Cloud Provider subscription
			klog.Infof("Trying to delete %s in %s", constant.ICPPICOperator, namespace)
			if err := b.deleteSubscription(constant.ICPPICOperator, namespace); err != nil {
				klog.Errorf("Failed to delete %s in %s: %v", constant.ICPPICOperator, namespace, err)
				return err
			}
		}

		if crossplaneInstalled {
			// wait compositeresourcedefinitions to be deleted
			if deleteErr := b.WaitForCRDeletion("apiextensions.ibm.crossplane.io", "v1", "compositeresourcedefinitions"); deleteErr != nil {
				return deleteErr
			}
			// wait compositions to be deleted
			if deleteErr := b.WaitForCRDeletion("apiextensions.ibm.crossplane.io", "v1", "compositions"); deleteErr != nil {
				return deleteErr
			}

			// delete crossplane operator subscription
			klog.Infof("Trying to delete %s in %s", constant.ICPOperator, namespace)
			if err := b.deleteSubscription(constant.ICPOperator, namespace); err != nil {
				klog.Errorf("Failed to delete %s in %s: %v", constant.ICPOperator, namespace, err)
				return err
			}
		}
		CFCrossplaneConfigmap := constant.CFCrossplaneConfigMap
		if err := b.DeleteFromYaml(CFCrossplaneConfigmap, b.CSData); err != nil {
			return err
		}
	}
	return nil
}

// decoupling Crossplane from Kafka resources and keep Kafka resources left untouched
func (b *Bootstrap) DeleteCrossplaneProviderSubscription(namespace string) error {

	// uninstall ibm-crossplane-provider-kubernetes-operator
	_, providerErr := b.GetSubscription(ctx, constant.ICPPKOperator, namespace)
	if errors.IsNotFound(providerErr) {
		klog.Infof("%s not installed, skipping", constant.ICPPKOperator)
	} else if providerErr != nil {
		klog.Errorf("Failed to get subscription %s/%s", namespace, constant.ICPPKOperator)
	} else {
		// delete Kubernetes Provider subscription
		klog.Infof("Trying to delete %s in %s", constant.ICPPKOperator, namespace)
		if err := b.deleteSubscription(constant.ICPPKOperator, namespace); err != nil {
			klog.Errorf("Failed to delete %s in %s: %v", constant.ICPPKOperator, namespace, err)
			return err
		}
	}

	// ibm-crossplane-provider-ibm-cloud-operator
	_, providerErr = b.GetSubscription(ctx, constant.ICPPICOperator, namespace)
	if errors.IsNotFound(providerErr) {
		klog.Infof("Skipped the uninstallation, %s not installed", constant.ICPPICOperator)
	} else if providerErr != nil {
		klog.Errorf("Failed to get subscription %s/%s", namespace, constant.ICPPICOperator)
	} else {
		// delete IBM Cloud Provider subscription
		klog.Infof("Trying to delete %s in %s", constant.ICPPICOperator, namespace)
		if err := b.deleteSubscription(constant.ICPPICOperator, namespace); err != nil {
			klog.Errorf("Failed to delete %s in %s: %v", constant.ICPPICOperator, namespace, err)
			return err
		}
	}
	if err := b.CreateCFCrossplaneConfigMap(); err != nil {
		return err
	}
	return nil
}

// create or update configmap cf-crossplane
func (b *Bootstrap) CreateCFCrossplaneConfigMap() error {
	resourceName := constant.CFCrossplaneConfigMap
	if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(resourceName, placeholder, b.CSData.MasterNs))); err != nil {
		return err
	}
	return nil
}

// wait for CR to be deleted from the cluster
func (b *Bootstrap) WaitForCRDeletion(APIGroup string, APIVersion string, kind string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		klog.Errorf("Failed to get config: %v", err)
		return err
	}
	dynamic := dynamic.NewForConfigOrDie(cfg)

	resourceList, err := util.GetResourcesDynamically(ctx, dynamic, APIGroup, APIVersion, kind)
	if err != nil {
		klog.Errorf("error getting resource: %v\n", err)
		return err
	}

	for _, item := range resourceList {
		// waiting for the object be deleted
		if err := utilwait.PollImmediate(time.Second*10, time.Minute*5, func() (done bool, err error) {
			_, errNotFound := b.GetObject(&item)
			if errors.IsNotFound(errNotFound) {
				return true, nil
			}
			klog.Infof("waiting for %s with name: %s to delete\n", item.GetKind(), item.GetName())
			return false, nil
		}); err != nil {
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
		klog.Info("Creating OperatorGroup for IBM Common Services in control namespace")
		if err := b.CreateOperatorGroup(b.CSData.ControlNs); err != nil {
			klog.Errorf("Failed to create OperatorGroup for IBM Common Services in control namespace: %v", err)
			return err
		}
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

	// Clean up deprecated resource
	if err := b.cleanup(operatorNs); err != nil {
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
		obj, err := b.GetObjs(constant.CSV3SaasOperandRegistry, b.CSData)
		if err != nil {
			return err
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
			v1IsLarger, convertErr := util.CompareVersion(obj[0].GetAnnotations()["version"], objInCluster.GetAnnotations()["version"])
			if convertErr != nil {
				return convertErr
			}
			if v1IsLarger {
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
			if v1IsLarger {
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

func (b *Bootstrap) CheckCsSubscription() error {
	subs, err := b.ListSubscriptions(ctx, b.CSData.MasterNs, client.ListOptions{Namespace: b.CSData.MasterNs, LabelSelector: labels.SelectorFromSet(map[string]string{
		"operators.coreos.com/ibm-common-service-operator." + b.CSData.MasterNs: "",
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

func (b *Bootstrap) CreateOperatorGroup(namespace string) error {
	existOG := &olmv1.OperatorGroupList{}
	if err := b.Reader.List(context.TODO(), existOG, &client.ListOptions{Namespace: namespace}); err != nil {
		return err
	}
	if len(existOG.Items) == 0 {
		if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(constant.CsOperatorGroup, placeholder, namespace))); err != nil {
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
func (b *Bootstrap) GetOperandRegistry(ctx context.Context, name, namespace string) *odlm.OperandRegistry {
	klog.V(2).Infof("Fetch OperandRegistry: %v/%v", namespace, name)
	opreg := &odlm.OperandRegistry{}
	opregKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	if err := b.Reader.Get(ctx, opregKey, opreg); err != nil {
		klog.Errorf("failed to get OperandRegistry %s: %v", opregKey.String(), err)
		return nil
	}

	return opreg
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
	// Create NSS CRs in Control Ns for multi instances case
	if b.MultiInstancesEnable {
		if err := b.renderTemplate(constant.NamespaceScopeCRMulti, b.CSData); err != nil {
			return err
		}
	} else {
		// Create NSS CRs managedby ODLM for Single CS instance case
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
		if b.MultiInstancesEnable {
			err := util.UpdateAllNSList(b.Reader, b.Client, cm, "common-service", b.CSData.ControlNs)
			if err != nil {
				return err
			}
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

	if err := b.waitResourceReady("pkg.ibm.crossplane.io/v1", "Configuration"); err != nil {
		return err
	}

	klog.Info("Creating Crossplane Configuration")
	if err := b.createCrossplaneConfiguration(); err != nil {
		klog.Errorf("Failed to create or update Crossplane Configuration: %v", err)
		return err
	}

	return nil
}

func (b *Bootstrap) installKubernetesProvider() error {
	klog.Info("Creating Crossplane Kubernetes Provider subscription")
	if err := b.createCrossplaneKubernetesProviderSubscription(); err != nil {
		klog.Errorf("Failed to create or update Crossplane Kubernetes Provider subscription: %v", err)
		return err
	}

	if err := b.waitResourceReady("kubernetes.crossplane.io/v1alpha1", "ProviderConfig"); err != nil {
		return err
	}

	klog.Info("Creating Crossplane Kubernetes ProviderConfig")
	if err := b.createCrossplaneKubernetesProviderConfig(); err != nil {
		klog.Errorf("Failed to create or update Crossplane Kubernetes ProviderConfig: %v", err)
		return err
	}
	return nil
}

func (b *Bootstrap) installIBMCloudProvider() error {
	klog.Info("Creating Crossplane IBM Cloud Provider subscription")
	if err := b.createCrossplaneIBMCloudProviderSubscription(); err != nil {
		klog.Errorf("Failed to create or update Crossplane IBM Cloud Provider subscription: %v", err)
		return err
	}

	if err := b.waitResourceReady("ibmcloud.crossplane.io/v1beta1", "ProviderConfig"); err != nil {
		return err
	}

	klog.Info("Creating Crossplane IBM Cloud ProviderConfig")
	if err := b.createCrossplaneIBMCloudProviderConfig(); err != nil {
		klog.Errorf("Failed to create or update Crossplane IBM Cloud ProviderConfig: %v", err)
		return err
	}
	return nil
}

func (b *Bootstrap) cleanup(operatorNs string) error {
	bindinfo := &unstructured.Unstructured{}
	bindinfo.SetGroupVersionKind(schema.GroupVersionKind{Group: "operator.ibm.com", Version: "v1alpha1", Kind: "OperandBindInfo"})
	bindinfo.SetName("ibm-commonui-bindinfo")
	bindinfo.SetNamespace(operatorNs)
	if err := b.Client.Delete(context.TODO(), bindinfo); err != nil && !errors.IsNotFound(err) {
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

	if b.MultiInstancesEnable {
		resourceName = constant.NSSubscriptionMulti
		subNameToRemove = constant.NsRestrictedSubName
		if manualManagement {
			resourceName = constant.NSRestrictedSubscriptionMulti
			subNameToRemove = constant.NsSubName
		}

		var isLater bool
		var channelerr error

		if isLater, channelerr = b.CompareChannel(resourceName); channelerr != nil {
			if errors.IsNotFound(channelerr) {
				if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
					return err
				}
			} else {
				return channelerr
			}
		}

		if isLater {
			klog.Infof("Namespace Scope operator already exists at a later version in control namespace. Skipping.")
		} else {
			if err := b.deleteSubscription(subNameToRemove, b.CSData.ControlNs); err != nil {
				return err
			}
			if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateNsScopeConfigmap creates nss configmap for operators
func (b *Bootstrap) CreateNsScopeConfigmap() error {
	cmRes := constant.NamespaceScopeConfigMap
	if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cmRes, placeholder, b.CSData.MasterNs))); err != nil {
		return err
	}
	if b.MultiInstancesEnable {
		cmRes = constant.NamespaceScopeConfigMapMulti
		if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cmRes, placeholder, b.CSData.ControlNs))); err != nil {
			return err
		}
	}
	return nil
}

// CompareChannel function sets up the CompareVersion function for When multi instance is enabled.
// When multi instance is enabled, the crossplane operator will be a singleton service deployed in the control ns.
// We do not want to overwrite a later version of crossplane operator with an earlier version, this is what CompareChannel checks for.
func (b *Bootstrap) CompareChannel(objectTemplate string, alwaysUpdate ...bool) (bool, error) {
	objects, err := b.GetObjs(objectTemplate, b.CSData)
	if err != nil {
		return true, err
	}

	obj := objects[0]

	_, err = b.GetObject(obj)
	if errors.IsNotFound(err) {
		klog.Infof("Creating resource with name: %s, namespace: %s\n", obj.GetName(), obj.GetNamespace())
		return false, nil
	} else if err != nil {
		return true, err
	}
	sub, err := b.GetSubscription(ctx, obj.GetName(), b.CSData.ControlNs) //doesn't actually return the subscription, returns an unstructured.Unstructured object
	if errors.IsNotFound(err) {
		klog.Errorf("Failed to get an existing subscription for %s/%s. Creating new subscription.", b.CSData.ControlNs, obj.GetName())
		return false, nil
	} else if err != nil {
		klog.Errorf("Failed to get an existing subscription for %s/%s because %s", b.CSData.ControlNs, obj.GetName(), err)
		return true, err
	}
	subVersion := fmt.Sprintf("%v", sub.Object["spec"].(map[string]interface{})["channel"])
	subVersionStr := subVersion[1:]
	channelStr := b.CSData.Channel[1:]
	isLater, convertErr := util.CompareVersion(subVersionStr, channelStr)
	//Return of "false" will mean that the operator will be installed as normal/updated to the new version
	//Return of "true" means that the existing crossplane operator is at a later version than the cs operator is attempting to install so we leave the existing untouched.
	return isLater, convertErr
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

func (b *Bootstrap) createCrossplaneKubernetesProviderSubscription() error {
	resourceName := constant.CrossKubernetesProviderSubscription
	if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) createCrossplaneKubernetesProviderConfig() error {
	resourceName := constant.CrossKubernetesProviderConfig
	if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) createCrossplaneIBMCloudProviderSubscription() error {
	resourceName := constant.CrossIBMCloudProviderSubscription

	if err := b.renderTemplate(resourceName, b.CSData, true); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) createCrossplaneIBMCloudProviderConfig() error {
	resourceName := constant.CrossIBMCloudProviderConfig
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
			klog.V(2).Infof("Cluster Service Version %s/%s is ready", csv.Namespace, csv.Name)
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
		if sub.Name == "ibm-zen-operator" {
			continue
		}
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
		commonserviceNS = b.CSData.MasterNs
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
		return fmt.Errorf("not found ibm-common-service-operator subscription in namespace: %v or %v", b.CSData.MasterNs, constant.ClusterOperatorNamespace)
	}

	if len(subList.Items) > 1 {
		return fmt.Errorf("found more than one ibm-common-service-operator subscription in namespace: %v or %v, skip this", b.CSData.MasterNs, constant.ClusterOperatorNamespace)
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

// update approval mode for the given operator
// need this function because ODLM and namespace operator are not in operandRegistry
// if it is NSS or Crossplane operator and it is in Multi-instance mode, we will update it in control NameSpace
func (b *Bootstrap) UpdateOpApproval(operatorName string, namespace string) error {
	sub := &olmv1alpha1.Subscription{}
	subKey := types.NamespacedName{
		Name:      operatorName,
		Namespace: namespace,
	}

	if err := b.Reader.Get(ctx, subKey, sub); err != nil {
		return err
	}
	if b.CSData.ApprovalMode == string(olmv1alpha1.ApprovalManual) && sub.Spec.InstallPlanApproval != olmv1alpha1.ApprovalManual {
		sub.Spec.InstallPlanApproval = olmv1alpha1.ApprovalManual
		if err := b.Client.Update(ctx, sub); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bootstrap) updateICPApprovalMode() error {
	klog.Info("Updating crossplane operators Approvalmode")

	if err := b.UpdateOpApproval(constant.ICPOperator, b.CSData.ControlNs); err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Failed to update %s subscription: %v", constant.ICPOperator, err)
			return err
		}
		klog.V(2).Infof("%s not installed, skipping updating approval strategy", constant.ICPOperator)

	}

	if err := b.UpdateOpApproval(constant.ICPPICOperator, b.CSData.ControlNs); err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Failed to update %s subscription: %v", constant.ICPPICOperator, err)
			return err
		}
		klog.V(2).Infof("%s not installed, skipping updating approval strategy", constant.ICPPICOperator)

	}

	if err := b.UpdateOpApproval(constant.ICPPKOperator, b.CSData.ControlNs); err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Failed to update %s subscription: %v", constant.ICPPKOperator, err)
			return err
		}
		klog.V(2).Infof("%s not installed, skipping updating approval strategy", constant.ICPPKOperator)

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

	if err = b.UpdateOpApproval(constant.IBMODLMPackage, b.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to update %s subscription: %v", constant.IBMODLMPackage, err)
		return err
	}

	if err = b.UpdateOpApproval(constant.IBMNSSPackage, b.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to update %s subscription: %v", constant.IBMNSSPackage, err)
		return err
	}

	if b.MultiInstancesEnable {
		if err = b.UpdateOpApproval(constant.IBMNSSPackage, b.CSData.ControlNs); err != nil {
			klog.Errorf("Failed to update %s subscription: %v", constant.IBMNSSPackage, err)
			return err
		}
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
	deployedNs := b.CSData.MasterNs
	if b.MultiInstancesEnable {
		deployedNs = b.CSData.ControlNs
	}
	_, err := b.GetSubscription(ctx, constant.CertManagerSub, deployedNs)
	if errors.IsNotFound(err) {
		klog.Infof("Skipped deploying cert manager CRs, %s not installed yet.", constant.CertManagerSub)
	} else if err != nil {
		klog.Errorf("Failed to get subscription %s/%s", deployedNs, constant.CertManagerSub)
	} else {
		klog.V(2).Info("Fetch all the CommonService instances")
		csList := util.NewUnstructuredList("operator.ibm.com", "CommonService", "v3")
		if err := b.Client.List(ctx, csList); err != nil {
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
			if err := b.waitResourceReady(constant.CertManagerAPIGroupVersion, kind); err != nil {
				klog.Errorf("Failed to wait for resource ready with kind: %s, apiGroupVersion: %s", kind, constant.CertManagerAPIGroupVersion)
			}
		}

		for _, cr := range constant.CertManagerIssuers {
			if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cr, placeholder, b.CSData.MasterNs))); err != nil {
				return err
			}
		}
		if deployRootCert {
			for _, cr := range constant.CertManagerCerts {
				if err := b.CreateOrUpdateFromYaml([]byte(util.Namespacelize(cr, placeholder, b.CSData.MasterNs))); err != nil {
					return err
				}
			}
		} else {
			klog.Infof("Skipped deploying %s, BYOCertififcate feature is enabled in %s", constant.CSCACertificate, crWithBYOCert)
		}
	}
	return nil
}
