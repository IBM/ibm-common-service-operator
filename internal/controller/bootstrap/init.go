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
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	utilyaml "github.com/ghodss/yaml"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"golang.org/x/mod/semver"
	admv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	util "github.com/IBM/ibm-common-service-operator/v4/internal/controller/common"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/deploy"
	nssv1 "github.com/IBM/ibm-namespace-scope-operator/v4/api/v1"
	ssv1 "github.com/IBM/ibm-secretshare-operator/api/v1"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"

	"maps"

	certmanagerv1 "github.com/ibm/ibm-cert-manager-operator/apis/cert-manager/v1"
)

var (
	placeholder = "placeholder"
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

func NewNonOLMBootstrap(mgr manager.Manager) (bs *Bootstrap, err error) {
	cpfsNs := util.GetCPFSNamespace(mgr.GetAPIReader())
	servicesNs := util.GetServicesNamespace(mgr.GetAPIReader())
	operatorNs, err := util.GetOperatorNamespace()
	if err != nil {
		return
	}
	csData := apiv3.CSData{
		CPFSNs:                  cpfsNs,
		ServicesNs:              servicesNs,
		OperatorNs:              operatorNs,
		CatalogSourceName:       "",
		CatalogSourceNs:         "",
		ApprovalMode:            "",
		WatchNamespaces:         util.GetWatchNamespace(),
		OnPremMultiEnable:       strconv.FormatBool(util.CheckMultiInstances(mgr.GetAPIReader())),
		ExcludedCatalog:         constant.ExcludedCatalog,
		StatusMonitoredServices: constant.StatusMonitoredServices,
		ServiceNames:            constant.ServiceNames,
		UtilsImage:              util.GetUtilsImage(),
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
	return
}

// NewBootstrap is the way to create a NewBootstrap struct
func NewBootstrap(mgr manager.Manager) (bs *Bootstrap, err error) {
	cpfsNs := util.GetCPFSNamespace(mgr.GetAPIReader())
	servicesNs := util.GetServicesNamespace(mgr.GetAPIReader())
	operatorNs, err := util.GetOperatorNamespace()
	if err != nil {
		return
	}

	odlmCatalogSourceName, odlmcatalogSourceNs := util.GetCatalogSource(constant.IBMCSPackage, operatorNs, mgr.GetAPIReader())
	if odlmCatalogSourceName == "" || odlmcatalogSourceNs == "" {
		err = fmt.Errorf("failed to get ODLM catalogsource")
		return
	}

	approvalMode, err := util.GetApprovalModeinNs(mgr.GetAPIReader(), operatorNs)
	if err != nil {
		return
	}

	csData := apiv3.CSData{
		CPFSNs:                  cpfsNs,
		ServicesNs:              servicesNs,
		OperatorNs:              operatorNs,
		CatalogSourceName:       "",
		CatalogSourceNs:         "",
		ODLMCatalogSourceName:   odlmCatalogSourceName,
		ODLMCatalogSourceNs:     odlmcatalogSourceNs,
		ApprovalMode:            approvalMode,
		WatchNamespaces:         util.GetWatchNamespace(),
		OnPremMultiEnable:       strconv.FormatBool(util.CheckMultiInstances(mgr.GetAPIReader())),
		ExcludedCatalog:         constant.ExcludedCatalog,
		StatusMonitoredServices: constant.StatusMonitoredServices,
		ServiceNames:            constant.ServiceNames,
		UtilsImage:              util.GetUtilsImage(),
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

	if r, ok := annotations["cloudPakThemesVersion"]; ok {
		bs.CSData.CloudPakThemesVersion = r
	}

	klog.Infof("Single Deployment Status: %v, MultiInstance Deployment status: %v, SaaS Depolyment Status: %v", !bs.MultiInstancesEnable, bs.MultiInstancesEnable, bs.SaasEnable)
	return
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

	// Clean v3 Namespace Scope Operator and CRs in the servicesNamespace
	if err := b.CleanNamespaceScopeResources(); err != nil {
		klog.Errorf("Failed to clean NamespaceScope resources: %v", err)
		return err
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

	// Temporary solution for EDB image ConfigMap reference
	if os.Getenv("NO_OLM") != "true" {
		klog.Infof("It is not a non-OLM mode, create EDB Image ConfigMap")
		if err := b.CreateEDBImageMaps(); err != nil {
			klog.Errorf("Failed to create EDB Image ConfigMap: %v", err)
			return err
		}
	}

	// Create Keycloak themes ConfigMap
	if err := b.CreateKeycloakThemesConfigMap(); err != nil {
		klog.Errorf("Failed to create Keycloak Themes ConfigMap: %v", err)
		return err
	}

	mutatingWebhooks := []string{constant.CSWebhookConfig, constant.OperanReqConfig}
	validatingWebhooks := []string{constant.CSMappingConfig}
	if err := b.DeleteV3Resources(mutatingWebhooks, validatingWebhooks); err != nil {
		klog.Errorf("Failed to delete v3 resources: %v", err)
		return err
	}

	// Backward compatible for All Namespace Installation Mode upgrade
	// Uninstall ODLM in servicesNamespace(ibm-common-services)
	if b.CSData.CPFSNs != b.CSData.ServicesNs {
		klog.V(2).Info("Uninstall ODLM in servicesNamespace when the topology is separation of control and data")
		if err := b.DeleteOperator(constant.IBMODLMPackage, b.CSData.ServicesNs); err != nil {
			klog.Errorf("Failed to uninstall ODLM in servicesNamespace %s", b.CSData.ServicesNs)
			return err
		}
	}

	// Check if ODLM OperandRegistry and OperandConfig are created
	klog.Info("Checking if OperandRegistry and OperandConfig CRD already exist")
	existOpreg, _ := b.CheckCRD(constant.OpregAPIGroupVersion, constant.OpregKind)
	existOpcon, _ := b.CheckCRD(constant.OpregAPIGroupVersion, constant.OpconKind)

	// Install/update Opreg and Opcon resources before installing ODLM if CRDs exist
	if existOpreg && existOpcon {

		klog.Info("Checking OperandRegistry and OperandConfig deployment status")
		if err := b.ConfigODLMOperandManagedByOperator(ctx); err != nil {
			return err
		}
		// Set "Pending" condition when creating OperandRegistry and OperandConfig
		instance.SetPendingCondition(constant.MasterCR, apiv3.ConditionTypePending, corev1.ConditionTrue, apiv3.ConditionReasonInit, apiv3.ConditionMessageInit)
		if err := b.Client.Status().Update(ctx, instance); err != nil {
			return err
		}

		klog.Info("Installing/Updating OperandRegistry")
		if err := b.InstallOrUpdateOpreg(installPlanApproval); err != nil {
			return err
		}

		klog.Info("Installing/Updating OperandConfig")
		if err := b.InstallOrUpdateOpcon(forceUpdateODLMCRs); err != nil {
			return err
		}
	}

	klog.Info("Installing ODLM Operator")
	if err := b.renderTemplate(constant.ODLMSubscription, b.CSData); err != nil {
		return err
	}

	klog.Info("Waiting for ODLM Operator to be ready")
	if isWaiting, err := b.waitOperatorCSV(constant.IBMODLMPackage, "ibm-odlm", b.CSData.CPFSNs); err != nil {
		return err
	} else if isWaiting {
		forceUpdateODLMCRs = true
	}

	// wait ODLM OperandRegistry and OperandConfig CRD
	if err := b.waitResourceReady(constant.OpregAPIGroupVersion, constant.OpregKind); err != nil {
		return err
	}
	if err := b.waitResourceReady(constant.OpregAPIGroupVersion, constant.OpconKind); err != nil {
		return err
	}
	// Reinstall/update OperandRegistry and OperandConfig if not installed/updated in the previous step
	if !existOpreg || !existOpcon || forceUpdateODLMCRs {

		// Set "Pending" condition when creating OperandRegistry and OperandConfig
		instance.SetPendingCondition(constant.MasterCR, apiv3.ConditionTypePending, corev1.ConditionTrue, apiv3.ConditionReasonInit, apiv3.ConditionMessageInit)
		if err := b.Client.Status().Update(ctx, instance); err != nil {
			return err
		}

		klog.Info("Installing/Updating OperandRegistry")
		if err := b.InstallOrUpdateOpreg(installPlanApproval); err != nil {
			return err
		}

		klog.Info("Installing/Updating OperandConfig")
		if err := b.InstallOrUpdateOpcon(forceUpdateODLMCRs); err != nil {
			return err
		}
	}
	return nil
}

// CheckWarningCondition
func (b *Bootstrap) CheckWarningCondition(instance *apiv3.CommonService) error {
	csStorageClass := &storagev1.StorageClassList{}
	err := b.Reader.List(context.TODO(), csStorageClass)
	if err != nil {
		return err
	}

	defaultCount := 0
	if len(csStorageClass.Items) > 0 {
		for _, sc := range csStorageClass.Items {
			if sc.Annotations != nil && sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
				klog.V(2).Infof("Default StorageClass found: %s\n", sc.Name)
				defaultCount++
			}
		}
	}

	// check if there is no storageClass declared under spec section and the default count is not 1
	if instance.Spec.StorageClass == "" && defaultCount != 1 {
		instance.SetWarningCondition(constant.MasterCR, apiv3.ConditionTypeWarning, corev1.ConditionTrue, apiv3.ConditionReasonWarning, apiv3.ConditionMessageMissSC)
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
		// using `ibm-common-services` ns as ServicesNs if CS CR does not exist
		if _, err := b.GetObject(cs); errors.IsNotFound(err) {
			b.CSData.ServicesNs = constant.MasterNamespace
			return b.renderTemplate(constant.CsCR, b.CSData)
		} else if err != nil {
			return err
		}
	} else {
		if _, err := b.GetObject(cs); errors.IsNotFound(err) { // Only if it's a fresh install
			// Fresh Intall: No ODLM and NO CR
			return b.renderTemplate(constant.CsCR, b.CSData)
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (b *Bootstrap) CreateCsCRNoOLM() error {
	klog.V(2).Infof("creating cs cr")

	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	cs.SetName("common-service")
	cs.SetNamespace(b.CSData.OperatorNs)

	deploy, err := b.GetDeployment()
	if err != nil {
		return err
	}

	annotations := deploy.GetAnnotations()
	almExample := annotations["alm-examples"]

	if _, err := b.GetObject(cs); errors.IsNotFound(err) { // Only if it's a fresh install
		// Fresh Intall: No ODLM and NO CR
		return b.CreateOrUpdateFromJson(almExample)
	} else if err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) CreateOrUpdateFromJson(objectTemplate string, alwaysUpdate ...bool) error {
	klog.V(2).Infof("creating object from Json: %s", objectTemplate)

	// Create a slice for crTemplates
	var crTemplates []interface{}

	// Convert CR template string to slice
	if err := json.Unmarshal([]byte(objectTemplate), &crTemplates); err != nil {
		return err
	}

	for _, crTemplate := range crTemplates {
		// Create an unstruct object for CR and request its value to CR template

		forceUpdate := false
		if len(alwaysUpdate) != 0 {
			forceUpdate = alwaysUpdate[0]
		}
		update := forceUpdate

		var cr unstructured.Unstructured
		cr.Object = crTemplate.(map[string]interface{})

		name := cr.GetName()
		if name == "" {
			continue
		}

		spec := cr.Object["spec"]
		if spec == "" {
			continue
		}

		crInCluster := unstructured.Unstructured{}
		crInCluster.SetGroupVersionKind(cr.GroupVersionKind())

		if err := b.Client.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: b.CSData.OperatorNs,
		}, &crInCluster); err != nil && !errors.IsNotFound(err) {
			return err
		} else if errors.IsNotFound(err) {
			// Create Custom Resource
			if err := b.CreateObject(&cr); err != nil {
				return err
			}
			continue
		} else {
			// Compare version
			v1IsLarger, convertErr := util.CompareVersion(cr.GetAnnotations()["version"], crInCluster.GetAnnotations()["version"])
			if convertErr != nil {
				return convertErr
			}
			if v1IsLarger {
				update = true
			}
		}

		if update {
			klog.Infof("Updating resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", cr.GetName(), cr.GetNamespace(), cr.GetKind(), cr.GetObjectKind().GroupVersionKind().Group, cr.GetObjectKind().GroupVersionKind().Version)
			resourceVersion := crInCluster.GetResourceVersion()
			cr.SetResourceVersion(resourceVersion)
			if err := b.UpdateObject(&cr); err != nil {
				return err
			}
		}
	}

	return nil
}

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
			klog.V(2).Infof("Creating resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
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
			klog.V(2).Infof("Not Found name: %s, namespace: %s, kind: %s, apiversion: %s/%s, skipping", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
			continue
		} else if err != nil {
			errMsg = err
			continue
		}

		klog.Infof("Deleting object with name: %s, namespace: %s, kind: %s, apiversion: %s/%s", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
		if err := b.DeleteObject(obj); err != nil {
			errMsg = err
		}

		// waiting for the object be deleted
		if err := utilwait.PollUntilContextTimeout(ctx, time.Second*10, time.Minute*5, true, func(ctx context.Context) (done bool, err error) {
			_, errNotFound := b.GetObject(obj)
			if errors.IsNotFound(errNotFound) {
				return true, nil
			}
			klog.Infof("waiting for object with name: %s, namespace: %s, kind: %s, apiversion: %s/%s to delete", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
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

// ListOperatorConfig returns the OperatorConfig instance with "options"
func (b *Bootstrap) ListOperatorConfig(ctx context.Context, opts ...client.ListOption) *odlm.OperatorConfigList {
	operatorConfigList := &odlm.OperatorConfigList{}
	if err := b.Client.List(ctx, operatorConfigList, opts...); err != nil {
		klog.Errorf("failed to List OperatorConfig: %v", err)
		return nil
	}

	return operatorConfigList
}

// ListNssCRs returns the NameSpaceScopes instance list with "options"
func (b *Bootstrap) ListNssCRs(ctx context.Context, namespace string) (*nssv1.NamespaceScopeList, error) {
	nssCRsList := &nssv1.NamespaceScopeList{}
	if err := b.Client.List(ctx, nssCRsList, &client.ListOptions{Namespace: namespace}); err != nil {
		klog.Errorf("failed to List NamespaceScope CRs in namespace %s: %v", namespace, err)
		return nil, err
	}

	return nssCRsList, nil
}

// ListCerts returns the Certificate instance list with "options"
func (b *Bootstrap) ListCerts(ctx context.Context, opts ...client.ListOption) *certmanagerv1.CertificateList {
	certList := &certmanagerv1.CertificateList{}
	if err := b.Client.List(ctx, certList, opts...); err != nil {
		klog.Errorf("failed to List Cert Manager Certificates: %v", err)
		return nil
	}

	return certList
}

// ListIssuer returns the Iusser instance list with "options"
func (b *Bootstrap) ListIssuer(ctx context.Context, opts ...client.ListOption) *certmanagerv1.IssuerList {
	issuerList := &certmanagerv1.IssuerList{}
	if err := b.Client.List(ctx, issuerList, opts...); err != nil {
		klog.Errorf("failed to List Cert Manager Issuers: %v", err)
		return nil
	}

	return issuerList
}

func (b *Bootstrap) CheckOperatorCatalog(ns string) error {

	err := utilwait.PollUntilContextTimeout(ctx, time.Second*10, time.Minute*3, true, func(ctx context.Context) (done bool, err error) {
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

// CheckCRD returns true if the given crd is existent
func (b *Bootstrap) CheckCRD(apiGroupVersion string, kind string) (bool, error) {
	dc := discovery.NewDiscoveryClientForConfigOrDie(b.Config)
	exist, err := b.ResourceExists(dc, apiGroupVersion, kind)
	if err != nil {
		return false, err
	}
	if !exist {
		return false, nil
	}
	return true, nil
}

// WaitResourceReady returns true only when the specific resource CRD is created and wait for infinite time
func (b *Bootstrap) WaitResourceReady(apiGroupVersion string, kind string) error {
	dc := discovery.NewDiscoveryClientForConfigOrDie(b.Config)
	if err := utilwait.PollUntilContextCancel(ctx, time.Second*10, true, func(ctx context.Context) (done bool, err error) {
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

func (b *Bootstrap) waitResourceReady(apiGroupVersion, kind string) error {
	dc := discovery.NewDiscoveryClientForConfigOrDie(b.Config)
	if err := utilwait.PollUntilContextTimeout(ctx, time.Second*10, time.Minute*2, true, func(ctx context.Context) (done bool, err error) {
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

// InstallOrUpdateOpreg will install or update OperandRegistry when Opreg CRD is existent
func (b *Bootstrap) InstallOrUpdateOpreg(installPlanApproval olmv1alpha1.Approval) error {

	if installPlanApproval != "" || b.CSData.ApprovalMode == string(olmv1alpha1.ApprovalManual) {
		if err := b.updateApprovalMode(); err != nil {
			return err
		}
	}

	// Read channel list from cpp configmap
	configMap := &corev1.ConfigMap{}
	err := b.Client.Get(context.TODO(), types.NamespacedName{
		Name:      constant.IBMCPPCONFIG,
		Namespace: b.CSData.ServicesNs,
	}, configMap)

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		klog.Infof("ConfigMap %s not found in namespace %s, using default values", constant.IBMCPPCONFIG, b.CSData.ServicesNs)
		configMap.Data = make(map[string]string)
	}

	var baseReg string
	registries := []string{
		constant.CSV4OpReg,
		constant.MongoDBOpReg,
		constant.IMOpReg,
		constant.IdpConfigUIOpReg,
		constant.PlatformUIOpReg,
		constant.KeyCloakOpReg,
		constant.CommonServicePGOpReg,
	}
	if b.SaasEnable {
		baseReg = constant.CSV3SaasOpReg
	} else {
		baseReg = constant.CSV3OpReg
	}

	concatenatedReg, err := constant.ConcatenateRegistries(baseReg, registries, b.CSData, configMap.Data)
	if err != nil {
		klog.Errorf("failed to concatenate OperandRegistry: %v", err)
		return err
	}

	if err := b.renderTemplate(concatenatedReg, b.CSData, true); err != nil {
		return err
	}
	return nil
}

// InstallOrUpdateOpcon will install or update OperandConfig when Opcon CRD is existent
func (b *Bootstrap) InstallOrUpdateOpcon(forceUpdateODLMCRs bool) error {

	var baseCon string
	configs := []string{
		constant.MongoDBOpCon,
		constant.IMOpCon,
		constant.UserMgmtOpCon,
		constant.IdpConfigUIOpCon,
		constant.PlatformUIOpCon,
		constant.EDBOpCon,
		constant.KeyCloakOpCon,
		constant.CommonServicePGOpCon,
	}

	baseCon = constant.CSV4OpCon

	concatenatedCon, err := constant.ConcatenateConfigs(baseCon, configs, b.CSData)
	if err != nil {
		klog.Errorf("failed to concatenate OperandConfig: %v", err)
		return err
	}

	if err := b.renderTemplate(concatenatedCon, b.CSData, forceUpdateODLMCRs); err != nil {
		return err
	}
	return nil
}

// InstallOrUpdateOpcon will install or update OperandConfig when Opcon CRD is existent
func (b *Bootstrap) InstallOrUpdateOperatorConfig(config string, forceUpdateODLMCRs bool) error {
	// clean up OperatorConfigs not in servicesNamespace every time function is called
	opts := []client.ListOption{
		client.MatchingLabels(
			map[string]string{constant.CsManagedLabel: "true"}),
	}
	operatorConfigList := b.ListOperatorConfig(ctx, opts...)
	if operatorConfigList != nil {
		for _, operatorConfig := range operatorConfigList.Items {
			if operatorConfig.Namespace != b.CSData.ServicesNs {
				if err := b.Client.Delete(ctx, &operatorConfig); err != nil {
					klog.Errorf("Failed to delete idle OperandConfig %s/%s which is managed by CS operator, but not in ServicesNamespace %s", operatorConfig.GetNamespace(), operatorConfig.GetName(), b.CSData.ServicesNs)
					return err
				}
				klog.Infof("Delete idle OperandConfig %s/%s which is managed by CS operator, but not in ServicesNamespace %s", operatorConfig.GetNamespace(), operatorConfig.GetName(), b.CSData.ServicesNs)
			}
		}
	}

	if err := b.renderTemplate(config, b.CSData, forceUpdateODLMCRs); err != nil {
		return err
	}

	return nil
}

// CreateNsScopeConfigmap creates nss configmap for operators
func (b *Bootstrap) CreateNsScopeConfigmap() error {
	cmRes := constant.NamespaceScopeConfigMap
	if err := b.renderTemplate(cmRes, b.CSData, false); err != nil {
		return err
	}
	return nil
}

// CreateEDBImageConfig creates a ConfigMap contains EDB image reference
func (b *Bootstrap) CreateEDBImageMaps() error {
	cmRes := constant.EDBImageConfigMap
	if err := b.renderTemplate(cmRes, b.CSData, false); err != nil {
		return err
	}
	return nil
}

// CreateKeycloakThemesConfigMap creates a ConfigMap contains Keycloak themes
func (b *Bootstrap) CreateKeycloakThemesConfigMap() error {

	klog.Info("Extracting Keycloak themes from jar file")
	themeFile := constant.KeycloakThemesJar
	themeFileContent, err := util.ReadFile(themeFile)
	if err != nil {
		return err
	}
	b.CSData.CloudPakThemes = util.EncodeBase64(themeFileContent)

	cmRes := constant.KeycloakThemesConfigMap
	if err := b.renderTemplate(cmRes, b.CSData, false); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) DeleteV3Resources(mutatingWebhooks, validatingWebhooks []string) error {

	// Delete the list of MutatingWebhookConfigurations
	for _, webhook := range mutatingWebhooks {
		if err := b.deleteResource(&admv1.MutatingWebhookConfiguration{}, webhook, "", "MutatingWebhookConfiguration"); err != nil {
			return err
		}
	}

	// Delete the list of ValidatingWebhookConfiguration
	for _, webhook := range validatingWebhooks {
		if err := b.deleteResource(&admv1.ValidatingWebhookConfiguration{}, webhook, "", "ValidatingWebhookConfiguration"); err != nil {
			return err
		}
	}

	if err := b.deleteWebhookResources(); err != nil {
		klog.Errorf("Error deleting webhook resources: %v", err)
	}

	if err := b.deleteSecretShareResources(); err != nil {
		klog.Errorf("Error deleting secretshare resources: %v", err)
	}
	return nil
}

// deleteWebhookResources deletes resources related to ibm-common-service-webhook
func (b *Bootstrap) deleteWebhookResources() error {
	exist, err := b.CheckCRD("operator.ibm.com/v1alpha1", "PodPreset")
	if err != nil {
		return err
	}
	if exist {
		// Delete PodPreset (CR)
		if err := b.deleteResource(&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "operator.ibm.com/v1alpha1",
				"kind":       "PodPreset",
			},
		}, constant.WebhookServiceName, b.CSData.ServicesNs, "PodPreset"); err != nil {
			return err
		}
	}
	// Delete ServiceAccount
	if err := b.deleteResource(&corev1.ServiceAccount{}, constant.WebhookServiceName, b.CSData.ServicesNs, "ServiceAccount"); err != nil {
		return err
	}

	// Delete Roles and RoleBindings
	if err := b.deleteResource(&rbacv1.Role{}, constant.WebhookServiceName, b.CSData.ServicesNs, "Role"); err != nil {
		return err
	}

	if err := b.deleteResource(&rbacv1.RoleBinding{}, constant.WebhookServiceName, b.CSData.ServicesNs, "RoleBinding"); err != nil {
		return err
	}

	if err := b.deleteResource(&rbacv1.ClusterRole{}, constant.WebhookServiceName, "", "ClusterRole"); err != nil {
		return err
	}

	if err := b.deleteResource(&rbacv1.ClusterRoleBinding{}, "ibm-common-service-webhook-"+b.CSData.ServicesNs, "", "ClusterRoleBinding"); err != nil {
		return err
	}

	// Delete Deployment
	if err := b.deleteResource(&appsv1.Deployment{}, constant.WebhookServiceName, b.CSData.ServicesNs, "Deployment"); err != nil {
		return err
	}

	return nil
}

// deleteSecretShareResources deletes resources related to secretshare
func (b *Bootstrap) deleteSecretShareResources() error {
	if err := b.deleteResource(&corev1.ServiceAccount{}, constant.Secretshare, b.CSData.ServicesNs, "ServiceAccount"); err != nil {
		return err
	}

	// Delete SecretShare ClusterRole and ClusterRoleBinding
	if err := b.deleteResource(&rbacv1.ClusterRole{}, constant.Secretshare, "", "ClusterRole"); err != nil {
		return err
	}

	if err := b.deleteResource(&rbacv1.ClusterRoleBinding{}, "secretshare-"+b.CSData.ServicesNs, "", "ClusterRoleBinding"); err != nil {
		return err
	}

	exist, err := b.CheckCRD("ibmcpcs.ibm.com/v1", "SecretShare")
	if err != nil {
		return err
	}
	if exist {
		// Delete SecretShare Operator CR
		if err := b.deleteResource(&ssv1.SecretShare{}, constant.MasterCR, b.CSData.ServicesNs, "SecretShare Operator CR"); err != nil {
			return err
		}
	}

	// Delete SecretShare Operator Deployment
	if err := b.deleteResource(&appsv1.Deployment{}, constant.Secretshare, b.CSData.ServicesNs, "Deployment"); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) deleteResource(resource client.Object, name, namespace string, resourceType string) error {
	namespacedName := types.NamespacedName{Name: name}
	if namespace != "" {
		namespacedName.Namespace = namespace
	}

	if err := b.Reader.Get(ctx, namespacedName, resource); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		klog.V(2).Infof("%s %s/%s not found, skipping deletion", resourceType, namespace, name)
		return nil
	}

	if err := b.Client.Delete(ctx, resource); err != nil {
		klog.Errorf("Failed to delete %s %s/%s: %v", resourceType, namespace, name, err)
		return err
	}

	klog.Infof("Successfully deleted %s %s/%s", resourceType, namespace, name)
	return nil
}

// CreateCsMaps will create a new common-service-maps configmap if not exists
func (b *Bootstrap) CreateCsMaps() error {

	var cmData util.CsMaps
	var newnsMapping util.NsMapping

	newnsMapping.RequestNs = append(newnsMapping.RequestNs, strings.Split(b.CSData.WatchNamespaces, ",")...)
	newnsMapping.CsNs = b.CSData.ServicesNs
	cmData.ControlNs = "cs-control"
	cmData.NsMappingList = append(cmData.NsMappingList, newnsMapping)
	commonServiceMap, error := utilyaml.Marshal(&cmData)
	if error != nil {
		klog.Errorf("failed to fetch data of configmap common-service-maps: %v", error)
	}

	data := make(map[string]string)
	data["common-service-maps.yaml"] = string(commonServiceMap)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "common-service-maps",
			Namespace: "kube-public",
		},
		Data: data,
	}

	if !(cm.Labels != nil && cm.Labels[constant.CsManagedLabel] == "true") {
		util.EnsureLabelsForConfigMap(cm, map[string]string{
			constant.CsManagedLabel: "true",
		})
	}

	if err := b.Client.Create(ctx, cm); err != nil {
		klog.Errorf("could not create common-service-map in kube-public: %v", err)
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

// deployResource deploys the given resource CR
func (b *Bootstrap) DeployResource(cr, placeholder string) bool {
	if err := utilwait.PollUntilContextCancel(ctx, time.Second*10, true, func(ctx context.Context) (done bool, err error) {
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

func (b *Bootstrap) CheckClusterType(ns string) (bool, error) {
	var isOCP bool
	dc := discovery.NewDiscoveryClientForConfigOrDie(b.Config)
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
		// check if the cluster is OCP by checking if the cluster has Infrastructure CR
		if apiList.GroupVersion == "config.openshift.io/v1" {
			for _, r := range apiList.APIResources {
				if r.Kind == "Infrastructure" {
					infraObj := &unstructured.Unstructured{}
					infraObj.SetGroupVersionKind(schema.GroupVersionKind{
						Group:   "config.openshift.io",
						Version: "v1",
						Kind:    "Infrastructure",
					})
					if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: "cluster"}, infraObj); err == nil {
						isOCP = true
					} else {
						klog.Errorf("Fail to get Infrastructure resource named cluster: %v", err)
					}
				}
			}
		}
	}
	klog.Infof("Cluster type is OCP: %v", isOCP)

	config := &corev1.ConfigMap{}
	if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: constant.IBMCPPCONFIG, Namespace: ns}, config); err != nil && !errors.IsNotFound(err) {
		return false, err
	} else if errors.IsNotFound(err) {
		if isOCP {
			return true, nil
		}
		klog.Errorf("Configmap %s/%s is required", ns, constant.IBMCPPCONFIG)
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
			klog.Errorf("cluster type isn't correct, kubernetes_cluster_type in configmap %s/%s is %s, but the cluster is %s environment", ns, constant.IBMCPPCONFIG, config.Data["kubernetes_cluster_type"], ocpCluster)
			return false, nil
		}

		klog.Info("cluster type is correct")
		return true, nil
	}
}

// 1. try to get cs-ca-certificate-secret
// 2. try to get cs-ca-certificate
// if we get secret but not get the cert, it is BYOC
func (b *Bootstrap) IsBYOCert() (bool, error) {
	klog.V(2).Info("Detect if it is BYO cert")
	secretName := "cs-ca-certificate-secret"
	secret := &corev1.Secret{}
	err := b.Client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: b.CSData.ServicesNs}, secret)
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}

	certList := &certmanagerv1.CertificateList{}
	opts := []client.ListOption{
		client.InNamespace(b.CSData.ServicesNs),
		client.MatchingLabels(
			map[string]string{"app.kubernetes.io/instance": "cs-ca-certificate"}),
	}
	if certerr := b.Reader.List(ctx, certList, opts...); certerr != nil {
		return false, certerr
	}

	if len(certList.Items) == 0 {
		return true, nil
	} else if len(certList.Items) == 1 {
		klog.V(2).Infof("found cs-ca-certificate, it is not BYOCertificate")
		return false, nil
	} else {
		return false, fmt.Errorf("found more than one cs-ca-certificate in namespace: %v, skip this", b.CSData.ServicesNs)
	}
}

func (b *Bootstrap) DeployCertManagerCR() error {
	for _, kind := range constant.CertManagerKinds {
		klog.Infof("Checking if resource %s CRD exsits ", kind)
		// if the crd is not exist, skip it
		exist, err := b.CheckCRD(constant.CertManagerAPIGroupVersionV1, kind)
		if err != nil {
			klog.Errorf("Failed to check resource with kind: %s, apiGroupVersion: %s", kind, constant.CertManagerAPIGroupVersionV1)
			return err
		}
		if !exist {
			klog.Infof("Skiped deploying %s, it is not exist in cluster", kind)
			return nil
		}
	}

	klog.V(2).Info("Fetch all the CommonService instances")
	csReq, err := labels.NewRequirement(constant.CsClonedFromLabel, selection.DoesNotExist, []string{})
	if err != nil {
		return err
	}
	csObjectList := &apiv3.CommonServiceList{}
	if err := b.Client.List(ctx, csObjectList, &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(*csReq),
	}); err != nil {
		return err
	}
	csList, err := util.ObjectListToNewUnstructuredList(csObjectList)
	if err != nil {
		return err
	}
	// If it is BYOCert
	isBYOC, err := b.IsBYOCert()
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

	if isBYOC {
		deployRootCert = false
		crWithBYOCert = "cs-ca-certificate-secret"
	}

	klog.Info("Deploying Cert Manager CRs")
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

	klog.Info("Checking Cert Manager Certs and Issuers deployment")
	if err := b.ConfigCertManagerOperandManagedByOperator(ctx); err != nil {
		return err
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

// CleanNamespaceScopeResources will delete the v3 NamespaceScopes resources and namespace scope operator
// NamespaceScope resources include common-service, nss-managedby-odlm, nss-odlm-scope, and odlm-scope-managedby-odlm
func (b *Bootstrap) CleanNamespaceScopeResources() error {

	// get namespace-scope ConfigMap in operatorNamespace
	nssCmNs, err := util.GetNssCmNs(b.Reader, b.CSData.OperatorNs)
	if err != nil {
		klog.Errorf("Failed to get %s configmap: %v", constant.NamespaceScopeConfigmapName, err)
		return err
	} else if nssCmNs == nil {
		klog.Infof("The %s configmap is not found in the %s namespace, skip cleaning the NamespaceScope resources", constant.NamespaceScopeConfigmapName, b.CSData.OperatorNs)
		return nil
	}

	// If the topology is (NOT ALL NS Mode) and (NOT Simple) , return
	if b.CSData.WatchNamespaces != "" && len(nssCmNs) > 1 {
		klog.Infof("The topology is not All Namespaces Mode or Simple Topology, skip cleaning the NamespaceScope resources")
		return nil
	}

	if isOpregAPI, err := b.CheckCRD(constant.OpregAPIGroupVersion, constant.OpregKind); err != nil {
		klog.Errorf("Failed to check if %s CRD exists: %v", constant.OpregKind, err)
		return err
	} else if !isOpregAPI {
		klog.Infof("%s CRD does not exist, skip checking no-op installMode", constant.OpregKind)
	} else if isOpregAPI {
		// Get the common-service OperandRegistry
		operandRegistry, err := b.GetOperandRegistry(ctx, constant.MasterCR, b.CSData.ServicesNs)
		if err != nil {
			klog.Errorf("Failed to get common-service OperandRegistry: %v", err)
			return err
		} else if operandRegistry == nil {
			klog.Infof("The common-service OperandRegistry is not found in the %s namespace, skip cleaning the NamespaceScope resources", b.CSData.ServicesNs)
			return nil
		}

		// Check if there is v4 OperandRegistry exists
		if operandRegistry.Annotations != nil {
			if v1IsLarger, convertErr := util.CompareVersion("4.0.0", operandRegistry.Annotations["version"]); convertErr != nil {
				klog.Errorf("Failed to convert version for OperandRegistry: %v", convertErr)
				return convertErr
			} else if v1IsLarger {
				klog.Infof("The OperandRegistry's version %v is smaller than 4.0.0, skip cleaning the NamespaceScope resources", operandRegistry.Annotations["version"])
				return nil
			}
		}
		// List all requested operators
		if operandRegistry.Status.OperatorsStatus != nil {
			for operator := range operandRegistry.Status.OperatorsStatus {
				// If there is a requested operator's installMode is "no-op", then skip call delete function
				for _, op := range operandRegistry.Spec.Operators {
					if op.Name == operator && op.InstallMode == "no-op" {
						klog.Infof("The operator %s with 'no-op' installMode is still requested in OperandRegistry, skip cleaning the NamespaceScope resources", operator)
						return nil
					}
				}
			}
		}
	}

	// Delete v3 Namespace Scope operator
	sub := &olmv1alpha1.Subscription{}
	if err := b.Client.Get(ctx, types.NamespacedName{Name: constant.NsSubName, Namespace: b.CSData.ServicesNs}, sub); err == nil {
		if strings.HasPrefix(sub.Spec.Channel, "v4.") {
			klog.Infof("The %s subscription is in the v4.x channel, skip cleaning up", constant.NsSubName)
			return nil
		}

		klog.Info("Cleaning NamespaceScope resources in Simple Topology or All Namespaces Mode")
		klog.Infof("Uninstall v3 Namespace Scope operator in servicesNamespace %s when the topology is Simple or All Namespaces Mode", b.CSData.ServicesNs)
		if err := b.DeleteOperator(constant.NsSubName, b.CSData.ServicesNs); err != nil {
			klog.Errorf("Failed to uninstall v3 Namespace Scope operator in servicesNamespace %s", b.CSData.ServicesNs)
			return err
		}
	} else {
		if !errors.IsNotFound(err) {
			klog.Errorf("Failed to get %s subscription in namespace %s: %v", constant.NsSubName, b.CSData.ServicesNs, err)
			return err
		}
		klog.Infof("The %s subscription is not found in the %s namespace, skip cleaning up", constant.NsSubName, b.CSData.ServicesNs)
	}

	// Patch and remove the ownerReference in the namespace-scope configmap if it exist
	if nssCm, err := util.GetCmOfNss(b.Reader, b.CSData.OperatorNs); err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("The %s configmap is not found in the %s namespace, skip patching ownerReference", constant.NamespaceScopeConfigmapName, b.CSData.OperatorNs)
		} else {
			klog.Errorf("Failed to get %s configmap: %v", constant.NamespaceScopeConfigmapName, err)
			return err
		}
	} else {
		if len(nssCm.OwnerReferences) > 0 {
			klog.Infof("Remove the ownerReference in the %s configmap", constant.NamespaceScopeConfigmapName)
			// Patch and remove the ownerReference in the namespace-scope configmap in data section
			originalCm := nssCm.DeepCopy()
			nssCm.OwnerReferences = nil
			if err := b.Client.Patch(context.TODO(), nssCm, client.MergeFrom(originalCm)); err != nil {
				klog.Errorf("Failed to patch and remove the ownerReference in the %s configmap", constant.NamespaceScopeConfigmapName)
				return err
			}
		}
	}

	// Delete NamespaceScope CRs and wait for those are deleted exactly, if time is out for deleting the CRs, then proceed to delete the operator
	// Check if the NamespaceScope CRD is existent
	exist, err := b.CheckCRD(constant.NssAPIVersion, constant.NssKindCR)
	if err != nil {
		klog.Errorf("Failed to check resource with kind: %s, apiGroupVersion: %s", constant.NssKindCR, constant.NssAPIVersion)
		return err
	}
	if !exist {
		klog.Infof("Skiped deleting NamespaceScope CRs, it is not exist in cluster")
		return nil
	}

	nssCRsList, err := b.ListNssCRs(ctx, b.CSData.ServicesNs)
	if len(nssCRsList.Items) > 0 && err == nil {
		for _, nssCR := range nssCRsList.Items {
			if err := b.Client.Delete(context.TODO(), &nssCR); err != nil {
				klog.Errorf("Failed to delete NamespaceScope CR %s: %v", nssCR.Name, err)
			}
		}

		klog.Infof("Waiting for the NamespaceScope CRs to be deleted in the %s namespace", b.CSData.ServicesNs)
		if err := utilwait.PollUntilContextTimeout(ctx, time.Second*5, time.Second*30, true, func(ctx context.Context) (done bool, err error) {
			nssCRsList, err := b.ListNssCRs(ctx, b.CSData.ServicesNs)
			if err != nil {
				return false, err
			}
			if len(nssCRsList.Items) > 0 {
				allDeleted := true
				for _, nssCR := range nssCRsList.Items {
					if nssCR.GetDeletionTimestamp() == nil {
						allDeleted = false
						break
					}
				}
				if !allDeleted {
					// At least one NSS resource doesn't have deletion timestamp set
					return false, nil
				}
				// Deletion timestamp set for all Nss resources
				return true, errors.NewResourceExpired("All NSS CRs are ready to be deleted.")
			}
			// No NSS resources found
			return len(nssCRsList.Items) == 0, nil
		}); err != nil {
			klog.Infof("Patch finalizers to delete the NamespaceScope CRs")
			nssCRsList, err := b.ListNssCRs(ctx, b.CSData.ServicesNs)
			if err != nil {
				return err
			}
			for _, nssCR := range nssCRsList.Items {
				if nssCR.GetDeletionTimestamp() != nil && len(nssCR.ObjectMeta.Finalizers) > 0 {
					originalCopy := nssCR.DeepCopy()
					if change := apiv3.RemoveFinalizer(&nssCR.ObjectMeta, constant.NssCRFinalizer); change {
						if err := b.Client.Patch(context.TODO(), &nssCR, client.MergeFrom(originalCopy)); err != nil {
							klog.Errorf("Failed to patch finalizers to delete the NamespaceScope CR %s: %v", nssCR.Name, err)
							return err
						}
						klog.Infof("Rmoved finalizers to delete the NamespaceScope CR %s", nssCR.Name)
					}
				}
			}
		}
	}
	return nil
}

func (b *Bootstrap) Cleanup(operatorNs string, resource *Resource) error {
	// Check if CRD exist
	APIGroupVersion := resource.Group + "/" + resource.Version
	exist, err := b.CheckCRD(APIGroupVersion, resource.Kind)
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

func (b *Bootstrap) CheckDeployStatus(ctx context.Context) (operatorDeployed bool, servicesDeployed bool) {
	if opreg, err := b.GetOperandRegistry(ctx, "common-service", b.CSData.ServicesNs); err == nil && opreg != nil && opreg.Status.Phase == odlm.RegistryRunning {
		operatorDeployed = true
	}

	if opconfig, err := b.GetOperandConfig(ctx, "common-service", b.CSData.ServicesNs); err == nil && opconfig != nil && opconfig.Status.Phase == odlm.ServiceRunning {
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

	operatorConfigList := b.ListOperatorConfig(ctx, opts...)
	if operatorConfigList != nil {
		for _, operatorConfig := range operatorConfigList.Items {
			if operatorConfig.Namespace != b.CSData.ServicesNs {
				if err := b.Client.Delete(ctx, &operatorConfig); err != nil {
					klog.Errorf("Failed to delete idle OperandConfig %s/%s which is managed by CS operator, but not in ServicesNamespace %s", operatorConfig.GetNamespace(), operatorConfig.GetName(), b.CSData.ServicesNs)
					return err
				}
				klog.Infof("Delete idle OperandConfig %s/%s which is managed by CS operator, but not in ServicesNamespace %s", operatorConfig.GetNamespace(), operatorConfig.GetName(), b.CSData.ServicesNs)
			}
		}
	}

	return nil
}

func (b *Bootstrap) ConfigCertManagerOperandManagedByOperator(ctx context.Context) error {
	opts := []client.ListOption{
		client.MatchingLabels(
			map[string]string{constant.CsManagedLabel: "true"}),
	}
	// Delete idle Cert Manager CRs which are managed by CS operator, but not in ServicesNamespace
	certsList := b.ListCerts(ctx, opts...)
	if certsList != nil {
		for _, cert := range certsList.Items {
			if cert.Namespace != b.CSData.ServicesNs {
				if err := b.Client.Delete(ctx, &cert); err != nil {
					klog.Errorf("Failed to delete idle Cert Manager Certificate %s/%s which is managed by CS operator, but not in ServicesNamespace %s", cert.GetNamespace(), cert.GetName(), b.CSData.ServicesNs)
					return err
				}
				klog.Infof("Deleted idle Cert Manager Certificate %s/%s which is managed by CS operator, but not in ServicesNamespace %s", cert.GetNamespace(), cert.GetName(), b.CSData.ServicesNs)

			}
		}
	}

	// Delete idle Cert Manager CRs which are managed by CS operator, but not in ServicesNamespace
	issuerList := b.ListIssuer(ctx, opts...)
	if issuerList != nil {
		for _, issuer := range issuerList.Items {
			if issuer.Namespace != b.CSData.ServicesNs {
				if err := b.Client.Delete(ctx, &issuer); err != nil {
					klog.Errorf("Failed to delete idle Cert Manager Issuer %s/%s which is managed by CS operator, but not in ServicesNamespace %s", issuer.GetNamespace(), issuer.GetName(), b.CSData.ServicesNs)
					return err
				}
				klog.Infof("Deleted idle Cert Manager Issuer %s/%s which is managed by CS operator, but not in ServicesNamespace %s", issuer.GetNamespace(), issuer.GetName(), b.CSData.ServicesNs)
			}
		}
	}

	watchNamespaceList := strings.Split(b.CSData.WatchNamespaces, ",")
	secretName := "cs-ca-certificate-secret"
	if len(watchNamespaceList) > 1 {
		for _, watchNamespace := range watchNamespaceList {
			secret := &corev1.Secret{}
			err := b.Client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: watchNamespace}, secret)
			if err != nil && !errors.IsNotFound(err) {
				return nil
			} else if errors.IsNotFound(err) {
				continue
			} else {
				if watchNamespace != b.CSData.ServicesNs {
					if err := b.Client.Delete(ctx, secret); err != nil {
						klog.Errorf("Failed to delete cs ca certificate secret %s/%s not in ServicesNamespace %s", secret.GetNamespace(), secret.GetName(), b.CSData.ServicesNs)
						return err
					}
					klog.Infof("Deleted cs ca certificate secret %s/%s not in ServicesNamespace %s", secret.GetNamespace(), secret.GetName(), b.CSData.ServicesNs)
				}
			}
		}
	}

	return nil
}

func (b *Bootstrap) PropagateDefaultCR(instance *apiv3.CommonService) error {
	// Copy Master CR into namespace in WATCH_NAMESPACE list
	watchNamespaceList := strings.Split(b.CSData.WatchNamespaces, ",")
	// Exclude CommonService cloned in AllNamespace Mode
	if len(watchNamespaceList) > 1 {
		// Get the unstructured object of the main CommonService CR
		mainCsInstance := &unstructured.Unstructured{}
		mainCsInstance.SetGroupVersionKind(apiv3.GroupVersion.WithKind("CommonService"))
		if err := b.Client.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, mainCsInstance); err != nil {
			return fmt.Errorf("failed to get CommonService CR %s in namespace %s: %v", instance.Name, instance.Namespace, err)
		}
		csLabel := make(map[string]string)
		// Copy from the original labels to the target labels
		for k, v := range mainCsInstance.GetLabels() {
			csLabel[k] = v
		}
		csLabel[constant.CsClonedFromLabel] = b.CSData.OperatorNs

		csAnnotation := make(map[string]string)
		// Copy from the original Annotations to the target Annotations
		for k, v := range mainCsInstance.GetAnnotations() {
			csAnnotation[k] = v
		}
		for _, watchNamespace := range watchNamespaceList {
			if watchNamespace == mainCsInstance.GetNamespace() {
				continue
			}
			copiedCsInstance := &unstructured.Unstructured{}
			copiedCsInstance.SetGroupVersionKind(apiv3.GroupVersion.WithKind("CommonService"))
			copiedCsInstance.SetNamespace(watchNamespace)
			copiedCsInstance.SetName(constant.MasterCR)
			copiedCsInstance.SetLabels(csLabel)
			copiedCsInstance.SetAnnotations(csAnnotation)
			copiedCsInstance.Object["spec"] = mainCsInstance.Object["spec"]

			if err := b.Client.Create(ctx, copiedCsInstance); err != nil {
				if errors.IsAlreadyExists(err) {
					csKey := types.NamespacedName{Name: constant.MasterCR, Namespace: watchNamespace}
					existingCsInstance := &unstructured.Unstructured{}
					existingCsInstance.SetGroupVersionKind(apiv3.GroupVersion.WithKind("CommonService"))
					if err := b.Client.Get(ctx, csKey, existingCsInstance); err != nil {
						return fmt.Errorf("failed to get cloned CommonService CR in namespace %s: %v", watchNamespace, err)
					}
					if needUpdate := util.CompareObj(copiedCsInstance, existingCsInstance); needUpdate {
						copiedCsInstance.SetResourceVersion(existingCsInstance.GetResourceVersion())
						if err := b.Client.Update(ctx, copiedCsInstance); err != nil {
							return fmt.Errorf("failed to update cloned CommonService CR in namespace %s: %v", watchNamespace, err)
						}
					}
				} else {
					return fmt.Errorf("failed to create cloned CommonService CR in namespace %s: %v", watchNamespace, err)
				}
			}
		}
	}
	return nil
}

func IdentifyCPFSNs(r client.Reader, operatorNs string) (string, error) {
	csKey := types.NamespacedName{Name: constant.MasterCR, Namespace: operatorNs}
	csCR := &apiv3.CommonService{}
	if err := r.Get(context.TODO(), csKey, csCR); err != nil && !errors.IsNotFound(err) {
		return operatorNs, err
	} else if errors.IsNotFound(err) {
		return operatorNs, nil
	}
	// Assign .spec.operatorNamespace from existing default CommonSerivce CR to CPFSNs

	cpfsNs := csCR.Spec.OperatorNamespace
	if csCR.Status.ConfigStatus.OperatorNamespace != "" {
		cpfsNs = csCR.Status.ConfigStatus.OperatorNamespace
	}
	return string(cpfsNs), nil
}

func (b *Bootstrap) PropagateCPPConfig(instance *corev1.ConfigMap) error {
	// Copy Master CR into namespace in WATCH_NAMESPACE list
	watchNamespaceList := strings.Split(b.CSData.WatchNamespaces, ",")

	// Do not copy ibm-cpp-config in AllNamespace Mode
	if len(watchNamespaceList) > 1 {
		for _, ns := range watchNamespaceList {
			if ns == instance.Namespace {
				continue
			}
			copiedCPPConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      constant.IBMCPPCONFIG,
					Namespace: ns,
					Labels:    instance.GetLabels(),
				},
				Data: instance.Data,
			}

			if err := b.Client.Create(ctx, copiedCPPConfigMap); err != nil {
				if errors.IsAlreadyExists(err) {
					cmKey := types.NamespacedName{Name: constant.IBMCPPCONFIG, Namespace: ns}
					existingCM := &corev1.ConfigMap{}
					if err := b.Reader.Get(ctx, cmKey, existingCM); err != nil {
						return fmt.Errorf("failed to get %s ConfigMap in namespace %s: %v", constant.IBMCPPCONFIG, ns, err)
					}
					for k, v := range existingCM.Data {
						if _, ok := copiedCPPConfigMap.Data[k]; !ok {
							copiedCPPConfigMap.Data[k] = v
						}
					}
					if !reflect.DeepEqual(copiedCPPConfigMap.Data, existingCM.Data) || !reflect.DeepEqual(copiedCPPConfigMap.Labels, existingCM.Labels) {
						copiedCPPConfigMap.SetResourceVersion(existingCM.GetResourceVersion())
						if err := b.Client.Update(ctx, copiedCPPConfigMap); err != nil {
							return fmt.Errorf("failed to update %s ConfigMap in namespace %s: %v", constant.IBMCPPCONFIG, ns, err)
						}
						klog.Infof("Global CPP config %s/%s is updated", ns, constant.IBMCPPCONFIG)
					}
				} else {
					return fmt.Errorf("failed to create cloned %s ConfigMap in namespace %s: %v", constant.IBMCPPCONFIG, ns, err)
				}
			} else {
				klog.Infof("Global CPP config %s/%s is propagated to namespace %s", b.CSData.ServicesNs, constant.IBMCPPCONFIG, ns)
			}
		}
	}
	return nil
}

func (b *Bootstrap) CleanupWebhookResources() error {
	validatingWebhookConfiguration := Resource{
		Name:    "ibm-common-service-validating-webhook-" + b.CSData.OperatorNs,
		Version: "v1",
		Group:   "admissionregistration.k8s.io",
		Kind:    "ValidatingWebhookConfiguration",
		Scope:   "clusterScope",
	}

	mutatingWebhookConfiguration := Resource{
		Name:    "ibm-operandrequest-webhook-configuration-" + b.CSData.OperatorNs,
		Version: "v1",
		Group:   "admissionregistration.k8s.io",
		Kind:    "MutatingWebhookConfiguration",
		Scope:   "clusterScope",
	}

	webhookService := Resource{
		Name:    "webhook-service",
		Version: "v1",
		Group:   "",
		Kind:    "Service",
		Scope:   "namespaceScope",
	}
	// cleanup old webhookconfigurations and services
	if err := b.Cleanup(b.CSData.OperatorNs, &validatingWebhookConfiguration); err != nil {
		klog.Errorf("Failed to cleanup validatingWebhookConfig: %v", err)
		return err
	}

	if err := b.Cleanup(b.CSData.OperatorNs, &mutatingWebhookConfiguration); err != nil {
		klog.Errorf("Failed to cleanup mutatingWebhookConfiguration: %v", err)
		return err
	}

	if err := b.Cleanup(b.CSData.OperatorNs, &webhookService); err != nil {
		klog.Errorf("Failed to cleanup webhookService: %v", err)
		return err
	}
	return nil
}

func (b *Bootstrap) UpdateResourceLabel(instance *apiv3.CommonService) error {
	labelsMap := make(map[string]string)
	// Fetch all the CommonService instances
	csReq, err := labels.NewRequirement(constant.CsClonedFromLabel, selection.DoesNotExist, []string{})
	if err != nil {
		return err
	}
	csObjectList := &apiv3.CommonServiceList{}
	if err := b.Client.List(ctx, csObjectList, &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(*csReq),
	}); err != nil {
		return err
	}
	csObjectList.Items = append(csObjectList.Items, *instance)

	// get spec.labels in the spec
	for _, cs := range csObjectList.Items {
		labels := cs.Spec.Labels
		maps.Copy(labelsMap, labels)
	}

	if len(labelsMap) == 0 {
		return nil
	}

	// Update labels in the CommonService CRs
	klog.Infof("Update labels for resources managed by CommonService CR %s/%s", instance.GetNamespace(), instance.GetName())
	for _, cs := range csObjectList.Items {
		util.EnsureLabelsForCsCR(&cs, labelsMap)
		if err := b.Client.Update(context.TODO(), &cs); err != nil {
			klog.Errorf("Failed to update label in commonservice cr:%v, %v", cs.GetName(), err)
			return err
		}
	}

	// update labels in the configmap
	nsscmName := "namespace-scope"
	cmList := &corev1.ConfigMapList{}

	cm := &corev1.ConfigMap{}
	if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: nsscmName, Namespace: b.CSData.OperatorNs}, cm); err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		klog.V(3).Infof("configmap %s is not found in namespace: %s", nsscmName, b.CSData.OperatorNs)
	} else {
		cmList.Items = append(cmList.Items, *cm)
	}
	cmUnstructedList, err := util.ObjectListToNewUnstructuredList(cmList)
	if err != nil {
		return err
	}
	if err := b.UpdateResourceWithLabel(cmUnstructedList, labelsMap); err != nil {
		return err
	}

	// Update labels in the OperandConfig and OperandRegistry
	opconfigList := &odlm.OperandConfigList{}
	opcon := &odlm.OperandConfig{}
	if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: "common-service", Namespace: b.CSData.ServicesNs}, opcon); err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		klog.V(3).Infof("OperandConfig common-service is not found in namespace: %s", b.CSData.ServicesNs)
	}
	opconfigList.Items = append(opconfigList.Items, *opcon)
	opconUnstructedList, err := util.ObjectListToNewUnstructuredList(opconfigList)
	if err != nil {
		return err
	}
	if err := b.UpdateResourceWithLabel(opconUnstructedList, labelsMap); err != nil {
		return err
	}
	opregList := &odlm.OperandRegistryList{}
	opreg := &odlm.OperandRegistry{}
	if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: "common-service", Namespace: b.CSData.ServicesNs}, opreg); err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		klog.V(3).Infof("OperandRegistry common-service is not found in namespace: %s", b.CSData.ServicesNs)
	} else {
		opregList.Items = append(opregList.Items, *opreg)
	}
	opregUnstructedList, err := util.ObjectListToNewUnstructuredList(opregList)
	if err != nil {
		return err
	}
	if err := b.UpdateResourceWithLabel(opregUnstructedList, labelsMap); err != nil {
		return err
	}

	// update labels in the Issuer
	issuerList := &certmanagerv1.IssuerList{}
	issuerNames := []string{"cs-ss-issuer", "cs-ca-issuer"}
	for _, issuerName := range issuerNames {
		issuer := &certmanagerv1.Issuer{}
		if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: issuerName, Namespace: b.CSData.ServicesNs}, issuer); err != nil && !errors.IsNotFound(err) {
			return err
		} else if errors.IsNotFound(err) {
			klog.V(3).Infof("Issuer %s is not found in namespace: %s", issuerName, b.CSData.ServicesNs)
		} else {
			issuerList.Items = append(issuerList.Items, *issuer)
		}
	}
	issuerUnstructedList, err := util.ObjectListToNewUnstructuredList(issuerList)
	if err != nil {
		return err
	}

	if err := b.UpdateResourceWithLabel(issuerUnstructedList, labelsMap); err != nil {
		return err
	}

	// update labels in the Certificate
	certList := &certmanagerv1.CertificateList{}
	cert := &certmanagerv1.Certificate{}
	if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: "cs-ca-certificate", Namespace: b.CSData.ServicesNs}, cert); err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		klog.V(3).Infof("certificate cs-ca-certificate is not found in namespace: %s", b.CSData.ServicesNs)
	} else {
		certList.Items = append(certList.Items, *cert)
	}
	certUnstructedList, err := util.ObjectListToNewUnstructuredList(certList)
	if err != nil {
		return err
	}
	if err := b.UpdateResourceWithLabel(certUnstructedList, labelsMap); err != nil {
		return err
	}

	return nil
}

