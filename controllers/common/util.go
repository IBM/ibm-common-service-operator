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

package common

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	utilyaml "github.com/ghodss/yaml"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	nssv1 "github.com/IBM/ibm-namespace-scope-operator/api/v1"
)

type CsMaps struct {
	ControlNs     string      `json:"controlNamespace"`
	NsMappingList []NsMapping `json:"namespaceMapping"`
	// DefaultCsNs   string      `json:"defaultCsNs"`
}

type NsMapping struct {
	RequestNs []string `json:"requested-from-namespace"`
	CsNs      string   `json:"map-to-common-service-namespace"`
}

var (
	ImageList = []string{"IBM_SECRETSHARE_OPERATOR_IMAGE", "IBM_CS_WEBHOOK_IMAGE"}
)

// CompareVersion takes vx.y.z, vx.y.z -> bool: if v1 is larger than v2
func CompareVersion(v1, v2 string) (v1IsLarger bool, err error) {
	if v1 == "" {
		v1 = "0.0.0"
	}
	v1Slice := strings.Split(v1, ".")
	if len(v1Slice) == 1 {
		v1 = "0.0." + v1
	}

	if v2 == "" {
		v2 = "0.0.0"
	}
	v2Slice := strings.Split(v2, ".")
	if len(v2Slice) == 1 {
		v2 = "0.0." + v2
	}

	v1Slice = strings.Split(v1, ".")
	v2Slice = strings.Split(v2, ".")
	for index := range v1Slice {
		v1SplitInt, e1 := strconv.Atoi(v1Slice[index])
		if e1 != nil {
			return false, e1
		}
		v2SplitInt, e2 := strconv.Atoi(v2Slice[index])
		if e2 != nil {
			return false, e2
		}

		if v1SplitInt > v2SplitInt {
			return true, nil
		} else if v1SplitInt == v2SplitInt {
			continue
		} else {
			return false, nil
		}
	}
	return false, nil
}

// YamlToObjects convert YAML content to unstructured objects
func YamlToObjects(yamlContent []byte) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured

	// This step is for converting large yaml file, we can remove it after using "apimachinery" v0.19.0
	if len(yamlContent) > 1024*64 {
		object, err := YamlToObject(yamlContent)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
		return objects, nil
	}

	yamlDecoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	reader := json.YAMLFramer.NewFrameReader(io.NopCloser(bytes.NewReader(yamlContent)))
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

// YamlToObject converting large yaml file, we can remove it after using "apimachinery" v0.19.0
func YamlToObject(yamlContent []byte) (*unstructured.Unstructured, error) {
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

// NewUnstructured return Unstructured object
func NewUnstructured(group, kind, version string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Kind:    kind,
		Version: version})
	return u
}

// NewUnstructuredList return UnstructuredList object
func NewUnstructuredList(group, kind, version string) *unstructured.UnstructuredList {
	ul := &unstructured.UnstructuredList{}
	ul.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Kind:    kind,
		Version: version})
	return ul
}

func ObjectListToNewUnstructuredList(objs interface{}) (*unstructured.UnstructuredList, error) {
	contents, err := runtime.DefaultUnstructuredConverter.ToUnstructured(objs)
	if err != nil {
		return nil, fmt.Errorf("could not convert Object to Unstructured resource: %v", err)
	}
	newUnstrList := &unstructured.UnstructuredList{}
	newUnstrList.SetUnstructuredContent(contents)
	return newUnstrList, nil
}

// GetOperatorName return the operator name
func GetOperatorName() (string, error) {
	operatorName, found := os.LookupEnv(constant.OperatorNameEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", constant.OperatorNameEnvVar)
	}
	if len(operatorName) == 0 {
		return "", fmt.Errorf("%s must not be empty", constant.OperatorNameEnvVar)
	}
	return operatorName, nil
}

// GetOperatorNamespace returns the namespace the operator should be running in.
func GetOperatorNamespace() (string, error) {
	ns, found := os.LookupEnv(constant.OperatorNamespaceEnvVar)
	if !found {
		nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("namespace not found for current environment")
			}
			return "", err
		}
		ns = strings.TrimSpace(string(nsBytes))
	}
	if len(ns) == 0 {
		return "", fmt.Errorf("operator namespace is empty")
	}
	klog.V(2).Info("Found namespace: ", ns)
	return ns, nil
}

