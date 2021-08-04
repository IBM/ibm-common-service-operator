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

package goroutines

import (
	"context"
	"regexp"
	"time"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
)

// UpdateCsCrStatus will update cs cr status according to each bedrock operator
func UpdateCsCrStatus(bs *bootstrap.Bootstrap) {
	MasterNamespace := bs.CSData.MasterNs
	// ControlNamespace := bs.CSData.ControlNs

	for {
		instance := &apiv3.CommonService{}
		// if err := bs.Client.Get(ctx, MasterNamespace.(types.NamespacedName), instance); err != nil { // get ibm-common-services ns cs cr
		if instance == nil {
			klog.Infof("[DEBUG] %s CS CR not created", MasterNamespace)
			// maybe add a waiting time?
			continue
		}

		var operatorSlice []util.BedrockOperator
		// get all installed operator

		operatorSlice = getBedrockOperator(bs, "ibm-iam-operator", operatorSlice) // atodo: can I pass the slice as pointer? so less deep copy
		operatorSlice = getBedrockOperator(bs, "ibm-cert-manager-operator", operatorSlice)
		klog.Info("[DEBUG] operatorSlice length")
		klog.Info(len(operatorSlice))

		// update cs cr according to operatorSlice
		for _, opt := range operatorSlice {
			// opt.Version
			// opt.Status
		}
		// instance.Status = operatorSlice // atodo: enable parameter for each oprator

		time.Sleep(1 * time.Minute) // atodo: adjust the wait time
	}
}

func getBedrockOperator(bs *bootstrap.Bootstrap, name string, operatorSlice []util.BedrockOperator) []util.BedrockOperator {
	var opt util.BedrockOperator

	// fetch subscription
	sub := &olmv1alpha1.Subscription{}
	// iamSub.SetGroupVersionKind(olmv1alpha1.SchemeGroupVersion.WithKind("subscription"))
	subKey := types.NamespacedName{
		Name:      name,
		Namespace: MasterNamespace,
	}
	if subErr := bs.Client.Get(ctx, subKey, sub); subErr != nil {
		klog.Info(subErr)
		return nil
	}
	klog.Info("[DEBUG] operator name + version")
	klog.Info(sub.Status.InstalledCSV)
	opt.Version = sub.Status.InstalledCSV

	// fetch csv
	csvKey := types.NamespacedName{
		Name:      opt.Version,
		Namespace: MasterNamespace,
	}
	csv := &olmv1alpha1.ClusterServiceVersion{}
	// opt.SetGroupVersionKind(olmv1alpha1.SchemeGroupVersion.WithKind("ClusterServiceVersion"))
	if csvErr := bs.Client.Get(ctx, csvKey, csv); csvErr != nil {
		klog.Info(csvErr)
		return nil
	}
	csvStatus := csv.Status.Conditions[len(csv.Status.Conditions)-1].Phase
	klog.Info("[DEBUG] csv Status")
	klog.Info(csvStatus)
	// opt.Status = csvStatus

	if opt.Version != nil && opt.Status != nil {
		operatorSlice = append(operatorSlice, opt)
	}
	return operatorSlice
}

// CheckIamStatus check IAM status if ready
func CheckIamStatus(bs *bootstrap.Bootstrap) {
	MasterNamespace = bs.CSData.MasterNs

	for {
		if !getIamSubscription(bs.Reader) {
			if err := cleanUpConfigmap(bs); err != nil {
				klog.Errorf("Create or update configmap failed: %v", err)
			}
			time.Sleep(2 * time.Minute)
			continue
		}

		var deploymentList []string
		if bs.SaasEnable {
			deploymentList = IAMSaaSDeployNames
		} else {
			deploymentList = IAMDeployNames
		}

		iamStatus := overallIamStatus(bs.Reader, deploymentList)
		if err := createUpdateConfigmap(bs, iamStatus); err != nil {
			klog.Errorf("Create or update configmap failed: %v", err)
		}
		time.Sleep(2 * time.Minute)
	}
}

// getIamSubscription return true if IAM subscription found, otherwise return false
func getIamSubscription(r client.Reader) bool {
	subName := "ibm-iam-operator"
	subNs := MasterNamespace
	sub := &olmv1alpha1.Subscription{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: subName, Namespace: subNs}, sub)
	return err == nil
}

func overallIamStatus(r client.Reader, deploymentList []string) string {
	for _, deploy := range deploymentList {
		status := getDeploymentStatus(r, deploy)
		if status == "NotReady" {
			return status
		}
	}
	for _, job := range IAMJobNames {
		status := getJobStatus(r, job)
		if status == "NotReady" {
			return status
		}
	}
	return "Ready"
}

