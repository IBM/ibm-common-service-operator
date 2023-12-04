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

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
)

func (r *CommonServiceReconciler) ScopeReconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	klog.Infof("Reconciling ConfigMap: %s", req.NamespacedName)

	// Validate common-service-maps and filter the namespace of CommonService CR
	cm, err := util.GetCmOfMapCs(r.Client)
	if err == nil {
		if err := util.ValidateCsMaps(cm); err != nil {
			klog.Errorf("Unsupported common-service-maps: %v", err)
			return reconcile.Result{RequeueAfter: constant.DefaultRequeueDuration}, err
		}
	} else if !errors.IsNotFound(err) {
		klog.Errorf("Failed to get common-service-maps: %v", err)
		return ctrl.Result{}, err
	}

	isMaintained, err := util.GetMaintenanceMode(r.Client, "common-service", r.Bootstrap.CSData.MasterNs)
	if err != nil {
		klog.Errorf("Failed to get maintenance mode: %v", err)
		return ctrl.Result{}, err
	}

	// Determine if the cluster is multi instance enabled, if it is not enabled, no isolation process is required
	MultiInstanceStatusFromCluster := util.CheckMultiInstances(r.Reader)
	if !MultiInstanceStatusFromCluster {
		klog.Infof("MultiInstancesEnable is not enabled in cluster, skip isolation process")
		if isMaintained {
			if err = util.DisableMaintenanceMode(r.Client, "common-service", r.Bootstrap.CSData.MasterNs); err != nil {
				klog.Errorf("Failed to disable maintenance mode: %v", err)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	r.Bootstrap.CSData.ControlNs = util.GetControlNs(r.Reader)

	// Get all namespaces which are not part of existing tenant scope from ConfigMap
	excludedScope, err := util.GetExcludedScope(cm, r.Bootstrap.CSData.MasterNs)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Get existing scope from `namespace-scope` ConfigMap in MasterNs
	nsScope := util.GetNssCmNs(r.Client, r.Bootstrap.CSData.MasterNs)
	// Compare ns_scope and excludedScope and get the to-be-detached namespace
	excludedNsList := util.FindIntersection(nsScope, excludedScope)

	// Only checking ExcludedNsList is not reliable.
	// If an error happens AFTER we update NamespaceScope CR to remove ExcludedNsList but BEFORE we finish the isolation, then in the second reconciliation here, CS will skip isolation because Excluded Ns List is already empty.
	// We need to have a marker(use pause annotation) to indicate that previous isolation is not done yet(a error happened during isolation), and continue isolation even ExcludedNsList is empty
	// If the intersection is empty, there is no isolation process required
	if len(excludedNsList) == 0 && !isMaintained {
		klog.Infof("Existing Common Service tenant scope contains following namespaces: %v, there is no isolation process required", nsScope)
		return ctrl.Result{}, nil
	}

	// Scale down ODLM deployment to 0 immediately to avoid ODLM reconcile CRs in the to-be-detached namespaces
	klog.Infof("Scaling down ODLM to 0 in %s", r.Bootstrap.CSData.MasterNs)
	if err := util.ScaleOperator(r.Reader, r.Client, "ibm-odlm", r.Bootstrap.CSData.MasterNs, 0); err != nil {
		klog.Errorf("Failed to scale down ODLM: %v", err)
		return ctrl.Result{}, err
	}

	// Silence CS 3.x CR reconciliation by enabling maintenance mode
	if err = util.EnableMaintenanceMode(r.Client, "common-service", r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to enable maintenance mode: %v", err)
		return ctrl.Result{}, err
	}

	// Refresh CommonService Operator memory/cache to re-construct the tenant scope.
	klog.Infof("Converting MultiInstancesEnable from %v to %v", r.Bootstrap.MultiInstancesEnable, MultiInstanceStatusFromCluster)
	if !r.Bootstrap.MultiInstancesEnable && MultiInstanceStatusFromCluster {
		if err := util.TurnOffRouteChangeInMgmtIngress(r.Client, "common-service", r.Bootstrap.CSData.MasterNs); err != nil {
			klog.Errorf("Failed to keep Route unchanged for %s/common-service: %v", r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
		klog.Infof("MultiInstancesEnable is changed from %v to %v", r.Bootstrap.MultiInstancesEnable, MultiInstanceStatusFromCluster)
		r.Bootstrap.MultiInstancesEnable = MultiInstanceStatusFromCluster
	}

	// Delete existing OperandConfig and OperandRegistry CRs
	var ODLMCRs = []*bootstrap.Resource{
		{
			Name:    "common-service",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandRegistry",
			Scope:   "namespaceScope",
		},
		{
			Name:    "common-service",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandConfig",
			Scope:   "namespaceScope",
		},
	}
	for _, cr := range ODLMCRs {
		if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Re-construct CP2 tenant scope
	// Re-construct the NamespaceScope CRs in ibm-common-services namespace.
	var NSSCRs = []*bootstrap.Resource{
		{
			Name:    "nss-managedby-odlm",
			Version: "v1",
			Group:   "operator.ibm.com",
			Kind:    "NamespaceScope",
			Scope:   "namespaceScope",
		},
		{
			Name:    "odlm-scope-managedby-odlm",
			Version: "v1",
			Group:   "operator.ibm.com",
			Kind:    "NamespaceScope",
			Scope:   "namespaceScope",
		},
	}
	for _, cr := range NSSCRs {
		if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	klog.Infof("Re-constructing NamespaceScope CRs in %s by removing %v", r.Bootstrap.CSData.MasterNs, excludedNsList)
	if err := util.ExcludeNsFromNSS(r.Reader, r.Client, "common-service", r.Bootstrap.CSData.MasterNs, excludedNsList); err != nil {
		klog.Errorf("Failed to exclude namespaces from NamespaceScope CR %s/%s: %v", r.Bootstrap.CSData.MasterNs, "common-service", err)
		return ctrl.Result{}, err
	}

	if err := util.ExcludeNsFromNSS(r.Reader, r.Client, "nss-odlm-scope", r.Bootstrap.CSData.MasterNs, excludedNsList); err != nil {
		klog.Errorf("Failed to exclude namespaces from NamespaceScope CR %s/%s: %v", r.Bootstrap.CSData.MasterNs, "nss-odlm-scope", err)
		return ctrl.Result{}, err
	}

	// 2. Patch ODLM subscription
	klog.Infof("Isolating ODLM in %s by excluding %v", r.Bootstrap.CSData.MasterNs, excludedNsList)
	if err := r.Bootstrap.IsolateODLM(excludedNsList); err != nil {
		klog.Errorf("Failed to isolate ODLM: %v", err)
		return ctrl.Result{}, err
	}

	// TODO: Migrate old services by following isolate documentation

	// 1. Migrate Licensing data
	// 2. Backup Licensing CR
	// 3. Delete Licensing Operator
	// 4. Delete Licensing CR
	// Check if ibm-licensing-operator deployment exists or not
	LicensingDeploy := &appsv1.Deployment{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      "ibm-licensing-operator",
		Namespace: r.Bootstrap.CSData.MasterNs,
	}, LicensingDeploy); err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("ibm-licensing-operator deployment is not found in %s", r.Bootstrap.CSData.MasterNs)
		} else {
			klog.Errorf("Failed to get Deployment %s in %s: %v", "ibm-licensing-operator", r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
	} else {
		var LicensingCR = []*bootstrap.Resource{
			{
				Name:    "instance",
				Version: "v1alpha1",
				Group:   "operator.ibm.com",
				Kind:    "IBMLicensing",
				Scope:   "clusterScope",
			},
		}
		for _, cr := range LicensingCR {
			if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, cr); err != nil {
				klog.Errorf("Failed to delete %s in %s: %v", cr.Name, r.Bootstrap.CSData.MasterNs, err)
				return ctrl.Result{}, err
			}
		}
	}

	if err := r.Bootstrap.DeleteSubscription("ibm-licensing-operator", r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", "ibm-licensing-operator", r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	// 5. Restore Licensing CR

	// 6. Migrate Cert-Manager
	var CertManagerCR = []*bootstrap.Resource{
		{
			Name:    "default",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "CertManager",
			Scope:   "clusterScope",
		},
	}
	for _, cr := range CertManagerCR {
		if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, cr); err != nil {
			klog.Errorf("Failed to delete %s in %s: %v", cr.Name, r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
	}

	if err := r.Bootstrap.DeleteSubscription(constant.CertManagerSub, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", constant.CertManagerSub, r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	// 7. Delete Crossplane, webhook, and secretshare deployment
	var CP2Deployments = []*bootstrap.Resource{
		{
			Name:    "secretshare",
			Version: "v1",
			Group:   "apps",
			Kind:    "Deployment",
			Scope:   "namespaceScope",
		},
		{
			Name:    "ibm-common-service-webhook",
			Version: "v1",
			Group:   "apps",
			Kind:    "Deployment",
			Scope:   "namespaceScope",
		},
	}

	var CP2Resources = []*bootstrap.Resource{
		{
			Name:    "ibm-common-service-webhook",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "PodPreset",
			Scope:   "namespaceScope",
		},
		{
			Name:    "ibm-common-service-webhook-configuration",
			Version: "v1",
			Group:   "admissionregistration.k8s.io",
			Kind:    "MutatingWebhookConfiguration",
			Scope:   "clusterScope",
		},
		{
			Name:    "ibm-operandrequest-webhook-configuration",
			Version: "v1",
			Group:   "admissionregistration.k8s.io",
			Kind:    "MutatingWebhookConfiguration",
			Scope:   "clusterScope",
		},
		{
			Name:    "ibm-cs-ns-mapping-webhook-configuration",
			Version: "v1",
			Group:   "admissionregistration.k8s.io",
			Kind:    "ValidatingWebhookConfiguration",
			Scope:   "clusterScope",
		},
		{
			Name:    "common-services",
			Version: "v1",
			Group:   "ibmcpcs.ibm.com",
			Kind:    "SecretShare",
			Scope:   "namespaceScope",
		},
	}

	// remove crossplane
	klog.Infof("Deleting operator %s in %s", constant.ICPPKOperator, r.Bootstrap.CSData.MasterNs)
	if err := r.Bootstrap.DeleteSubscription(constant.ICPPKOperator, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", constant.ICPPKOperator, r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	klog.Infof("Deleting operator %s in %s", constant.ICPPICOperator, r.Bootstrap.CSData.MasterNs)
	if err := r.Bootstrap.DeleteSubscription(constant.ICPPICOperator, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", constant.ICPPICOperator, r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	klog.Infof("Deleting operator %s in %s", constant.ICPOperator, r.Bootstrap.CSData.MasterNs)
	if err := r.Bootstrap.DeleteSubscription(constant.ICPOperator, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", constant.ICPOperator, r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	// if updateErr := r.Bootstrap.CreateorUpdateCFCrossplaneConfigMap("'true'"); updateErr != nil {
	// 	return ctrl.Result{}, updateErr
	// }

	// remove webhook and secretshare
	for _, deployment := range CP2Deployments {
		if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, deployment); err != nil {
			klog.Errorf("Failed to delete %s in %s: %v", deployment.Name, r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
	}

	for _, resource := range CP2Resources {
		if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, resource); err != nil {
			klog.Errorf("Failed to delete %s in %s: %v", resource.Name, r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
	}

	klog.Infof("Scaling up ODLM to 1 in %s", r.Bootstrap.CSData.MasterNs)
	if err := util.ScaleOperator(r.Reader, r.Client, "ibm-odlm", r.Bootstrap.CSData.MasterNs, 1); err != nil {
		klog.Errorf("Failed to scale down ODLM: %v", err)
		return ctrl.Result{}, err
	}

	// Release the maintenance mode on CS CR reconciliation
	if err = util.DisableMaintenanceMode(r.Client, "common-service", r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to disable maintenance mode: %v", err)
		return ctrl.Result{}, err
	}

	// Get the latest tenant scope by removing the to-be-detached namespace from existing scope
	updatedNsList := util.FindDifference(nsScope, excludedNsList)
	// Get the existing tenant scope configuration from ConfigMap
	csScope, err := util.GetCsScope(cm, r.Bootstrap.CSData.MasterNs)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Common Service v3 should not manipulate tenant scope entry in ConfigMap if this v3 tenant has been added into ConfigMap by user
	// if no entry found in common-service-maps ConfigMap, then add latest scope into the ConfigMap
	if len(csScope) == 0 {
		klog.Infof("No entry found in common-service-maps ConfigMap for %s", r.Bootstrap.CSData.MasterNs)

		// Update the ConfigMap to add latest scope, it should be the updatedNsList as requested-from-namespace, and MasterNs as map-to-common-service-namespace
		klog.Infof("Updating namespace mapping with latest scope %v for %s", updatedNsList, r.Bootstrap.CSData.MasterNs)
		if err := util.UpdateCsMaps(cm, updatedNsList, r.Bootstrap.CSData.MasterNs); err != nil {
			klog.Errorf("Failed to update common-service-maps: %v", err)
			return ctrl.Result{}, err
		}
		// Validate common-service-maps
		if err := util.ValidateCsMaps(cm); err != nil {
			klog.Errorf("Unsupported common-service-maps: %v", err)
			return ctrl.Result{}, err
		}
		if err := r.Client.Update(context.TODO(), cm); err != nil {
			klog.Errorf("Failed to update namespaceMapping in common-service-maps: %v", err)
			return ctrl.Result{}, err
		}
	}

	klog.Infof("Existing Common Service tenant scope has been isolated with namespaces %v", updatedNsList)

	return ctrl.Result{}, nil

}