// GetCPFSNamespace returns the namespace where foundational services operator should be running
func GetCPFSNamespace(r client.Reader) (cpfsNamespace string) {
	cpfsNamespace, err := GetOperatorNamespace()
	if err != nil {
		return
	}

	defaultCsCR := &apiv3.CommonService{}
	csName := "common-service"
	if err := r.Get(context.TODO(), types.NamespacedName{Name: csName, Namespace: cpfsNamespace}, defaultCsCR); err != nil {
		return
	}

	if string(defaultCsCR.Status.ConfigStatus.OperatorNamespace) != "" {
		cpfsNamespace = string(defaultCsCR.Status.ConfigStatus.OperatorNamespace)
	}
	return
}

// GetServicesNamespace returns the namespace where foundational services CRs should be running
func GetServicesNamespace(r client.Reader) (servicesNamespace string) {
	servicesNamespace, err := GetOperatorNamespace()
	if err != nil {
		return
	}

	defaultCsCR := &apiv3.CommonService{}
	csName := "common-service"
	if err := r.Get(context.TODO(), types.NamespacedName{Name: csName, Namespace: servicesNamespace}, defaultCsCR); err != nil {
		return
	}

	if string(defaultCsCR.Spec.ServicesNamespace) != "" {
		servicesNamespace = string(defaultCsCR.Spec.ServicesNamespace)
	}
	if string(defaultCsCR.Status.ConfigStatus.ServicesNamespace) != "" {
		servicesNamespace = string(defaultCsCR.Status.ConfigStatus.ServicesNamespace)
	}
	return
}

// GetWatchNamespace returns the list of namespaces that the operator watches
func GetWatchNamespace() string {
	ns, found := os.LookupEnv("WATCH_NAMESPACE")
	if !found {
		ns, err := GetOperatorNamespace()
		if err != nil {
			return ""
		}
		return ns
	}
	return ns
}

// GetNSSCMSynchronization returns whether NSS ConfigMap shchronization with OperatorGroup is enabled
func GetNSSCMSynchronization() bool {
	isEnable, found := os.LookupEnv("NSSCM_SYNC_MODE")
	if !found || isEnable == "true" {
		return true
	}
	return false
}

// Contains returns whether the sub-string is contained
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Reverse resverses the string
func Reverse(original []string) []string {
	reversed := make([]string, 0, len(original))
	for i := len(original) - 1; i >= 0; i-- {
		reversed = append(reversed, original[i])
	}
	return reversed
}

// Namespacelize adds the namespace specified
func Namespacelize(resource, placeholder, ns string) string {
	return strings.ReplaceAll(resource, placeholder, ns)
}

func ReplaceImages(resource string) (result string) {
	result = resource
	for _, image := range ImageList {
		result = strings.ReplaceAll(result, image, GetImage(image))
	}
	return
}

func GetImage(imageName string) string {
	ns, _ := os.LookupEnv(imageName)
	return ns
}

// GetCmOfMapCs gets ConfigMap of Common Services Maps
func GetCmOfMapCs(r client.Reader) (*corev1.ConfigMap, error) {
	cmName := constant.CsMapConfigMap
	cmNs := "kube-public"
	csConfigmap := &corev1.ConfigMap{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: cmNs}, csConfigmap)
	if err != nil {
		return nil, err
	}
	return csConfigmap, nil
}

// CheckStorageClass gets StorageClassList in current cluster, then validates whether StorageClass created
func CheckStorageClass(r client.Reader) error {
	csStorageClass := &storagev1.StorageClassList{}
	err := r.List(context.TODO(), csStorageClass)
	if err != nil {
		return fmt.Errorf("fail to list storageClass: %v", err)
	}

	size := len(csStorageClass.Items)
	klog.Info("StorageClass Number: ", size)

	if size <= 0 {
		klog.Warning("StorageClass is not found, which might be required by CloudPak services, please refer to CloudPak's documentation for prerequisites.")
	}
	return nil
}

