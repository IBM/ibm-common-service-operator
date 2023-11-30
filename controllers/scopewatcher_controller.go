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

		// TODO: Update the ConfigMap to add latest scope, it should be the updatedNsList as requested-from-namespace, and MasterNs as map-to-common-service-namespace
		klog.Infof("%v, %s", updatedNsList, r.Bootstrap.CSData.MasterNs)
	}

	// If the intersection is empty, then do nothing
	if len(excludedNsList) == 0 {
		klog.Infof("Existing Common Service tenant scope contains following namespaces: %v, there is no isolation process required", nsScope)
		return ctrl.Result{}, nil
	}

	// TODO: Silence CS 3.x CR reconciliation by enabling maintenance mode

	// TODO: Re-construct CP2 tenant scope
	// 1. Refresh CommonService Operator memory/cache to re-construct the tenant scope.
	// - Update Bootstrap and CSdata structure.

	// 2. Patch ODLM subscription

	// 3. Re-construct the NamespaceScope CRs in ibm-common-services namespace.

	// TODO: Migrate old services by following isolate documentation

	// 1. Migrate Licensing data
	// 2. Backup Licensing CR
	// 3. Delete Licensing Operator
	// 4. Delete Licensing CR
	// 5. Restore Licensing CR

	// 6. Migrate Cert-Manager

	// 7. Delete Crossplane, webhook, and secretshare deployment

	// TODO: Release the maintenance mode on CS CR reconciliation

	return ctrl.Result{}, nil

}