func (b *Bootstrap) UpdateResourceWithLabel(resources *unstructured.UnstructuredList, labels map[string]string) error {
	for _, resource := range resources.Items {
		if resource.GetName() == "" {
			continue
		}
		util.EnsureLabels(&resource, labels)
		klog.Infof("Updating labels in %s %s/%s", resource.GetKind(), resource.GetNamespace(), resource.GetName())
		if err := b.UpdateObject(&resource); err != nil {
			klog.Errorf("Failed to update label in kind:%v namespace/name:%v/%v, %v", resource.GetKind(), resource.GetNamespace(), resource.GetName(), err)
			return err
		}
	}
	return nil
}

func (b *Bootstrap) UpdateManageCertRotationLabel(instance *apiv3.CommonService) error {

	// Fetch all the CommonService instances
	csReq, err := labels.NewRequirement(constant.CsClonedFromLabel, selection.DoesNotExist, []string{})
	if err != nil {
		return err
	}
	csObjectList := &apiv3.CommonServiceList{}
	if err := b.Client.List(ctx, csObjectList, &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(*csReq),
	}); err != nil {
		return err
	}
	csObjectList.Items = append(csObjectList.Items, *instance)

	// if EnableManageCertRotation is set to false in any cs cr,
	// we set the value of this label to 'no'
	// otherwise, set the value of this label to 'yes'
	certLabel := make(map[string]string)
	for _, cs := range csObjectList.Items {
		if cs.Spec.DisableManageCertRotation {
			certLabel[constant.ManageCertRotationLabel] = "false"
			break
		}
		certLabel[constant.ManageCertRotationLabel] = "true"
	}

	// update labels in the Certificate
	certList := &certmanagerv1.CertificateList{}
	cert := &certmanagerv1.Certificate{}
	if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: "cs-ca-certificate", Namespace: b.CSData.ServicesNs}, cert); err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		klog.V(3).Infof("certificate cs-ca-certificate is not found in namespace: %s", b.CSData.ServicesNs)
	} else {
		certList.Items = append(certList.Items, *cert)
	}
	certUnstructedList, err := util.ObjectListToNewUnstructuredList(certList)
	if err != nil {
		return err
	}
	if err := b.UpdateResourceWithLabel(certUnstructedList, certLabel); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) UpdateEDBUserManaged() error {
	operatorNamespace, err := util.GetOperatorNamespace()
	if err != nil {
		return err
	}
	defaultCsCR := &apiv3.CommonService{}
	csName := "common-service"
	if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: csName, Namespace: operatorNamespace}, defaultCsCR); err != nil {
		return err
	}
	servicesNamespace := string(defaultCsCR.Spec.ServicesNamespace)

	config := &corev1.ConfigMap{}
	if err := b.Client.Get(context.TODO(), types.NamespacedName{Name: constant.IBMCPPCONFIG, Namespace: servicesNamespace}, config); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	userManaged := config.Data["EDB_USER_MANAGED_OPERATOR_ENABLED"]
	if userManaged != "true" {
		unsetEDBUserManaged(defaultCsCR)
	} else {
		setEDBUserManaged(defaultCsCR)
	}

	if err := b.Client.Update(context.TODO(), defaultCsCR); err != nil {
		return err
	}
	return nil
}