// UpdateNSList updates adopter namespaces of Common Services
func UpdateNSList(r client.Reader, c client.Client, cm *corev1.ConfigMap, nssKey, cpfsNamespace string, addControlNs bool) error {
	nsScope := &nssv1.NamespaceScope{}
	nsScopeKey := types.NamespacedName{Name: nssKey, Namespace: cpfsNamespace}
	if err := r.Get(context.TODO(), nsScopeKey, nsScope); err != nil {
		return err
	}
	var nsMems []string
	nsSet := make(map[string]interface{})

	for _, ns := range nsScope.Spec.NamespaceMembers {
		nsSet[ns] = struct{}{}
	}

	commonServiceMaps, ok := cm.Data["common-service-maps.yaml"]
	if !ok {
		return fmt.Errorf("there is no common-service-maps.yaml in configmap kube-public/common-service-maps")
	}

	var cmData CsMaps
	if err := utilyaml.Unmarshal([]byte(commonServiceMaps), &cmData); err != nil {
		return fmt.Errorf("failed to fetch data of configmap common-service-maps: %v", err)
	}

	if addControlNs {
		if len(cmData.ControlNs) > 0 {
			nsSet[cmData.ControlNs] = struct{}{}
		}
	}

	for _, nsMapping := range cmData.NsMappingList {
		if cpfsNamespace == nsMapping.CsNs {
			for _, ns := range nsMapping.RequestNs {
				nsSet[ns] = struct{}{}
			}
		}
	}

	for ns := range nsSet {
		nsMems = append(nsMems, ns)
	}

	nsScope.Spec.NamespaceMembers = nsMems

	if err := c.Update(context.TODO(), nsScope); err != nil {
		return err
	}

	return nil
}

// UpdateAllNSList updates all adopter and CS namespaces into NSS CR
func UpdateAllNSList(r client.Reader, c client.Client, cm *corev1.ConfigMap, nssKey, nssNs string) error {
	nsScope := &nssv1.NamespaceScope{}
	nsScopeKey := types.NamespacedName{Name: nssKey, Namespace: nssNs}
	if err := r.Get(context.TODO(), nsScopeKey, nsScope); err != nil {
		return err
	}
	var nsMems []string
	nsSet := make(map[string]interface{})

	for _, ns := range nsScope.Spec.NamespaceMembers {
		nsSet[ns] = struct{}{}
	}

	commonServiceMaps, ok := cm.Data["common-service-maps.yaml"]
	if !ok {
		return fmt.Errorf("there is no common-service-maps.yaml in configmap kube-public/common-service-maps")
	}

	var cmData CsMaps
	if err := utilyaml.Unmarshal([]byte(commonServiceMaps), &cmData); err != nil {
		return fmt.Errorf("failed to fetch data of configmap common-service-maps: %v", err)
	}

	if len(cmData.ControlNs) > 0 {
		nsSet[cmData.ControlNs] = struct{}{}
	}

	for _, nsMapping := range cmData.NsMappingList {
		nsSet[nsMapping.CsNs] = struct{}{}
		for _, ns := range nsMapping.RequestNs {
			nsSet[ns] = struct{}{}
		}
	}

	for ns := range nsSet {
		nsMems = append(nsMems, ns)
	}

	nsScope.Spec.NamespaceMembers = nsMems

	if err := c.Update(context.TODO(), nsScope); err != nil {
		return err
	}

	return nil
}

// CheckSaas checks whether it is a SaaS deployment for Common Services
func CheckSaas(r client.Reader) (enable bool) {
	cmName := constant.SaasConfigMap
	cmNs := "kube-public"
	saasConfigmap := &corev1.ConfigMap{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: cmNs}, saasConfigmap)
	if errors.IsNotFound(err) {
		klog.V(2).Infof("There is no configmap %v/%v in the cluster: Running Common Service Operator in On-Prem mode", cmNs, cmName)
		return false
	} else if err != nil {
		klog.Errorf("Failed to fetch configmap %v/%v: %v", cmNs, cmName, err)
		return false
	}
	v, ok := saasConfigmap.Data["ibm_cloud_saas"]
	if !ok {
		klog.V(2).Infof("There is no ibm_cloud_saas in configmap %v/%v: Running Common Service Operator in On-Prem mode", cmNs, cmName)
		return false
	}
	if v != "true" {
		return false
	}
	klog.V(2).Infof("Running Common Service Operator in SaaS mode")
	return true
}

