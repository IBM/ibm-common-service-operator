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

	"k8s.io/klog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
)

// +kubebuilder:docs-gen:collapse=Imports

var _ = Describe("CommonService controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		CloudPakNamespace              = "cloudpak-ns"
		OdlmOperatorName               = "operand-deployment-lifecycle-manager"
		CommonServiceOperatorName      = "ibm-common-service-operator"
		CommonServiceOperatorNamespace = "ibm-common-services"
		CommonServiceOperandName       = "common-service"
	)

	var (
		ctx context.Context = context.Background()
	)

	Context("Common service operator bootstrap", func() {
		It("Should be ready", func() {
			Expect(createNamespace(CloudPakNamespace)).Should(Succeed())
			Expect(createOperatorGroup(CloudpakOgYamlObj)).Should(Succeed())
			Expect(createSubscription(CloudpakCsSubYamlObj)).Should(Succeed())

			By("Checking common service operator deployment status")
			Eventually(waitForDeploymentReady(CommonServiceOperatorName, CommonServiceOperatorNamespace), timeout, interval).Should(BeTrue())

			By("Checking ODLM status")
			Eventually(waitForDeploymentReady(OdlmOperatorName, constant.ClusterOperatorNamespace),
				timeout, interval).Should(BeTrue())
			By("Checking secretshare status")
			Eventually(waitForDeploymentReady("secretshare", CommonServiceOperatorNamespace), timeout, interval).Should(BeTrue())

			By("Checking common service webhook status")
			Eventually(waitForDeploymentReady("ibm-common-service-webhook", CommonServiceOperatorNamespace), timeout, interval).Should(BeTrue())
		})
	})

	Context("Install Common Services", func() {
		It("Should common services were installed", func() {
			By("Create Common Service OperandRequest")
			Expect(createOperandRequest(OpreqYamlObj)).Should(Succeed())
			opreq := util.NewUnstructured("operator.ibm.com", "OperandRequest", "v1alpha1")
			opreqKey := types.NamespacedName{Name: CommonServiceOperandName, Namespace: CommonServiceOperatorNamespace}
			Eventually(func() bool {
				if err := k8sReader.Get(ctx, opreqKey, opreq); err != nil {
					return false
				}
				if opreq.Object["status"].(map[string]interface{})["phase"].(string) != "Running" {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Update Common Services Size to medium", func() {
		It("Should be applied into OperandConfig", func() {
			By("Update CommonService operand commonservice")
			cs := &apiv3.CommonService{}
			csKey := types.NamespacedName{Name: CommonServiceOperandName, Namespace: CommonServiceOperatorNamespace}

			Expect(k8sReader.Get(ctx, csKey, cs)).Should(Succeed())
			cs.Spec.Size = "medium"
			Expect(k8sClient.Update(ctx, cs)).Should(Succeed())
			Expect(k8sReader.Get(ctx, csKey, cs)).Should(Succeed())

			Expect(cs.Spec.Size).To(Equal("medium"), "OperandConfig common-service ")

		})
	})

	Context("Uninstall Common Services and Cleanup environment", func() {
		It("Should be cleanup", func() {

			By("Delete CommonService instance")
			csKey := types.NamespacedName{Name: CommonServiceOperandName, Namespace: CommonServiceOperatorNamespace}
			cs := &apiv3.CommonService{}
			Expect(k8sReader.Get(ctx, csKey, cs)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, cs)).Should(Succeed())

			Eventually(func() bool {
				err := k8sReader.Get(ctx, csKey, cs)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			By("Delete cloudpak-ns namespace")
			Expect(deleteNamespace(CloudPakNamespace)).Should(Succeed())
		})
	})
})

func waitForDeploymentReady(name, namespace string) bool {
	klog.Infof("Waiting for deployment %s/%s ready.", namespace, name)
	deployKey := types.NamespacedName{Name: name, Namespace: namespace}
	deploy := &appsv1.Deployment{}
	if err := k8sReader.Get(ctx, deployKey, deploy); err != nil {
		if !errors.IsNotFound(err) {
			klog.Error("Get deployment failed: ", err)
		} else {
			klog.Error("Cannot found deployment failed: ", err)
		}
		return false
	}
	if deploy.Status.ReadyReplicas != deploy.Status.Replicas {
		return false
	}
	return true
}

// Create namespace resource obj
func createNamespace(name string) error {
	nsObj := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := k8sClient.Create(ctx, nsObj); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// Delete namespace resource obj
func deleteNamespace(name string) error {
	nsObj := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := k8sClient.Delete(ctx, nsObj); err != nil {
		return err
	}
	return nil
}

// Create OperandRequest obj
func createOperandRequest(request string) error {
	if err := deployMgr.CreateFromYaml([]byte(request)); err != nil {
		return err
	}
	return nil
}

// Create operator subscription
func createSubscription(sub string) error {
	if err := deployMgr.CreateFromYaml([]byte(sub)); err != nil {
		return err
	}
	return nil
}

// Create operator group
func createOperatorGroup(og string) error {
	if err := deployMgr.CreateFromYaml([]byte(og)); err != nil {
		return err
	}
	return nil
}

const CloudpakCsSubYamlObj = `
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: cs-for-cloudpak
  namespace: cloudpak-ns
spec:
  channel: dev
  installPlanApproval: Automatic
  name: ibm-common-service-operator
  source: opencloud-operators
  sourceNamespace: openshift-marketplace
`

// CloudpakOgYamlObj is OperatorGroup constent for the cloudpak-ns namespace
const CloudpakOgYamlObj = `
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: cloudpack-cs-operatorgroup
  namespace: cloudpak-ns
spec:
  targetNamespaces:
  - cloudpak-ns
`

const OpreqYamlObj = `
apiVersion: operator.ibm.com/v1alpha1
kind: OperandRequest
metadata:
  namespace: ibm-common-services
  name: common-service
spec:
  requests:
  - registry: common-service
    registryNamespace: ibm-common-services
    operands:
      - name: ibm-mongodb-operator
`