func unsetEDBUserManaged(instance *apiv3.CommonService) {
	if instance.Spec.OperatorConfigs == nil {
		return
	}
	for i := range instance.Spec.OperatorConfigs {
		i := i
		if instance.Spec.OperatorConfigs[i].Name == "internal-use-only-edb" {
			instance.Spec.OperatorConfigs[i].UserManaged = false
		}
	}
}

func setEDBUserManaged(instance *apiv3.CommonService) {
	if instance.Spec.OperatorConfigs == nil {
		instance.Spec.OperatorConfigs = []apiv3.OperatorConfig{}
	}
	isExist := false
	for i := range instance.Spec.OperatorConfigs {
		i := i
		if instance.Spec.OperatorConfigs[i].Name == "internal-use-only-edb" {
			instance.Spec.OperatorConfigs[i].UserManaged = true
			isExist = true
		}
	}
	if !isExist {
		instance.Spec.OperatorConfigs = append(instance.Spec.OperatorConfigs, apiv3.OperatorConfig{Name: "internal-use-only-edb", UserManaged: true})
	}
}

func (b *Bootstrap) waitOperatorCSV(subName, packageManifest, operatorNs string) (bool, error) {
	var isWaiting bool
	// Wait for the operator CSV to be installed
	klog.Infof("Waiting for the operator CSV with packageManifest %s in namespace %s to be installed", packageManifest, operatorNs)
	if err := utilwait.PollUntilContextTimeout(ctx, time.Second*5, time.Minute*5, true, func(ctx context.Context) (done bool, err error) {
		installed, err := b.checkOperatorCSV(subName, packageManifest, operatorNs)
		if err != nil {
			return false, err
		} else if !installed {
			klog.Infof("The operator CSV with packageManifest %s in namespace %s is not installed yet", packageManifest, operatorNs)
			isWaiting = true
		}
		return installed, nil
	}); err != nil {
		return isWaiting, fmt.Errorf("failed to wait for the operator CSV to be installed: %v", err)
	}
	return isWaiting, nil
}