// CheckMultiInstance checks whether it is a MultiInstances including SaaS and on-prem MultiInstances
func CheckMultiInstances(r client.Reader) (enable bool) {
	return true
}

// GetControlNs gets control namespace of deploying cluster scope services
func GetControlNs(r client.Reader) (controlNs string) {
	operatorNs, err := GetOperatorNamespace()
	if err != nil {
		klog.Errorf("Getting operator namespace failed: %v", err)
		return
	}
	controlNs = operatorNs
	return
}

func GetApprovalModeinNs(r client.Reader, ns string) (approvalMode string, err error) {
	approvalMode = string(olmv1alpha1.ApprovalAutomatic)
	subList := &olmv1alpha1.SubscriptionList{}
	if err := r.List(context.TODO(), subList, &client.ListOptions{Namespace: ns}); err != nil {
		return approvalMode, err
	}
	for _, sub := range subList.Items {
		if sub.Spec.InstallPlanApproval == olmv1alpha1.ApprovalManual {
			approvalMode = string(olmv1alpha1.ApprovalManual)
			return
		}
	}
	return
}

// GetCatalogSource gets CatalogSource will be used by operators
func GetCatalogSource(packageName, ns string, r client.Reader) (CatalogSourceName, CatalogSourceNS string) {
	subList := &olmv1alpha1.SubscriptionList{}
	if err := r.List(context.TODO(), subList, &client.ListOptions{Namespace: ns}); err != nil {
		klog.Info(err)
	}

	var subscriptions []olmv1alpha1.Subscription
	for _, sub := range subList.Items {
		if sub.Spec.Package == packageName {
			subscriptions = append(subscriptions, sub)
		}
	}

	if len(subscriptions) == 0 {
		klog.Errorf("not found %v subscription in namespace: %v", packageName, ns)
		return "", ""
	}

	if len(subscriptions) > 1 {
		klog.Errorf("found more than one %v subscription in namespace: %v", packageName, ns)
		return "", ""
	}

	return subscriptions[0].Spec.CatalogSource, subscriptions[0].Spec.CatalogSourceNamespace
}

// UpdateCsMaps will update namespaceMapping in common-service-maps
func UpdateCsMaps(cm *corev1.ConfigMap, requestNsList, servicesNS, operatorNs string) error {
	commonServiceMaps, ok := cm.Data["common-service-maps.yaml"]
	if !ok {
		return fmt.Errorf("there is no common-service-maps.yaml in configmap kube-public/common-service-maps")
	}

	var cmData CsMaps
	if err := utilyaml.Unmarshal([]byte(commonServiceMaps), &cmData); err != nil {
		return fmt.Errorf("failed to fetch data of configmap common-service-maps: %v", err)
	}

	var newNsMapping NsMapping
	var newNsMappingList []NsMapping

	// construct new mapping for current entry
	newNsMapping.RequestNs = append(newNsMapping.RequestNs, strings.Split(requestNsList, ",")...)
	newNsMapping.CsNs = servicesNS
	newNsMappingList = append(newNsMappingList, newNsMapping)

	// add well known control namespace for CP2 backward compatibility
	if cmData.ControlNs == "" {
		cmData.ControlNs = "cs-control"
	}

	for _, nsMapping := range cmData.NsMappingList {
		var copiedNsMapping NsMapping
		// exclude entire entry if its map-to-cs-namespace is in WATCH_NAMESPACE
		if Contains(strings.Split(requestNsList, ","), nsMapping.CsNs) {
			continue
		}
		copiedNsMapping.CsNs = nsMapping.CsNs

		for _, ns := range nsMapping.RequestNs {
			// exclude request namespace if it is in WATCH_NAMESPACE
			if Contains(strings.Split(requestNsList, ","), ns) {
				continue
			}
			copiedNsMapping.RequestNs = append(copiedNsMapping.RequestNs, ns)
		}

		// existing entry is valid and add it into new mapping list
		if len(copiedNsMapping.CsNs) != 0 && len(copiedNsMapping.RequestNs) != 0 {
			newNsMappingList = append(newNsMappingList, copiedNsMapping)
		}
	}

	cmData.NsMappingList = newNsMappingList
	commonServiceMap, error := utilyaml.Marshal(&cmData)
	if error != nil {
		return fmt.Errorf("failed to fetch data of configmap common-service-maps: %v", error)
	}
	cm.Data["common-service-maps.yaml"] = string(commonServiceMap)

	if !(cm.Labels != nil && cm.Labels[constant.CsManagedLabel] == "true") {
		EnsureLabelsForConfigMap(cm, map[string]string{
			constant.CsManagedLabel: "true",
		})
	}
	return nil
}