func getJobStatus(r client.Reader, name string) string {
	job := &batchv1.Job{}
	jobName := name
	jobNs := MasterNamespace
	err := r.Get(context.TODO(), types.NamespacedName{Name: jobName, Namespace: jobNs}, job)
	if err != nil {
		klog.Errorf("Failed to get Job %s: %v", jobName, err)
		return "NotReady"
	}

	if job.Status.Succeeded >= *job.Spec.Completions {
		return "Ready"
	}
	return "NotReady"
}

func getDeploymentStatus(r client.Reader, name string) string {
	deploy := &appsv1.Deployment{}
	deployName := name
	deployNs := MasterNamespace

	err := r.Get(context.TODO(), types.NamespacedName{Name: deployName, Namespace: deployNs}, deploy)
	if err != nil {
		klog.Errorf("Failed to get Deployment %s: %v", deployName, err)
		return "NotReady"
	}

	if deploy.Status.ReadyReplicas != deploy.Status.Replicas {
		return "NotReady"
	}
	return "Ready"
}

func createUpdateConfigmap(bs *bootstrap.Bootstrap, status string) error {
	cm := &corev1.ConfigMap{}
	cmName := "ibm-common-services-status"
	cmNs := "kube-public"
	if status == "NotReady" {
		klog.Info("IAM status is NotReady, waiting some minutes...")
	}

	nssNsSlice := util.GetNssCmNs(bs.Reader, bs.CSData.MasterNs)
	err := bs.Reader.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: cmNs}, cm)
	if err != nil {
		// create the iam-status configMap
		if errors.IsNotFound(err) {
			cm.Name = cmName
			cm.Namespace = cmNs
			cm.Data = make(map[string]string)
			for _, nssNs := range nssNsSlice {
				statusKey := nssNs + "-iamstatus"
				cm.Data[statusKey] = status
			}
			cm.Data["iamstatus"] = status
			if err := bs.Client.Create(context.TODO(), cm); err != nil {
				klog.Errorf("Failed to create ConfigMap %s: %v", cmName, err)
				return err
			}
			return nil
		}
		return err
	}

	if _, err := util.GetCmOfMapCs(bs.Reader); err != nil {
		// backward compatibility for non-cs-mapping case
		// overwrite the cm.Data by nss ConfigMap
		if errors.IsNotFound(err) {
			cm.Data = make(map[string]string)
			for _, nssNs := range nssNsSlice {
				statusKey := nssNs + "-iamstatus"
				cm.Data[statusKey] = status
			}
			cm.Data["iamstatus"] = status
			if err = bs.Client.Update(context.TODO(), cm); err != nil {
				klog.Errorf("Failed to update ConfigMap %s: %v", cmName, err)
				return err
			}
			return nil
		}
		return err
	}

	// cs-mapping configMap is found
	isUpdate := false

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	for _, ns := range nssNsSlice {
		statusKey := ns + "-iamstatus"
		if status == "NotReady" {
			delete(cm.Data, statusKey)
		} else {
			cm.Data[statusKey] = status
		}
		isUpdate = true
	}

	if overallStatus := checkOverallStatus(cm.Data); cm.Data["iamstatus"] != overallStatus {
		cm.Data["iamstatus"] = overallStatus
		isUpdate = true
	}

	if isUpdate {
		if err = bs.Client.Update(context.TODO(), cm); err != nil {
			klog.Errorf("Failed to update ConfigMap %s: %v", cmName, err)
			return err
		}
	}

	return nil
}

func cleanUpConfigmap(bs *bootstrap.Bootstrap) error {
	cm := &corev1.ConfigMap{}
	cmName := "ibm-common-services-status"
	cmNs := "kube-public"
	err := bs.Reader.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: cmNs}, cm)
	if errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	cmOfMapCs, err := util.GetCmOfMapCs(bs.Reader)
	if err != nil {
		// backward compatibility for non-cs-mapping case
		// clean up the cm.Data if there is no iam subscriptions
		if errors.IsNotFound(err) {
			cm.Data = make(map[string]string)
		} else {
			return err
		}
	} else {
		nsMems, err := util.GetCsScope(cmOfMapCs, bs.CSData.MasterNs)
		if err != nil {
			return err
		}
		for _, ns := range nsMems {
			delete(cm.Data, ns)
		}
	}

	if _, ok := cm.Data["iamstatus"]; ok {
		cm.Data["iamstatus"] = checkOverallStatus(cm.Data)
	}

	if err = bs.Client.Update(context.TODO(), cm); err != nil {
		klog.Errorf("Failed to update ConfigMap %s: %v", cmName, err)
		return err
	}

	return nil
}

func checkOverallStatus(statusMap map[string]string) string {
	reg, _ := regexp.Compile(`^(.*)\-iamstatus`)
	statusSlice := make([]string, 0)
	for key, status := range statusMap {
		if reg.MatchString(key) {
			statusSlice = append(statusSlice, status)
		}
	}
	if len(statusSlice) == 0 {
		return "NotReady"
	}

	for _, status := range statusSlice {
		if status != "Ready" {
			return status
		}
	}
	return "Ready"
}