func (b *Bootstrap) checkOperatorCSV(subName, packageManifest, operatorNs string) (bool, error) {
	// Get the subscription by name and namespace
	sub, err := b.fetchSubscription(subName, packageManifest, operatorNs)
	if err != nil {
		klog.Errorf("Failed to get Subscription %s in namespace %s: %v", subName, operatorNs, err)
		return false, err
	}

	// Get the channel in the subscription .spec.channel, and check if it is semver
	channel := sub.Spec.Channel
	if !semver.IsValid(channel) {
		klog.Warningf("channel %s is not a semver for operator with packageManifest %s and operatorNs %s", channel, packageManifest, operatorNs)
		return false, nil
	}

	// Get the CSV from subscription .status.installedCSV
	installedCSV := sub.Status.InstalledCSV
	var installedVersion string
	if installedCSV != "" {
		// installedVersion is the version after the first dot in installedCSV
		// For example, version is v4.3.1 for operand-deployment-lifecycle-manager.v4.3.1
		installedVersion = installedCSV[strings.IndexByte(installedCSV, '.')+1:]
	}

	// 0 if channel == installedVersion - v4.3 == v4.3.0
	// -1 if channel < installedVersion - v4.3 < v4.3.1
	// +1 if channel > installedVersion - v4.3 > v4.2.0
	if semver.Compare(channel, installedVersion) > 0 {
		return false, nil
	}

	return true, nil
}

