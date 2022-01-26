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
	"context"
	"strconv"
	"strings"
	"time"

	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	"github.com/IBM/ibm-common-service-operator/controllers/deploy"
)

var (
	CsSubResource             = "csOperatorSubscription"
	OdlmNamespacedSubResource = "odlmNamespacedSubscription"
	OdlmClusterSubResource    = "odlmClusterSubscription"
	OdlmCrResources           = []string{"csOperandRegistry", "csOperandConfig"}
)

var csOperators = []struct {
	Name       string
	CRD        string
	RBAC       string
	CR         string
	Deployment string
	Kind       string
	APIVersion string
}{
	{"Webhook Operator", constant.WebhookCRD, constant.WebhookRBAC, constant.WebhookCR, constant.CsWebhookOperator, constant.WebhookKind, constant.WebhookAPIVersion},
	{"Secretshare Operator", constant.SecretshareCRD, constant.SecretshareRBAC, constant.SecretshareCR, constant.CsSecretshareOperator, constant.SecretshareKind, constant.SecretshareAPIVersion},
}

var ctx = context.Background()

type Bootstrap struct {
	client.Client
	client.Reader
	Config *rest.Config
	*deploy.Manager
}

// NewBootstrap is the way to create a NewBootstrap struct
func NewBootstrap(mgr manager.Manager) *Bootstrap {
	return &Bootstrap{
		Client:  mgr.GetClient(),
		Reader:  mgr.GetAPIReader(),
		Config:  mgr.GetConfig(),
		Manager: deploy.NewDeployManager(mgr),
	}
}

// InitResources initialize resources at the bootstrap of operator
func (b *Bootstrap) InitResources(manualManagement bool) error {
	// Get all the resources from the deployment annotations
	annotations, err := b.GetAnnotations()
	if err != nil {
		return err
	}

	operatorNs, err := util.GetOperatorNamespace()
	if err != nil {
		klog.Errorf("Getting operator namespace failed: %v", err)
		return err
	}

	// Grant cluster-admin to namespace scope operator
	if operatorNs == constant.ClusterOperatorNamespace {
		klog.Info("Creating cluster-admin permission RBAC")
		if err := b.createOrUpdateFromYaml([]byte(util.Namespacelize(constant.ClusterAdminRBAC))); err != nil {
			return err
		}
	}

	// Install Namespace Scope Operator
	klog.Info("Creating namespace-scope configmap")
	// Backward compatible upgrade from version 3.4.x
	if err := b.CreateNsScopeConfigmap(); err != nil {
		klog.Errorf("Failed to create Namespace Scope ConfigMap: %v", err)
		return err
	}

	klog.Info("Creating Namespace Scope Operator subscription")
	if err := b.createNsSubscription(manualManagement, annotations); err != nil {
		klog.Errorf("Failed to create Namespace Scope Operator subscription: %v", err)
		return err
	}

	if err := b.waitResourceReady("operator.ibm.com/v1", "NamespaceScope"); err != nil {
		return err
	}

	// Create NamespaceScope CR
	if err := b.createOrUpdateFromYaml([]byte(util.Namespacelize(constant.NamespaceScopeCR))); err != nil {
		return err
	}

	// Install CS Operators
	for _, operator := range csOperators {
		klog.Infof("Installing %s", operator.Name)
		// Create Operator CRD
		if err := b.createOrUpdateFromYaml([]byte(operator.CRD)); err != nil {
			return err
		}
		// Create Operator RBAC
		if err := b.createOrUpdateFromYaml([]byte(util.Namespacelize(operator.RBAC))); err != nil {
			return err
		}
		// Create Operator Deployment
		if err := b.createOrUpdateFromYaml([]byte(util.ReplaceImages(util.Namespacelize(operator.Deployment)))); err != nil {
			return err
		}
		// Wait for CRD ready
		if err := b.waitResourceReady(operator.APIVersion, operator.Kind); err != nil {
			return err
		}
		// Create Operator CR
		if err := b.createOrUpdateFromYaml([]byte(util.Namespacelize(operator.CR))); err != nil {
			return err
		}
	}

	// Create extra RBAC for ibmcloud-cluster-ca-cert and ibmcloud-cluster-info in kube-public
	klog.Info("Creating RBAC for ibmcloud-cluster-info & ibmcloud-cluster-ca-cert in kube-public")
	if err := b.createOrUpdateFromYaml([]byte(constant.ExtraRBAC)); err != nil {
		return err
	}

	// Delete the previous version ODLM operator
	klog.Info("Trying to delete ODLM operator in openshift-operators")
	if err := b.deleteSubscription("operand-deployment-lifecycle-manager-app", "openshift-operators"); err != nil {
		klog.Errorf("Failed to delete ODLM operator in openshift-operators: %v", err)
		return err
	}

	isUpgrade, err := b.checkODLMVersion("operand-deployment-lifecycle-manager-app", "ibm-common-services")
	if err != nil {
		klog.Errorf("Failed to check ODLM operator version in ibm-common-services: %v", err)
		return err
	}
	if isUpgrade {
		if err := b.checkODLMDeletion("ibm-common-services"); err != nil {
			klog.Errorf("Failed to delete ODLM operator in ibm-common-services: %v", err)
			return err
		}
	}

	// Install ODLM Operator
	klog.Info("Installing ODLM Operator")
	if operatorNs == constant.ClusterOperatorNamespace {
		if err := b.createOrUpdateResource(annotations, OdlmClusterSubResource); err != nil {
			return err
		}
	} else {
		if err := b.createOrUpdateResource(annotations, OdlmNamespacedSubResource); err != nil {
			return err
		}
	}

	// create or ODLM  OperandRegistry and OperandConfig CR resources
	if err := b.waitResourceReady("operator.ibm.com/v1alpha1", "OperandRegistry"); err != nil {
		return err
	}
	if err := b.waitResourceReady("operator.ibm.com/v1alpha1", "OperandConfig"); err != nil {
		return err
	}
	if err := b.createOrUpdateResources(annotations, OdlmCrResources); err != nil {
		return err
	}

	return nil
}

