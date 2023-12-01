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
	"fmt"

	// certmanagerv1alpha1 "github.com/ibm/ibm-cert-manager-operator/apis/certmanager/v1alpha1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
)

// CommonServiceReconciler reconciles a CommonService object
type CommonServiceReconciler struct {
	*bootstrap.Bootstrap
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

const (
	CRInitializing string = "Initializing"
	CRUpdating     string = "Updating"
	CRPending      string = "Pending"
	CRSucceeded    string = "Succeeded"
	CRFailed       string = "Failed"
)

var ctx = context.Background()

//+kubebuilder:rbac:groups="",resources=namespaces,verbs=create;get;list;watch;update
//+kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions;operatorgroups,verbs=create;get;list;watch;update
//+kubebuilder:rbac:groups=operators.coreos.com,resources=catalogsources,verbs=get
//+kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions;clusterserviceversions,verbs=get;list;update;patch;delete
//+kubebuilder:rbac:groups=apiextensions.ibm.crossplane.io,resources=compositeresourcedefinitions;compositions,verbs=get;list;watch;delete
//+kubebuilder:rbac:groups=operator.ibm.com,resources=commonservices;commonservices/status;commonservices/finalizers,verbs=get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=create;get;update
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;roles;clusterrolebindings;rolebindings,verbs=create;get;list;watch;update;delete;escalate;bind
//+kubebuilder:rbac:groups="",resources=configmaps,resourceNames=common-service-maps;ibm-common-services-status;odlm-scope;namespace-scope,verbs=update;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=create;get;list;watch
//+kubebuilder:rbac:groups="",resources=serviceaccounts;events,verbs=create;get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=create;get
//+kubebuilder:rbac:groups=apps,resources=deployments,resourceNames=ibm-common-service-webhook;secretshare,verbs=update
//+kubebuilder:rbac:groups=pkg.ibm.crossplane.io,resources=locks;configurations,verbs=create;get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=kubernetes.crossplane.io;ibmcloud.crossplane.io,resources=providerconfigs,verbs=create;get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=ibmcpcs.ibm.com,resources=secretshares;secretshares/status,verbs=create;get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=operator.ibm.com,resources=podpresets;podpresets/status,verbs=create;get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=operator.ibm.com,resources=namespacescopes,verbs=create;get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=create;get;list;watch
//+kubebuilder:rbac:groups=operator.ibm.com,resources=meteringreportservers,verbs=get;delete
//+kubebuilder:rbac:groups=config.openshift.io,resources=infrastructures,verbs=get

//+kubebuilder:rbac:groups=operator.ibm.com,namespace="placeholder",resources=commonservices,verbs=create
//+kubebuilder:rbac:groups=operator.ibm.com,namespace="placeholder",resources=operandregistries;operandconfigs,verbs=create;get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=operator.ibm.com,namespace="placeholder",resources=operandrequests;operandbindinfos;cataloguis;helmapis;helmrepos,verbs=delete
//+kubebuilder:rbac:groups=elasticstack.ibm.com,namespace="placeholder",resources=elasticstacks,verbs=delete
//+kubebuilder:rbac:groups=monitoring.operator.ibm.com,namespace="placeholder",resources=exporters;prometheusexts,verbs=delete
//+kubebuilder:rbac:groups=cert-manager.io,namespace="placeholder",resources=certificates;issuers,verbs=create;get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=batch,namespace="placeholder",resources=jobs,verbs=create;get;list;watch

func (r *CommonServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	klog.Infof("Reconciling CommonService: %s", req.NamespacedName)

	// Validate common-service-maps and filter the namespace of CommonService CR
	inScope := true
	cm, err := util.GetCmOfMapCs(r.Client)
	if err == nil {
		if err := util.ValidateCsMaps(cm); err != nil {
			klog.Errorf("Unsupported common-service-maps: %v", err)
			return reconcile.Result{RequeueAfter: constant.DefaultRequeueDuration}, err
		}
		csScope, err := util.GetCsScope(cm, r.Bootstrap.CSData.MasterNs)
		if err != nil {
			return ctrl.Result{}, err
		}
		if inScope = r.checkScope(csScope, req.NamespacedName.Namespace); !inScope {
			klog.Infof("CommonService CR %v is not in the scope, only reconciles its configuration of cluster scope resource", req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
	} else if !errors.IsNotFound(err) {
		klog.Errorf("Failed to get common-service-maps: %v", err)
		return ctrl.Result{}, err
	}

	// Fetch the CommonService instance
	instance := &apiv3.CommonService{}

	if err := r.Bootstrap.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			if err := r.handleDelete(ctx); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.Bootstrap.DeleteCrossplaneAndProviderSubscription(r.Bootstrap.CSData.ControlNs); err != nil {
				return ctrl.Result{}, err
			}
			// Generate Issuer and Certificate CR
			if err := r.Bootstrap.DeployCertManagerCR(); err != nil {
				return ctrl.Result{}, err
			}
			klog.Infof("Finished reconciling to delete CommonService: %s/%s", req.NamespacedName.Namespace, req.NamespacedName.Name)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If the CommonService CR is not paused, continue to reconcile
	if !r.reconcilePauseRequest(instance) {
		if r.checkNamespace(req.NamespacedName.String()) {
			return r.ReconcileMasterCR(ctx, instance, inScope)
		}
		return r.ReconcileGeneralCR(ctx, instance, inScope)
	}
	// If the CommonService CR is paused, update the status to pending
	if err := r.updatePhase(ctx, instance, CRPending); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}
	klog.Infof("%s/%s is in pending status due to pause request", instance.Namespace, instance.Name)
	return ctrl.Result{}, nil
}

func (r *CommonServiceReconciler) ReconcileMasterCR(ctx context.Context, instance *apiv3.CommonService, inScope bool) (ctrl.Result, error) {

	if instance.Status.Phase == "" {
		if err := r.updatePhase(ctx, instance, CRInitializing); err != nil {
			klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
			return ctrl.Result{}, err
		}
	} else {
		if err := r.updatePhase(ctx, instance, CRUpdating); err != nil {
			klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
			return ctrl.Result{}, err
		}
	}

	// Generate Issuer and Certificate CR
	if err := r.Bootstrap.DeployCertManagerCR(); err != nil {
		klog.Errorf("Failed to deploy cert manager CRs: %v", err)
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	// Init common service bootstrap resource
	// Including namespace-scope configmap, nss operator, nss CR
	// Webhook Operator and Secretshare
	// Delete ODLM from openshift-operators and deploy it in the masterNamespaces
	// Deploy OperandConfig and OperandRegistry
	if err := r.Bootstrap.InitResources(instance); err != nil {
		klog.Errorf("Failed to initialize resources: %v", err)
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	if err := r.Bootstrap.Client.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, cs); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	if inScope {
		if err := r.Bootstrap.CrossplaneOperatorProviderOperator(instance); err != nil {
			klog.Errorf("Failed to install crossplane or provider operator: %v", err)
			if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
				klog.Error(err)
			}
			klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
			return ctrl.Result{}, err
		}
	}

	newConfigs, serviceControllerMapping, err := r.getNewConfigs(cs, inScope)
	if err != nil {
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isEqual, err := r.updateOperandConfig(ctx, newConfigs, serviceControllerMapping)
	if err != nil {
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	// Create Event if there is no update in OperandConfig after applying current CR
	if isEqual {
		r.Recorder.Event(instance, corev1.EventTypeNormal, "Noeffect", fmt.Sprintf("No update, resource sizings in the OperandConfig %s/%s are larger than the profile from CommonService CR %s/%s", r.Bootstrap.CSData.MasterNs, "common-service", instance.Namespace, instance.Name))
	}

	if err := r.updatePhase(ctx, instance, CRSucceeded); err != nil {
		klog.Error(err)
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s/%s", instance.Namespace, instance.Name)
	return ctrl.Result{}, nil
}

// ReconcileGeneralCR is for setting the OperandConfig
func (r *CommonServiceReconciler) ReconcileGeneralCR(ctx context.Context, instance *apiv3.CommonService, inScope bool) (ctrl.Result, error) {

	if instance.Status.Phase == "" {
		if err := r.updatePhase(ctx, instance, CRInitializing); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
	} else {
		if err := r.updatePhase(ctx, instance, CRUpdating); err != nil {
			klog.Error(err)
			return ctrl.Result{}, err
		}
	}

	opcon := util.NewUnstructured("operator.ibm.com", "OperandConfig", "v1alpha1")
	opconKey := types.NamespacedName{
		Name:      "common-service",
		Namespace: r.Bootstrap.CSData.MasterNs,
	}
	if err := r.Reader.Get(ctx, opconKey, opcon); err != nil {
		klog.Errorf("failed to get OperandConfig %s: %v", opconKey.String(), err)
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	cs := util.NewUnstructured("operator.ibm.com", "CommonService", "v3")
	if err := r.Bootstrap.Client.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, cs); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	if inScope {
		if err := r.Bootstrap.CrossplaneOperatorProviderOperator(instance); err != nil {
			klog.Errorf("Failed to install crossplane or provider operator: %v", err)
			if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
				klog.Error(err)
			}
			klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
			return ctrl.Result{}, err
		}

		// Generate Issuer and Certificate CR
		if err := r.Bootstrap.DeployCertManagerCR(); err != nil {
			klog.Errorf("Failed to deploy cert manager CRs: %v", err)
			if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
				klog.Error(err)
			}
			klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
			return ctrl.Result{}, err
		}
	}

	newConfigs, serviceControllerMapping, err := r.getNewConfigs(cs, inScope)
	if err != nil {
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	isEqual, err := r.updateOperandConfig(ctx, newConfigs, serviceControllerMapping)
	if err != nil {
		if err := r.updatePhase(ctx, instance, CRFailed); err != nil {
			klog.Error(err)
		}
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	// Create Event if there is no update in OperandConfig after applying current CR
	if isEqual {
		r.Recorder.Event(instance, corev1.EventTypeNormal, "Noeffect", fmt.Sprintf("No update, resource sizings in the OperandConfig %s/%s are larger than the profile from CommonService CR %s/%s", r.Bootstrap.CSData.MasterNs, "common-service", instance.Namespace, instance.Name))
	}

	if err := r.updatePhase(ctx, instance, CRSucceeded); err != nil {
		klog.Errorf("Fail to reconcile %s/%s: %v", instance.Namespace, instance.Name, err)
		return ctrl.Result{}, err
	}

	klog.Infof("Finished reconciling CommonService: %s/%s", instance.Namespace, instance.Name)
	return ctrl.Result{}, nil
}

func (r *CommonServiceReconciler) mappingToCsRequest() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		CsInstance := []reconcile.Request{}
		cmName := object.GetName()
		cmNs := object.GetNamespace()
		if cmName == constant.CsMapConfigMap && cmNs == "kube-public" {
			CsInstance = append(CsInstance, reconcile.Request{NamespacedName: types.NamespacedName{Name: "common-service", Namespace: r.Bootstrap.CSData.MasterNs}})
		}
		return CsInstance
	}
}

// func (r *CommonServiceReconciler) certsToCsRequest() handler.MapFunc {
// 	return func(object client.Object) []reconcile.Request {
// 		CsInstance := []reconcile.Request{}
// 		certName := object.GetName()
// 		certNs := object.GetNamespace()
// 		if certName == constant.CSCACertificate && certNs == r.Bootstrap.CSData.MasterNs {
// 			CsInstance = append(CsInstance, reconcile.Request{NamespacedName: types.NamespacedName{Name: "common-service", Namespace: r.Bootstrap.CSData.MasterNs}})
// 		}
// 		return CsInstance
// 	}
// }

func (r *CommonServiceReconciler) certSubToCsRequest() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		CsInstance := []reconcile.Request{}
		certSubName := object.GetName()
		certSubNs := object.GetNamespace()
		if certSubName == constant.CertManagerSub && certSubNs == r.Bootstrap.CSData.ControlNs {
			CsInstance = append(CsInstance, reconcile.Request{NamespacedName: types.NamespacedName{Name: "common-service", Namespace: r.Bootstrap.CSData.MasterNs}})
		}
		return CsInstance
	}
}

func (r *CommonServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// AnnotationChangedPredicate is intended to be used in conjunction with the GenerationChangedPredicate
		For(&apiv3.CommonService{}, builder.WithPredicates(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
				predicate.LabelChangedPredicate{}))).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.mappingToCsRequest()),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool {
					return true
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					return !e.DeleteStateUnknown
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					return true
				},
			})).
		// Watches(
		// 	&source.Kind{Type: &certmanagerv1alpha1.Certificate{}},
		// 	handler.EnqueueRequestsFromMapFunc(r.certsToCsRequest()),
		// 	builder.WithPredicates(predicate.Funcs{
		// 		DeleteFunc: func(e event.DeleteEvent) bool {
		// 			return !e.DeleteStateUnknown
		// 		},
		// 		UpdateFunc: func(e event.UpdateEvent) bool {
		// 			return true
		// 		},
		// 	})).
		Watches(
			&source.Kind{Type: &olmv1alpha1.Subscription{}},
			handler.EnqueueRequestsFromMapFunc(r.certSubToCsRequest()),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool {
					return true
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					return !e.DeleteStateUnknown
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					return true
				},
			})).Complete(r)
}