// CheckSubOperatorStatus checks the status of the sub-operator by listing the OperandRegistry
func (b *Bootstrap) CheckSubOperatorStatus(instance *apiv3.CommonService) (bool, error) {
	// Get the common-service OperandRegistry
	operandRegistry, err := b.GetOperandRegistry(ctx, constant.MasterCR, b.CSData.ServicesNs)
	if err != nil || operandRegistry == nil {
		klog.Errorf("Failed to get common-service OperandRegistry: %v", err)
		return false, err

	}
	var operatorSlice []apiv3.BedrockOperator

	if operandRegistry.Status.Phase == odlm.RegistryReady || operandRegistry.Status.OperatorsStatus == nil {
		klog.Infof("There is no service installed yet from the OperandRegistry %s/%s , skipping checking the operator status", operandRegistry.GetNamespace(), operandRegistry.GetName())
		instance.Status.BedrockOperators = operatorSlice
		instance.Status.OverallStatus = ""
		return false, nil
	}
	for opt := range operandRegistry.Status.OperatorsStatus {
		operator, err := b.GetOperatorInfo(operandRegistry.Spec.Operators, opt)
		if err != nil {
			klog.Errorf("Failed to get operator %s info: %v", opt, err)
			return false, err
		}
		if operator == nil {
			klog.Infof("The operator %s is in no-op mode, skipping it", opt)
			continue
		}
		optStatus, err := b.setOperatorStatus(instance, operator.Name, operator.PackageName, operator.Namespace)
		if err != nil {
			klog.Errorf("Failed to get operator status: %v", err)
			return false, err
		}
		// Only optStatus append into the operatorSlice if the optStatus.name is not duplicated
		// Otherwise overwrite the existing one
		if len(operatorSlice) == 0 {
			operatorSlice = append(operatorSlice, optStatus)
		} else {
			for i, opt := range operatorSlice {
				if opt.Name == optStatus.Name {
					operatorSlice[i] = optStatus
					break
				}
				if i == len(operatorSlice)-1 {
					operatorSlice = append(operatorSlice, optStatus)
				}
			}
		}
	}
	instance.Status.BedrockOperators = operatorSlice

	instance.Status.OverallStatus = apiv3.CRSucceeded
	for _, opt := range operatorSlice {
		if opt.OperatorStatus != apiv3.CRSucceeded {
			instance.Status.OverallStatus = apiv3.CRNotReady
			break
		}
	}
	if instance.Status.OverallStatus == apiv3.CRNotReady {
		return false, fmt.Errorf("the operator overall status is not ready")
	}
	return true, nil
}