func (b *Bootstrap) CreateNamespace() error {
	nsObj := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: constant.MasterNamespace,
		},
	}

	if err := b.Client.Create(ctx, nsObj); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (b *Bootstrap) CreateCsSubscription() error {
	// Get all the resources from the deployment annotations
	annotations, err := b.GetAnnotations()
	if err != nil {
		return err
	}
	klog.Info("Creating cs operator in master namespace")
	if err := b.createOrUpdateResource(annotations, CsSubResource); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) CreateCsCR() error {
	odlm := util.NewUnstructured("operators.coreos.com", "Subscription", "v1alpha1")
	odlm.SetName("operand-deployment-lifecycle-manager-app")
	odlm.SetNamespace(constant.ClusterOperatorNamespace)
	_, err := b.GetObject(odlm)
	if errors.IsNotFound(err) {
		// Fresh Intall: No ODLM
		return b.createOrUpdateFromYaml([]byte(util.Namespacelize(constant.CsCR)))
	} else if err != nil {
		return err
	}

	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	cs.SetName("common-service")
	cs.SetNamespace(constant.MasterNamespace)
	_, err = b.GetObject(cs)
	if errors.IsNotFound(err) {
		// Upgrade: Have ODLM and NO CR
		return b.createOrUpdateFromYaml([]byte(util.Namespacelize(constant.CsNoSizeCR)))
	} else if err != nil {
		return err
	}

	// Restart: Have ODLM and CR
	return b.createOrUpdateFromYaml([]byte(util.Namespacelize(constant.CsCR)))
}