// ValidateCsMaps checks common-service-maps has no scope overlapping
func ValidateCsMaps(cm *corev1.ConfigMap) error {
	commonServiceMaps, ok := cm.Data["common-service-maps.yaml"]
	if !ok {
		return fmt.Errorf("there is no common-service-maps.yaml in configmap kube-public/common-service-maps")
	}

	var cmData CsMaps
	if err := utilyaml.Unmarshal([]byte(commonServiceMaps), &cmData); err != nil {
		return fmt.Errorf("failed to fetch data of configmap common-service-maps: %v", err)
	}

	CsNsSet := make(map[string]interface{})
	RequestNsSet := make(map[string]interface{})

	for _, nsMapping := range cmData.NsMappingList {
		// validate masterNamespace and controlNamespace
		if cmData.ControlNs == nsMapping.CsNs {
			return fmt.Errorf("invalid controlNamespace: %v. Cannot be the same as map-to-common-service-namespace", cmData.ControlNs)
		}
		if _, ok := CsNsSet[nsMapping.CsNs]; ok {
			return fmt.Errorf("invalid map-to-common-service-namespace: %v", nsMapping.CsNs)
		}
		CsNsSet[nsMapping.CsNs] = struct{}{}
		// validate CloudPak Namespace and controlNamespace
		for _, ns := range nsMapping.RequestNs {
			if cmData.ControlNs == ns {
				return fmt.Errorf("invalid controlNamespace: %v. Cannot be the same as requested-from-namespace", cmData.ControlNs)
			}
			if _, ok := RequestNsSet[ns]; ok {
				return fmt.Errorf("invalid requested-from-namespace: %v", ns)
			}
			RequestNsSet[ns] = struct{}{}
		}
	}
	return nil
}

// GetCsScope fetchs the namespaces from its own requested-from-namespace and map-to-common-service-namespace
func GetCsScope(cm *corev1.ConfigMap, cpfsNamespace string) ([]string, error) {
	var nsMems []string
	nsSet := make(map[string]interface{})

	commonServiceMaps, ok := cm.Data["common-service-maps.yaml"]
	if !ok {
		return nsMems, fmt.Errorf("there is no common-service-maps.yaml in configmap kube-public/common-service-maps")
	}

	var cmData CsMaps
	if err := utilyaml.Unmarshal([]byte(commonServiceMaps), &cmData); err != nil {
		return nsMems, fmt.Errorf("failed to fetch data of configmap common-service-maps: %v", err)
	}

	for _, nsMapping := range cmData.NsMappingList {
		if cpfsNamespace == nsMapping.CsNs {
			nsSet[cpfsNamespace] = struct{}{}
			for _, ns := range nsMapping.RequestNs {
				nsSet[ns] = struct{}{}
			}
		}
	}

	for ns := range nsSet {
		nsMems = append(nsMems, ns)
	}

	return nsMems, nil
}

// EnsureLabelsForConfigMap ensures that the specifc ConfigMap has the certain labels
func EnsureLabelsForConfigMap(cm *corev1.ConfigMap, labels map[string]string) {
	if cm.Labels == nil {
		cm.Labels = make(map[string]string)
	}
	for k, v := range labels {
		cm.Labels[k] = v
	}
}

// EnsureLabels ensures that the specifc resource has the certain labels
func EnsureLabels(resource *unstructured.Unstructured, labels map[string]string) {
	if resource.Object["metadata"].(map[string]interface{})["labels"] == nil {
		resource.Object["metadata"].(map[string]interface{})["labels"] = make(map[string]string)
	}
	for k, v := range labels {
		resource.Object["metadata"].(map[string]interface{})["labels"].(map[string]string)[k] = v
	}
}