func (b *Bootstrap) GetOperatorInfo(optList []odlm.Operator, optName string) (*odlm.Operator, error) {
	for _, opt := range optList {
		if opt.Name == optName && opt.InstallMode != "no-op" { // If the operator is no-op mode, skip it
			return &opt, nil
		}
		if opt.Name == optName && opt.InstallMode == "no-op" {
			return nil, nil
		}
	}
	return nil, fmt.Errorf("operator %s not found in OperandRegistry", optName)
}

func (b *Bootstrap) setOperatorStatus(instance *apiv3.CommonService, name, packageManifest, namespace string) (apiv3.BedrockOperator, error) {
	var opt apiv3.BedrockOperator
	opt.Name = name

	sub, err := b.fetchSubscription(name, packageManifest, namespace)
	if err != nil {
		klog.Errorf("Failed to get Subscription %s in namespace %s: %v", name, namespace, err)
		return opt, err
	}

	installedCSV := sub.Status.InstalledCSV
	if installedCSV != "" {
		opt.Name = installedCSV[:strings.IndexByte(installedCSV, '.')]
		opt.Version = installedCSV[strings.IndexByte(installedCSV, '.')+1:]

		csv := &olmv1alpha1.ClusterServiceVersion{}
		csvKey := types.NamespacedName{
			Name:      installedCSV,
			Namespace: namespace,
		}
		if err := b.Reader.Get(context.TODO(), csvKey, csv); err != nil {
			klog.Errorf("Failed to get %s CSV: %s", name, err)
			opt.OperatorStatus = apiv3.CRNotReady
		} else {
			if len(csv.Status.Conditions) > 0 {
				csvStatus := csv.Status.Conditions[len(csv.Status.Conditions)-1].Phase
				opt.OperatorStatus = fmt.Sprintf("%v", csvStatus)
			} else {
				opt.OperatorStatus = apiv3.CRNotReady
			}
		}
	} else {
		klog.Warningf("Failed to get installed CSV for Subscription %s in namespace %s", name, namespace)
		opt.OperatorStatus = apiv3.CRNotReady
	}

	// fetch installplanName
	installplanName := ""
	if sub.Status.Install != nil {
		installplanName = sub.Status.Install.Name
	}
	opt.InstallPlanName = installplanName

	// determinate subscription status
	if installplanName == "" {
		opt.SubscriptionStatus = apiv3.CRFailed
		opt.InstallPlanName = "Not Found"
	} else {
		currentCSV := sub.Status.CurrentCSV
		if installedCSV == currentCSV && installedCSV != "" {
			opt.SubscriptionStatus = apiv3.CRSucceeded
		} else {
			opt.SubscriptionStatus = fmt.Sprintf("%v", sub.Status.State)
		}
	}

	if opt.OperatorStatus == "" || opt.OperatorStatus != apiv3.CRSucceeded || opt.SubscriptionStatus == "" || opt.SubscriptionStatus != apiv3.CRSucceeded {
		opt.Troubleshooting = "Operator status is not healthy, please check " + constant.GeneralTroubleshooting + " for more information"
	}

	if opt.SubscriptionStatus == "" || opt.SubscriptionStatus != apiv3.CRSucceeded {
		b.EventRecorder.Eventf(instance, "Warning", "Bedrock Operator Failed", "Subscription %s/%s is not healthy, please check troubleshooting document %s for reasons and solutions", name, installedCSV, constant.GeneralTroubleshooting)
	} else if opt.OperatorStatus == "" || opt.OperatorStatus != apiv3.CRSucceeded {
		b.EventRecorder.Eventf(instance, "Warning", "Bedrock Operator Failed", "ClusterServiceVersion %s/%s is not healthy, please check troubleshooting document %s for reasons and solutions", namespace, installedCSV, constant.GeneralTroubleshooting)
	}

	return opt, nil
}