func (b *Bootstrap) CreateOperatorGroup() error {
	existOG := &olmv1.OperatorGroupList{}
	if err := b.Reader.List(context.TODO(), existOG, &client.ListOptions{Namespace: constant.MasterNamespace}); err != nil {
		return err
	}
	if len(existOG.Items) == 0 {
		if err := b.createOrUpdateFromYaml([]byte(util.Namespacelize(constant.CsOperatorGroup))); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bootstrap) createOrUpdateResource(annotations map[string]string, resName string) error {
	if r, ok := annotations[resName]; ok {
		if err := b.createOrUpdateFromYaml([]byte(util.Namespacelize(r))); err != nil {
			return err
		}
	} else {
		klog.Warningf("No resource %s found in annotations", resName)
	}
	return nil
}

func (b *Bootstrap) createOrUpdateResources(annotations map[string]string, resNames []string) error {
	for _, res := range resNames {
		if r, ok := annotations[res]; ok {
			if err := b.createOrUpdateFromYaml([]byte(util.Namespacelize(r))); err != nil {
				return err
			}
		} else {
			klog.Warningf("no resource %s found in annotations", res)
		}
	}
	return nil
}

func (b *Bootstrap) createOrUpdateFromYaml(yamlContent []byte) error {
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

func (b *Bootstrap) waitResourceReady(apiGroupVersion, kind string) error {
	dc := discovery.NewDiscoveryClientForConfigOrDie(b.Config)
	if err := utilwait.PollImmediate(time.Second*10, time.Minute*5, func() (done bool, err error) {
		exist, err := resourceExists(dc, apiGroupVersion, kind)
		if err != nil {
			return exist, err
		}
		if !exist {
			klog.Infof("waiting for resource ready with kind: %s, apiGroupVersion: %s", kind, apiGroupVersion)
		}
		return true, nil
	}); err != nil {
		return err
	}
	return nil
}

// resourceExists returns true if the given resource kind exists
// in the given api groupversion
func resourceExists(dc discovery.DiscoveryInterface, apiGroupVersion, kind string) (bool, error) {
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

func (b *Bootstrap) createNsSubscription(manualManagement bool, annotations map[string]string) error {
	resourceName := constant.NsSubResourceName
	subNameToRemove := constant.NsRestrictedSubName
	if manualManagement {
		resourceName = constant.NsRestrictedSubResourceName
		subNameToRemove = constant.NsSubName
	}

	if err := b.deleteSubscription(subNameToRemove, constant.MasterNamespace); err != nil {
		return err
	}

	if err := b.createOrUpdateResource(annotations, resourceName); err != nil {
		return err
	}

	return nil
}

// CreateNsScopeConfigmap creates nss configmap for operators
func (b *Bootstrap) CreateNsScopeConfigmap() error {
	cmRes := constant.NamespaceScopeConfigMap
	if err := b.createOrUpdateFromYaml([]byte(util.Namespacelize(cmRes))); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) checkODLMDeletion(ns string) error {
	if err := utilwait.PollImmediate(time.Second*10, time.Minute*5, func() (done bool, err error) {
		if err := b.deleteSubscription("operand-deployment-lifecycle-manager-app", "ibm-common-services"); err != nil {
			klog.Errorf("Failed to delete ODLM operator in ibm-common-services: %v", err)
			return false, err
		}

		deploy := &appsv1.Deployment{}
		if err := b.Reader.Get(context.TODO(), types.NamespacedName{Name: "operand-deployment-lifecycle-manager", Namespace: ns}, deploy); err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}

		owners := deploy.GetOwnerReferences()
		for _, owner := range owners {
			if owner.Kind != "ClusterServiceVersion" || owner.APIVersion != "operators.coreos.com/v1alpha1" || owner.Name == "" {
				continue
			}

			csvName := owner.Name

			csv := &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      csvName,
					Namespace: ns,
				},
			}

			if err := b.Client.Delete(context.TODO(), csv); err != nil {
				klog.Errorf("Failed to delete Cluster Service Version: %v", err)
				return false, err
			}
		}

		if err := b.Client.Delete(context.TODO(), deploy); err != nil && !errors.IsNotFound(err) {
			klog.Errorf("Failed to delete deployment: %v", err)
			return false, err
		}

		deployCheck := &appsv1.Deployment{}
		if err := b.Reader.Get(context.TODO(), types.NamespacedName{Name: "operand-deployment-lifecycle-manager", Namespace: ns}, deployCheck); err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}); err != nil {
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

func (b *Bootstrap) checkODLMVersion(name, namespace string) (bool, error) {
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

	if sub.Status.InstalledCSV == "" || sub.Status.CurrentCSV == "" {
		return true, nil
	}

	if sub.Status.InstalledCSV != sub.Status.CurrentCSV {
		return true, nil
	}

	if sub.Status.State != olmv1alpha1.SubscriptionStateAtLatest {
		return true, nil
	}

	csvList := strings.Split(sub.Status.InstalledCSV, ".v")
	if len(csvList) != 2 {
		return true, nil
	}
	csvVersion := csvList[1]
	csvVersionSlice := strings.Split(csvVersion, ".")
	// need to delete the ODLM whose version is smaller than 1.4.3 in upgrade
	OldODLMVersion := []int{1, 4, 3}
	if len(csvVersionSlice) != 3 {
		return true, nil
	}
	for index := range csvVersionSlice {
		csvVersion, err := strconv.Atoi(csvVersionSlice[index])
		if err != nil {
			return true, nil
		}
		if csvVersion < OldODLMVersion[index] {
			return true, nil
		} else if csvVersion == OldODLMVersion[index] {
			continue
		}
	}

	deploy := &appsv1.Deployment{}
	if err := b.Reader.Get(context.TODO(), types.NamespacedName{Name: "operand-deployment-lifecycle-manager", Namespace: namespace}, deploy); err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		klog.Info()
		return false, err
	}

	owners := deploy.GetOwnerReferences()
	for _, owner := range owners {
		if owner.Kind != "ClusterServiceVersion" || owner.APIVersion != "operators.coreos.com/v1alpha1" || owner.Name == "" {
			continue
		}

		csvName := owner.Name

		if csvName != sub.Status.InstalledCSV {
			return true, nil
		}
	}

	return false, nil
}
