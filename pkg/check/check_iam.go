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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	DeployNames = []string{"ibm-iam-operator", "auth-idp", "auth-pap", "auth-pdp", "configmap-watcher", "iam-policy-controller", "oidcclient-watcher", "secret-watcher"}
)

func IamStatus(mgr manager.Manager) {
	reader := mgr.GetAPIReader()
	client := mgr.GetClient()

	for {
		iamStatus := overallIamStatus(reader)
		if err := createUpdateConfigmap(reader, client, iamStatus); err != nil {
			klog.Error("create or update configmap failed")
		}
		time.Sleep(2 * time.Minute)
	}
}

func overallIamStatus(reader client.Reader) string {
	for _, po := range DeployNames {
		status := getDeploymentStatus(reader, po)
		if status == "NotReady" {
			return status
		}
	}
	return "Ready"
}

func getDeploymentStatus(reader client.Reader, name string) string {
	deploy := &appsv1.Deployment{}
	deployName := name
	deployNs := "ibm-common-services"

	err := reader.Get(context.TODO(), types.NamespacedName{Name: deployName, Namespace: deployNs}, deploy)
	if err != nil {
		return "NotReady"
	}

	if deploy.Status.ReadyReplicas != deploy.Status.Replicas {
		return "NotReady"
	}
	return "Ready"
}

func createUpdateConfigmap(reader client.Reader, client client.Client, status string) error {
	cm := &corev1.ConfigMap{}
	cmName := "ibm-common-services-status"
	cmNs := "kube-public"
	if status == "NotReady" {
		klog.Info("IAM status is NoReady, waiting some minutes...")
	}
	err := reader.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: cmNs}, cm)
	if err != nil {
		if errors.IsNotFound(err) {
			cm.Name = cmName
			cm.Namespace = cmNs
			cm.Data = make(map[string]string)
			cm.Data["iamstatus"] = status
			if err := client.Create(context.TODO(), cm); err != nil {
				return err
			}
		}
		return err
	}
	if cm.Data["iamstatus"] != status {
		klog.Infof("IAM status is %s", status)
		cm.Data["iamstatus"] = status
		if err = client.Update(context.TODO(), cm); err != nil {
			return err
		}
	}
	return nil
}