func (b *Bootstrap) fetchSubscription(subName, packageManifest, operatorNs string) (*olmv1alpha1.Subscription, error) {
	sub := &olmv1alpha1.Subscription{}
	if err := b.Reader.Get(context.TODO(), types.NamespacedName{Name: subName, Namespace: operatorNs}, sub); err != nil {
		klog.V(2).Infof("Failed to get Subscription %s in namespace %s: %v, list subscription by packageManifest and operatorNs", subName, operatorNs, err)
		// List the subscription by packageManifest and operatorNs
		// The subscription contain label "operators.coreos.com/<packageManifest>.<operatorNs>: ''"
		subList := &olmv1alpha1.SubscriptionList{}
		labelKey := util.GetFirstNCharacter(packageManifest+"."+operatorNs, 63)
		if err := b.Reader.List(context.TODO(), subList, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(labels.Set{
				"operators.coreos.com/" + labelKey: "",
			}),
			Namespace: operatorNs,
		}); err != nil {
			klog.Errorf("Failed to list Subscription by packageManifest %s and operatorNs %s: %v", packageManifest, operatorNs, err)
			return nil, err
		}

		// Check if multiple subscriptions exist
		if len(subList.Items) > 1 {
			return nil, fmt.Errorf("multiple subscriptions found by packageManifest %s and operatorNs %s", packageManifest, operatorNs)
		} else if len(subList.Items) == 0 {
			return nil, fmt.Errorf("no subscription found by packageManifest %s and operatorNs %s", packageManifest, operatorNs)
		}
		sub = &subList.Items[0]
	}
	return sub, nil
}