// GetRequestNs gets requested-from-namespace of map-to-common-service-namespace
func GetRequestNs(r client.Reader) (requestNs []string) {
	operatorNs, err := GetOperatorNamespace()
	if err != nil {
		klog.Errorf("Getting operator namespace failed: %v", err)
		return
	}

	csConfigmap, err := GetCmOfMapCs(r)
	if err != nil {
		klog.V(2).Infof("Could not find configmap kube-public/common-service-maps: %v", err)
		return
	}

	commonServiceMaps, ok := csConfigmap.Data["common-service-maps.yaml"]
	if !ok {
		klog.Infof("There is no common-service-maps.yaml in configmap kube-public/common-service-maps")
		return
	}

	var cmData CsMaps
	if err := utilyaml.Unmarshal([]byte(commonServiceMaps), &cmData); err != nil {
		klog.Errorf("Failed to fetch data of configmap common-service-maps: %v", err)
		return
	}

	for _, nsMapping := range cmData.NsMappingList {
		if operatorNs == nsMapping.CsNs {
			requestNs = nsMapping.RequestNs
			break
		}
	}

	return
}

// GetNssCmNs gets namespaces from namespace-scope ConfigMap
func GetNssCmNs(r client.Reader, cpfsNamespace string) (nssCmNs []string) {
	nssConfigMap := GetCmOfNss(r, cpfsNamespace)

	nssNsMems, ok := nssConfigMap.Data["namespaces"]
	if !ok {
		klog.Infof("There is no namespace in configmap %v/%v", cpfsNamespace, constant.NamespaceScopeConfigmapName)
		return
	}
	nssCmNs = strings.Split(nssNsMems, ",")

	return nssCmNs
}

// GetCmOfNss gets ConfigMap of Namespace-scope
func GetCmOfNss(r client.Reader, operatorNs string) *corev1.ConfigMap {
	cmName := constant.NamespaceScopeConfigmapName
	cmNs := operatorNs
	nssConfigmap := &corev1.ConfigMap{}

	for {
		if err := utilwait.PollImmediateInfinite(time.Second*10, func() (done bool, err error) {
			err = r.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: cmNs}, nssConfigmap)
			if err != nil {
				if errors.IsNotFound(err) {
					klog.Infof("waiting for configmap %v/%v", operatorNs, constant.NamespaceScopeConfigmapName)
					return false, nil
				}
				return false, err
			}
			return true, nil
		}); err == nil {
			break
		} else {
			klog.Errorf("Failed to get configmap %v/%v: %v, retry in 10 seconds", operatorNs, constant.NamespaceScopeConfigmapName, err)
			time.Sleep(10 * time.Second)
		}
	}

	return nssConfigmap
}

func GetResourcesDynamically(ctx context.Context, dynamic dynamic.Interface, group string, version string, resource string) (
	[]unstructured.Unstructured, error) {

	resourceID := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	// Namespace is empty refer to all namespace
	list, err := dynamic.Resource(resourceID).Namespace("").List(ctx, metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// GetEnableOpreqWebhook check if enable the webhook for the OperandRequest
func GetEnableOpreqWebhook() bool {
	enable, found := os.LookupEnv("ENABLE_OPREQ_WEBHOOK")
	if !found {
		return true
	}
	if enable != "TRUE" {
		return false
	}
	return true
}

// EnsureLabelsForConfigMap ensures that the specifc ConfigMap has the certain labels
func EnsureLabelsForCsCR(cs *apiv3.CommonService, labels map[string]string) {
	if cs.Labels == nil {
		cs.Labels = make(map[string]string)
	}
	for k, v := range labels {
		cs.Labels[k] = v
	}
}

func CompareCsCR(csCR *apiv3.CommonService, existingCsCR *apiv3.CommonService) (needUpdate bool) {
	return !equality.Semantic.DeepEqual(csCR.GetLabels(), existingCsCR.GetLabels()) || !equality.Semantic.DeepEqual(csCR.GetAnnotations(), existingCsCR.GetAnnotations()) || !equality.Semantic.DeepEqual(csCR.Spec, existingCsCR.Spec)
}
