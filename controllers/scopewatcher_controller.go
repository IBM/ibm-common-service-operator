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

var (
	odlmCRs = []*bootstrap.Resource{
		{
			Name:    constant.MasterCR,
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandRegistry",
			Scope:   "namespaceScope",
		},
		{
			Name:    constant.MasterCR,
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandConfig",
			Scope:   "namespaceScope",
		},
	}

	nssCRs = []*bootstrap.Resource{
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

	certManagerCR = []*bootstrap.Resource{
		{
			Name:    "default",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "CertManager",
			Scope:   "clusterScope",
		},
	}

	licensingCR = []*bootstrap.Resource{
		{
			Name:    "instance",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "IBMLicensing",
			Scope:   "clusterScope",
		},
	}

	cp2Deployments = []*bootstrap.Resource{
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

	cp2Resources = []*bootstrap.Resource{
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

	licensingConfigMaps = []string{
		"ibm-licensing-annotations",
		"ibm-licensing-products",
		"ibm-licensing-products-vpc-hour",
		"ibm-licensing-cloudpaks",
		"ibm-licensing-products-groups",
		"ibm-licensing-cloudpaks-groups",
		"ibm-licensing-cloudpaks-metrics",
		"ibm-licensing-products-metrics",
		"ibm-licensing-products-metrics-groups",
		"ibm-licensing-cloudpaks-metrics-groups",
		"ibm-licensing-services",
	}

	licenseservicereporterCR = []*bootstrap.Resource{
		{
			Name:    "instance",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "IBMLicenseServiceReporter",
			Scope:   "clusterScope",
		},
	}
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

	isMaintained, err := util.GetMaintenanceMode(r.Client, constant.MasterCR, r.Bootstrap.CSData.MasterNs)
	if err != nil {
		klog.Errorf("Failed to get maintenance mode: %v", err)
		return ctrl.Result{}, err
	}

	// Determine if the cluster is multi instance enabled, if it is not enabled, no isolation process is required
	MultiInstanceStatusFromCluster := util.CheckMultiInstances(r.Reader)
	if !MultiInstanceStatusFromCluster {
		klog.Infof("MultiInstancesEnable is not enabled in cluster, skip isolation process")
		if isMaintained {
			if err = util.DisableMaintenanceMode(r.Client, constant.MasterCR, r.Bootstrap.CSData.MasterNs); err != nil {
				klog.Errorf("Failed to disable maintenance mode: %v", err)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	r.Bootstrap.CSData.ControlNs = util.GetControlNs(r.Reader)
	if err := r.Bootstrap.CreateNamespace(r.Bootstrap.CSData.ControlNs); err != nil {
		klog.Errorf("Failed to create control namespace: %v", err)
		return ctrl.Result{}, err
	}

	// Get all namespaces which are not part of existing tenant scope from ConfigMap
	excludedScope, err := util.GetExcludedScope(cm, r.Bootstrap.CSData.MasterNs)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Get existing scope from `namespace-scope` ConfigMap in MasterNs
	nsScope := util.GetNssCmNs(r.Client, r.Bootstrap.CSData.MasterNs)
	// Compare ns_scope and excludedScope and get the to-be-detached namespace
	excludedNsList := util.FindIntersection(nsScope, excludedScope)
	// Get the latest tenant scope by removing the to-be-detached namespace from existing scope
	updatedNsList := util.FindDifference(nsScope, excludedNsList)

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
	if err = util.EnableMaintenanceMode(r.Client, constant.MasterCR, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to enable maintenance mode: %v", err)
		return ctrl.Result{}, err
	}

	// Refresh CommonService Operator memory/cache to re-construct the tenant scope.
	klog.Infof("Converting MultiInstancesEnable from %v to %v", r.Bootstrap.MultiInstancesEnable, MultiInstanceStatusFromCluster)
	if !r.Bootstrap.MultiInstancesEnable && MultiInstanceStatusFromCluster {
		if err := util.TurnOffRouteChangeInMgmtIngress(r.Client, constant.MasterCR, r.Bootstrap.CSData.MasterNs); err != nil {
			klog.Errorf("Failed to keep Route unchanged for %s/common-service: %v", r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
		klog.Infof("MultiInstancesEnable is changed from %v to %v", r.Bootstrap.MultiInstancesEnable, MultiInstanceStatusFromCluster)
		r.Bootstrap.MultiInstancesEnable = MultiInstanceStatusFromCluster
	}

	// Delete existing OperandConfig and OperandRegistry CRs

	for _, cr := range odlmCRs {
		if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Re-construct CP2 tenant scope
	// 1. Re-construct the NamespaceScope CRs in ibm-common-services namespace.
	klog.Infof("Updating NamespaceScope CRs in %s by adding %v and removing %v", r.Bootstrap.CSData.MasterNs, updatedNsList, excludedNsList)
	if err := util.UpdateNsToNSS(r.Reader, r.Client, constant.MasterCR, r.Bootstrap.CSData.MasterNs, updatedNsList, excludedNsList); err != nil {
		klog.Errorf("Failed to update namespaces in NamespaceScope CR %s/%s: %v", r.Bootstrap.CSData.MasterNs, constant.MasterCR, err)
		return ctrl.Result{}, err
	}

	klog.Infof("Updating NamespaceScope CRs in nss-odlm-scope by adding %v and removing %v", updatedNsList, excludedNsList)
	if err := util.UpdateNsToNSS(r.Reader, r.Client, "nss-odlm-scope", r.Bootstrap.CSData.MasterNs, updatedNsList, excludedNsList); err != nil {
		klog.Errorf("Failed to update namespaces in NamespaceScope CR %s/%s: %v", r.Bootstrap.CSData.MasterNs, "nss-odlm-scope", err)
		return ctrl.Result{}, err
	}

	// 2. Delete NamespaceScope CRs managed by ODLM
	for _, cr := range nssCRs {
		if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. Patch ODLM subscription
	klog.Infof("Isolating ODLM in %s by excluding %v", r.Bootstrap.CSData.MasterNs, excludedNsList)
	if err := r.Bootstrap.IsolateODLM(excludedNsList); err != nil {
		klog.Errorf("Failed to isolate ODLM: %v", err)
		return ctrl.Result{}, err
	}

	// Migrate datum
	// 1. Migrate Licensing data
	klog.Infof("Migrating Licensing data from %s to %s", r.Bootstrap.CSData.MasterNs, r.Bootstrap.CSData.ControlNs)
	for _, licensingCm := range licensingConfigMaps {
		if err := util.MigrateConfigMap(r.Reader, r.Client, licensingCm, r.Bootstrap.CSData.MasterNs, r.Bootstrap.CSData.ControlNs); err != nil {
			klog.Errorf("Failed to migrate ConfigMap %s: %v", licensingCm, err)
			return ctrl.Result{}, err
		}
		// Delete the ConfigMap in MasterNs
		if err := util.DeleteConfigMap(r.Client, licensingCm, r.Bootstrap.CSData.MasterNs); err != nil {
			klog.Errorf("Failed to delete ConfigMap %s: %v", licensingCm, err)
			return ctrl.Result{}, err
		}
	}

	// 2. Backup and Delete Licensing CR
	LicensingDeploy := &appsv1.Deployment{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      constant.LicensingSub,
		Namespace: r.Bootstrap.CSData.MasterNs,
	}, LicensingDeploy); err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("%s deployment is not found in %s", constant.LicensingSub, r.Bootstrap.CSData.MasterNs)
		} else {
			klog.Errorf("Failed to get Deployment %s in %s: %v", constant.LicensingSub, r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
	} else {
		klog.Info("Deleting Licensing CR for migration")
		for _, cr := range licensingCR {
			if err := r.Bootstrap.BackupCRtoCm(r.Bootstrap.CSData.MasterNs, "ibmlicensing-instance-bak", "ibmlicensing.yaml", r.Bootstrap.CSData.ControlNs, cr); err != nil {
				klog.Errorf("Failed to backup %s/%s in %s: %v", cr.Kind, cr.Name, r.Bootstrap.CSData.MasterNs, err)
				return ctrl.Result{}, err
			}
			if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, cr); err != nil {
				klog.Errorf("Failed to delete %s in %s: %v", cr.Name, r.Bootstrap.CSData.MasterNs, err)
				return ctrl.Result{}, err
			}
		}
	}

	// 3. Delete Licensing Operator
	klog.Infof("Deleting operator %s in %s", constant.LicensingSub, r.Bootstrap.CSData.MasterNs)
	if err := r.Bootstrap.DeleteOperator(constant.LicensingSub, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", constant.LicensingSub, r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	// 4. Restore Licensing CR if Licensing operator deployment does not exist in ControlNs
	LicensingDeploy = &appsv1.Deployment{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      constant.LicensingSub,
		Namespace: r.Bootstrap.CSData.ControlNs,
	}, LicensingDeploy); err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("%s deployment is not found in %s", constant.LicensingSub, r.Bootstrap.CSData.MasterNs)
			klog.Infof("Restoring Licensing CR from ConfigMap ibmlicensing-instance-bak in %s", r.Bootstrap.CSData.ControlNs)
			if err := r.Bootstrap.RestoreCmtoCR("ibmlicensing-instance-bak", r.Bootstrap.CSData.ControlNs, "ibmlicensing.yaml"); err != nil {
				klog.Errorf("Failed to restore Licensing CR in %s: %v", r.Bootstrap.CSData.ControlNs, err)
				return ctrl.Result{}, err
			}
			// delete the `ibmlicensing-instance-bak` ConfigMap after restore
			if err := util.DeleteConfigMap(r.Client, "ibmlicensing-instance-bak", r.Bootstrap.CSData.ControlNs); err != nil {
				klog.Errorf("Failed to delete ConfigMap ibmlicensing-instance-bak in %s: %v", r.Bootstrap.CSData.ControlNs, err)
				return ctrl.Result{}, err
			}
		} else {
			klog.Errorf("Failed to get Deployment %s in %s: %v", constant.LicensingSub, r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}

		klog.Infof("Upding Licensing CR with setting a template for sender configuration and creating a secret")
		for _, cr := range licensingCR {
			if err := r.Bootstrap.UpdateLicensingCR(r.Bootstrap.CSData.ControlNs, cr); err != nil {
				klog.Errorf("Failed to update Licensing CR in %s: %v", r.Bootstrap.CSData.ControlNs, err)
				return ctrl.Result{}, err
			}
		}
	} else {
		klog.Infof("%s deployment is found in %s, skipping restore Licensing CR", constant.LicensingSub, r.Bootstrap.CSData.MasterNs)
	}

	// 5. Migrate Cert-Manager
	certManagerDeploy := &appsv1.Deployment{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      constant.CertManagerSub,
		Namespace: r.Bootstrap.CSData.MasterNs,
	}, certManagerDeploy); err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("%s deployment is not found in %s", constant.CertManagerSub, r.Bootstrap.CSData.MasterNs)
		} else {
			klog.Errorf("Failed to get Deployment %s in %s: %v", constant.CertManagerSub, r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
	} else {
		klog.Info("Deleting Cert-Manager CR for migration")
		for _, cr := range certManagerCR {
			if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, cr); err != nil {
				klog.Errorf("Failed to delete %s in %s: %v", cr.Name, r.Bootstrap.CSData.MasterNs, err)
				return ctrl.Result{}, err
			}
		}
	}

	klog.Infof("Deleting operator %s in %s", constant.CertManagerSub, r.Bootstrap.CSData.MasterNs)
	if err := r.Bootstrap.DeleteOperator(constant.CertManagerSub, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", constant.CertManagerSub, r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	// 6. Delete Crossplane, webhook, and secretshare deployment
	klog.Infof("Deleting operator %s in %s", constant.ICPPKOperator, r.Bootstrap.CSData.MasterNs)
	if err := r.Bootstrap.DeleteOperator(constant.ICPPKOperator, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", constant.ICPPKOperator, r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	klog.Infof("Deleting operator %s in %s", constant.ICPPICOperator, r.Bootstrap.CSData.MasterNs)
	if err := r.Bootstrap.DeleteOperator(constant.ICPPICOperator, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", constant.ICPPICOperator, r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	klog.Infof("Deleting operator %s in %s", constant.ICPOperator, r.Bootstrap.CSData.MasterNs)
	if err := r.Bootstrap.DeleteOperator(constant.ICPOperator, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to delete operator %s in %s: %v", constant.ICPOperator, r.Bootstrap.CSData.MasterNs, err)
		return ctrl.Result{}, err
	}

	// 7. Remove webhook and secretshare
	for _, deployment := range cp2Deployments {
		if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, deployment); err != nil {
			klog.Errorf("Failed to delete %s in %s: %v", deployment.Name, r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
	}

	for _, resource := range cp2Resources {
		if err := r.Bootstrap.Cleanup(r.Bootstrap.CSData.MasterNs, resource); err != nil {
			klog.Errorf("Failed to delete %s in %s: %v", resource.Name, r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
	}

	// 8. Isolate License Service Reporter
	klog.Infof("Isolating License Service Reporter")
	for _, cr := range licenseservicereporterCR {
		if err := r.Bootstrap.IsolateLSR(r.Bootstrap.CSData.MasterNs, cr); err != nil {
			klog.Errorf("Failed to isolate License Service Reporter: %v", err)
			return ctrl.Result{}, err
		}
	}

	klog.Infof("Scaling up ODLM to 1 in %s", r.Bootstrap.CSData.MasterNs)
	if err := util.ScaleOperator(r.Reader, r.Client, "ibm-odlm", r.Bootstrap.CSData.MasterNs, 1); err != nil {
		klog.Errorf("Failed to scale down ODLM: %v", err)
		return ctrl.Result{}, err
	}

	// Release the maintenance mode on CS CR reconciliation
	if err = util.DisableMaintenanceMode(r.Client, constant.MasterCR, r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to disable maintenance mode: %v", err)
		return ctrl.Result{}, err
	}

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
