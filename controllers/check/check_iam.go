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

package check

import (
	"context"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	util "github.com/IBM/ibm-common-service-operator/controllers/common"
)

var (
	DeployNames     = []string{"ibm-iam-operator", "auth-idp", "auth-pap", "auth-pdp", "oidcclient-watcher", "secret-watcher"}
	JobNames        = []string{"iam-onboarding", "security-onboarding", "oidc-client-registration"}
	MasterNamespace string
)

// IamStatus check IAM status if ready
func IamStatus(mgr manager.Manager) {
	r := mgr.GetAPIReader()
	c := mgr.GetClient()

	MasterNamespace = util.GetMasterNs(r)

	for {
		if !getIamSubscription(r) {
			time.Sleep(2 * time.Minute)
			continue
		}
		iamStatus := overallIamStatus(r)
		if err := createUpdateConfigmap(r, c, iamStatus); err != nil {
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

func overallIamStatus(r client.Reader) string {
	for _, deploy := range DeployNames {
		status := getDeploymentStatus(r, deploy)
		if status == "NotReady" {
			return status
		}
	}
	for _, job := range JobNames {
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

func createUpdateConfigmap(r client.Reader, c client.Client, status string) error {
	cm := &corev1.ConfigMap{}
	cmName := "ibm-common-services-status"
	cmNs := "kube-public"
	if status == "NotReady" {
		klog.Info("IAM status is NoReady, waiting some minutes...")
	}
	err := r.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: cmNs}, cm)
	if err != nil {
		if errors.IsNotFound(err) {
			cm.Name = cmName
			cm.Namespace = cmNs
			cm.Data = make(map[string]string)
			cm.Data["iamstatus"] = status
			if err := c.Create(context.TODO(), cm); err != nil {
				klog.Errorf("Failed to create ConfigMap %s: %v", cmName, err)
				return err
			}
			return nil
		}
		return err
	}
	if cm.Data["iamstatus"] != status {
		klog.Infof("IAM status is %s", status)
		cm.Data["iamstatus"] = status
		if err = c.Update(context.TODO(), cm); err != nil {
			klog.Errorf("Failed to update ConfigMap %s: %v", cmName, err)
			return err
		}
	}
	return nil
}