// UpdatePostgresClusterImage checks if the Postgres Cluster image needs to be updated
func (b *Bootstrap) UpdatePostgresClusterImage(ctx context.Context, instance *apiv3.CommonService) error {
	// Check if Postgres Cluster CR "common-service-db" is created in service namespace
	pgCluster := &unstructured.Unstructured{}
	pgCluster.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   constant.PGClusterGroup,
		Version: "v1",
		Kind:    constant.PGClusterKind,
	})

	if clusterCRDExists, err := b.CheckCRD(constant.PGClusterGroup+"/v1", constant.PGClusterKind); err != nil {
		klog.Errorf("Failed to check if Postgres Cluster CRD exists: %v", err)
		return err
	} else if !clusterCRDExists {
		klog.Infof("Postgres %s Cluster CRD not found, skipping Postgres Cluster image update check", constant.PGClusterGroup+"/v1")
		return nil
	}

	if err := b.Client.Get(ctx, types.NamespacedName{
		Name:      constant.CSPGCluster,
		Namespace: b.CSData.ServicesNs,
	}, pgCluster); err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Postgres Cluster CR %s not found in namespace %s, skipping Cluster CR image update check", constant.CSPGCluster, b.CSData.ServicesNs)
			return nil
		}
		return err
	}

	configMap, err := b.getPostgresImageConfigMap(ctx)
	if err != nil {
		return err
	} else if configMap == nil {
		klog.Infof("Neither %s nor %s configmap found in namespace %s, skipping Postgres Cluster image update check",
			constant.PostgreSQLImageConfigMap, constant.CSPostgreSQLImageConfigMap, b.CSData.OperatorNs)
		return nil
	}
	configMapName := configMap.GetName()
	imageKeyName := constant.PostgreSQL16ImageKey

	// Get the image from the configmap
	desiredImage, exists := configMap.Data[imageKeyName]
	if !exists {
		klog.Infof("Image key %s not found in configmap %s/%s, skipping image update check",
			imageKeyName, b.CSData.OperatorNs, configMapName)
		return nil
	}

	// Get current image from the Postgres Cluster CR
	currentImage, found, err := unstructured.NestedString(pgCluster.Object, "spec", "imageName")
	if err != nil {
		return err
	}

	// If image is already updated, return nil
	if found && currentImage == desiredImage {
		klog.Infof("Postgres Cluster CR %s image already updated to the desired image in configmap %s/%s",
			constant.CSPGCluster, b.CSData.OperatorNs, configMapName)
		return nil
	}

	// Update the configmap with ODLM metadata to trigger ODLM reconciliation
	if err := b.updateConfigMapWithODLMMetadata(ctx, configMap); err != nil {
		return err
	}

	// Wait for the image to be updated
	// If timeout occurs, controller will update the image in the Postgres Cluster CR
	if err := utilwait.PollUntilContextTimeout(ctx, time.Second*10, time.Minute*2, true, func(ctx context.Context) (done bool, err error) {
		// Fetch the latest Postgres Cluster CR
		err = b.Client.Get(ctx, types.NamespacedName{
			Name:      constant.CSPGCluster,
			Namespace: b.CSData.ServicesNs,
		}, pgCluster)

		if err != nil {
			if errors.IsNotFound(err) {
				klog.Infof("Postgres Cluster CR %s not found in namespace %s, skipping image update check",
					constant.CSPGCluster, b.CSData.ServicesNs)
				return true, nil
			}
			return false, err
		}

		// Check if image is updated
		currentImage, found, err := unstructured.NestedString(pgCluster.Object, "spec", "imageName")
		if err != nil {
			return false, err
		}

		if !found {
			klog.Warningf("spec.imageName field not found in Postgres Cluster CR %s in namespace %s",
				constant.CSPGCluster, b.CSData.ServicesNs)
			return false, nil
		}

		if currentImage == desiredImage {
			klog.Infof("Postgres Cluster CR %s image updated to the desired image in configmap %s/%s",
				constant.CSPGCluster, b.CSData.OperatorNs, configMapName)
			return true, nil
		}

		klog.Infof("Postgres Cluster CR %s image is not updated, waiting for update to the desired image in configmap %s/%s",
			constant.CSPGCluster, b.CSData.OperatorNs, configMapName)
		return false, nil
	}); err != nil {
		klog.Warningf("Failed to wait for Postgres Cluster CR %s image update: %v", constant.CSPGCluster, err)
		// If the image is not updated, update the Postgres Cluster CR with the desired image
		klog.Infof("Updating Postgres Cluster CR %s image to the desired image %s in configmap %s/%s",
			constant.CSPGCluster, desiredImage, b.CSData.OperatorNs, configMapName)
		if err := unstructured.SetNestedField(pgCluster.Object, desiredImage, "spec", "imageName"); err != nil {
			klog.Errorf("Failed to set desired image in Postgres Cluster CR %s: %v", constant.CSPGCluster, err)
			return err
		}
		if err := b.Client.Update(ctx, pgCluster); err != nil {
			return fmt.Errorf("failed to update Postgres Cluster CR %s image update: %v", constant.CSPGCluster, err)
		}
	}

	klog.Infof("Postgres Cluster CR %s image successfully updated to the desired image in configmap %s/%s",
		constant.CSPGCluster, b.CSData.OperatorNs, configMapName)
	return nil
}

// getPostgresImageConfigMap gets the configmap containing PostgreSQL image information
// It first tries to get the configmap deployed with Postgres Operator,
// and if not found, it looks for the configmap created by CS operator
func (b *Bootstrap) getPostgresImageConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	configMapName := constant.PostgreSQLImageConfigMap

	if err := b.Reader.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: b.CSData.OperatorNs,
	}, configMap); err != nil && errors.IsNotFound(err) {
		// If the configmap is not found, find configmap created by CS operator
		configMapName = constant.CSPostgreSQLImageConfigMap

		if err := b.Client.Get(ctx, types.NamespacedName{
			Name:      configMapName,
			Namespace: b.CSData.OperatorNs,
		}, configMap); err != nil {
			if errors.IsNotFound(err) {

				return nil, nil
			}
			klog.Errorf("Failed to get configmap %s in namespace %s: %v", configMapName, b.CSData.OperatorNs, err)
			return nil, err
		}
	} else if err != nil {
		klog.Errorf("Failed to get configmap %s in namespace %s: %v", configMapName, b.CSData.OperatorNs, err)
		return nil, err
	}

	return configMap, nil
}

// updateConfigMapWithODLMMetadata adds ODLM-specific labels and annotations to a ConfigMap
// to ensure it's properly reconciled by the Operand Deployment Lifecycle Manager
func (b *Bootstrap) updateConfigMapWithODLMMetadata(ctx context.Context, configMap *corev1.ConfigMap) error {
	// Check if the configmap has the required label and annotation
	needsLabelUpdate := false
	cmLabels := configMap.GetLabels()
	if cmLabels == nil {
		cmLabels = make(map[string]string)
		needsLabelUpdate = true
	}

	if _, exists := cmLabels[constant.ODLMWatchLabel]; !exists {
		cmLabels[constant.ODLMWatchLabel] = "true"
		needsLabelUpdate = true
	}

	cmAnnotations := configMap.GetAnnotations()
	if cmAnnotations == nil {
		cmAnnotations = make(map[string]string)
		needsLabelUpdate = true
	}

	expectedAnnotation := fmt.Sprintf("OperandConfig.%s.common-service", b.CSData.ServicesNs)
	if cmAnnotations[constant.ODLMReferenceAnno] != expectedAnnotation {
		cmAnnotations[constant.ODLMReferenceAnno] = expectedAnnotation
		needsLabelUpdate = true
	}

	if needsLabelUpdate {
		klog.Infof("Updating configmap %s/%s with ODLM labels and annotations to trigger the ODLM reconciliation", configMap.Namespace, configMap.Name)
		configMap.SetLabels(cmLabels)
		configMap.SetAnnotations(cmAnnotations)
		if err := b.Client.Update(ctx, configMap); err != nil {
			return err
		}
	}

	return nil
}
