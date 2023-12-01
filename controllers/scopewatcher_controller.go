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

	"k8s.io/apimachinery/pkg/api/errors"
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

	// Determine if the cluster is multi instance enabled, if it is not enabled, no isolation process is required
	r.Bootstrap.CSData.ControlNs = util.GetControlNs(r.Reader)
	MultiInstanceStatusFromCluster := util.CheckMultiInstances(r.Reader)
	if !MultiInstanceStatusFromCluster {
		klog.Infof("MultiInstancesEnable is not enabled in cluster, skip isolation process")
		return ctrl.Result{}, nil
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

	// If the intersection is empty, there is no isolation process required
	if len(excludedNsList) == 0 {
		klog.Infof("Existing Common Service tenant scope contains following namespaces: %v, there is no isolation process required", nsScope)
		return ctrl.Result{}, nil
	}

	// Silence CS 3.x CR reconciliation by enabling maintenance mode
	if err = util.EnableMaintenanceMode(r.Client, "common-service", r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to enable maintenance mode: %v", err)
		return ctrl.Result{}, err
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

	// TODO: Re-construct CP2 tenant scope
	// 1. Refresh CommonService Operator memory/cache to re-construct the tenant scope.
	klog.Infof("Converting MultiInstancesEnable from %v to %v", r.Bootstrap.MultiInstancesEnable, MultiInstanceStatusFromCluster)
	if !r.Bootstrap.MultiInstancesEnable && MultiInstanceStatusFromCluster {
		if err := util.TurnOffRouteChangeInMgmtIngress(r.Client, "common-service", r.Bootstrap.CSData.MasterNs); err != nil {
			klog.Errorf("Failed to keep Route unchanged for %s/common-service: %v", r.Bootstrap.CSData.MasterNs, err)
			return ctrl.Result{}, err
		}
		klog.Infof("MultiInstancesEnable is changed from %v to %v", r.Bootstrap.MultiInstancesEnable, MultiInstanceStatusFromCluster)
		r.Bootstrap.MultiInstancesEnable = MultiInstanceStatusFromCluster
	}

	// 2. Re-construct the NamespaceScope CRs in ibm-common-services namespace.
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

	if err := util.ExcludeNsFromNSS(r.Reader, r.Client, "common-service", r.Bootstrap.CSData.MasterNs, excludedNsList); err != nil {
		klog.Errorf("Failed to exclude namespaces from NamespaceScope CR %s/%s: %v", r.Bootstrap.CSData.MasterNs, "common-service", err)
		return ctrl.Result{}, err
	}

	if err := util.ExcludeNsFromNSS(r.Reader, r.Client, "nss-odlm-scope", r.Bootstrap.CSData.MasterNs, excludedNsList); err != nil {
		klog.Errorf("Failed to exclude namespaces from NamespaceScope CR %s/%s: %v", r.Bootstrap.CSData.MasterNs, "nss-odlm-scope", err)
		return ctrl.Result{}, err
	}

	// 2. Patch ODLM subscription
	if err := r.Bootstrap.IsolateODLM(excludedNsList); err != nil {
		klog.Errorf("Failed to isolate ODLM: %v", err)
		return ctrl.Result{}, err
	}

	// TODO: Migrate old services by following isolate documentation

	// 1. Migrate Licensing data
	// 2. Backup Licensing CR
	// 3. Delete Licensing Operator
	// 4. Delete Licensing CR
	// 5. Restore Licensing CR

	// 6. Migrate Cert-Manager

	// 7. Delete Crossplane, webhook, and secretshare deployment

	// Release the maintenance mode on CS CR reconciliation
	if err = util.DisableMaintenanceMode(r.Client, "common-service", r.Bootstrap.CSData.MasterNs); err != nil {
		klog.Errorf("Failed to disable maintenance mode: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil

}
