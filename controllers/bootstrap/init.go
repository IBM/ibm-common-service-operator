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

package bootstrap

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	CsExtResource           = "extraResources"
	CsSubResource           = []string{"csOperatorSubscription"}
	OdlmNsSubResources      = []string{"odlmNsSubscription"}
	OdlmClusterSubResources = []string{"odlmClusterSubscription"}
	OdlmCrResources         = []string{"csOperandRegistry", "csOperandConfig"}
	OdlmClusterPermission   = []string{"odlmClusterRole", "odlmClusterRoleBinding"}
)

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
func (b *Bootstrap) InitResources() error {
	// Get all the resources from the deployment annotations
	annotations, err := b.GetAnnotations()
	if err != nil {
		return err
	}

	operatorNs, err := util.GetOperatorNamespace()
	if err != nil {
		klog.Error("get operator namespace failed: ", err)
		return err
	}

	// Grant cluster-admin to namespace scope operator
	if operatorNs == constant.ClusterOperatorNamespace {
		klog.Info("Creating cluster-admin permission RBAC")
		clusterRBAC := fmt.Sprintf(constant.ClusterAdminRBAC, constant.MasterNamespace)
		if err := b.createOrUpdateFromYaml([]byte(clusterRBAC)); err != nil {
			return err
		}
	}

	// Install Namespace Scope Operator

	// Create extra RBAC for ibmcloud-cluster-ca-cert and ibmcloud-cluster-info in kube-public
	klog.Info("Creating RBAC for ibmcloud-cluster-info & ibmcloud-cluster-ca-cert in kube-public")
	if err := b.createOrUpdateFromYaml([]byte(constant.ExtraRBAC)); err != nil {
		return err
	}

	// create or update ODLM operator
	if operatorNs == constant.MasterNamespace {
		klog.Info("create odlm in namespace ibm-common-services")
		if err := b.createOrUpdateResources(annotations, OdlmNsSubResources); err != nil {
			return err
		}
	} else if operatorNs == constant.ClusterOperatorNamespace {
		klog.Info("create operator group in namespace ibm-common-services")
		if err := b.createOperatorGroup(); err != nil {
			return err
		}
		klog.Info("create odlm in namespace ibm-common-services")
		if err := b.createOrUpdateResources(annotations, OdlmClusterSubResources); err != nil {
			return err
		}
		klog.Info("create odlm cluster permission for ODLM")
		if err := b.createOrUpdateResources(annotations, OdlmClusterPermission); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported namespace")
	}

	// create or update extra resources for common services
	if err := b.createOrUpdateResources(annotations, strings.Split(annotations[CsExtResource], ",")); err != nil {
		return err
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
	klog.Info("create operator group in namespace ibm-common-services")
	if err := b.createOperatorGroup(); err != nil {
		return err
	}
	klog.Info("create cs operator in namespace ibm-common-services")
	if err := b.createOrUpdateResources(annotations, CsSubResource); err != nil {
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
		return b.createOrUpdateFromYaml([]byte(constant.CsCR))
	} else if err != nil {
		return err
	}

	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	cs.SetName("common-service")
	cs.SetNamespace("ibm-common-services")
	_, err = b.GetObject(cs)
	if errors.IsNotFound(err) {
		// Upgrade: Have ODLM and NO CR
		return b.createOrUpdateFromYaml([]byte(constant.CsNoSizeCR))
	} else if err != nil {
		return err
	}

	// Restart: Have ODLM and CR
	return b.createOrUpdateFromYaml([]byte(constant.CsCR))
}

func (b *Bootstrap) createOperatorGroup() error {
	existOG := &olmv1.OperatorGroupList{}
	if err := b.Reader.List(context.TODO(), existOG, &client.ListOptions{Namespace: "ibm-common-services"}); err != nil {
		return err
	}
	if len(existOG.Items) == 0 {
		if err := b.createOrUpdateFromYaml([]byte(constant.CsOg)); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bootstrap) createOrUpdateResources(annotations map[string]string, resNames []string) error {
	for _, res := range resNames {
		if r, ok := annotations[res]; ok {
			if err := b.createOrUpdateFromYaml([]byte(r)); err != nil {
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
			klog.Infof("create resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
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
			klog.Infof("update resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
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
