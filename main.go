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

package main

import (
	"context"
	"flag"
	"os"
	"time"

	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	cache "github.com/IBM/controller-filtered-cache/filteredcache"
	nssv1 "github.com/IBM/ibm-namespace-scope-operator/api/v1"
	ssv1 "github.com/IBM/ibm-secretshare-operator/api/v1"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"

	operatorv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	"github.com/IBM/ibm-common-service-operator/controllers/goroutines"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(odlm.AddToScheme(scheme))
	utilruntime.Must(nssv1.AddToScheme(scheme))
	utilruntime.Must(ssv1.AddToScheme(scheme))
	utilruntime.Must(operatorv3.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme

	utilruntime.Must(olmv1alpha1.AddToScheme(scheme))
	utilruntime.Must(olmv1.AddToScheme(scheme))
	utilruntime.Must(operatorsv1.AddToScheme(scheme))
}

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	gvkLabelMap := map[schema.GroupVersionKind]cache.Selector{
		corev1.SchemeGroupVersion.WithKind("ConfigMap"): {
			LabelSelector: constant.CsManagedLabel,
		},
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "be598e12.ibm.com",
		NewCache:               cache.NewFilteredCacheBuilder(gvkLabelMap),
	})
	if err != nil {
		klog.Errorf("Unable to start manager: %v", err)
		os.Exit(1)
	}

	// Validate common-service-maps
	cm, err := util.GetCmOfMapCs(mgr.GetAPIReader())
	if err == nil {
		if err := util.ValidateCsMaps(cm); err != nil {
			klog.Errorf("Unsupported common-service-maps: %v", err)
			os.Exit(1)
		}
		if !(cm.Labels != nil && cm.Labels[constant.CsManagedLabel] == "true") {
			util.EnsureLabelsForConfigMap(cm, map[string]string{
				constant.CsManagedLabel: "true",
			})
			if err := mgr.GetClient().Update(context.TODO(), cm); err != nil {
				klog.Errorf("Failed to update labels for common-service-maps: %v", err)
				os.Exit(1)
			}
		}
	} else if !errors.IsNotFound(err) {
		klog.Errorf("Failed to get common-service-maps: %v", err)
		os.Exit(1)
	}

	for {
		typeCorrect, err := bootstrap.CheckClusterType(mgr, util.GetMasterNs(mgr.GetAPIReader()))
		if err != nil {
			klog.Errorf("Failed to verify cluster type  %v", err)
			continue
		}

		if !typeCorrect {
			klog.Error("Cluster type specificed in the ibm-cpp-config isn't correct")
			time.Sleep(2 * time.Minute)
		} else {
			break
		}
	}

	// New bootstrap Object
	bs, err := bootstrap.NewBootstrap(mgr)
	if err != nil {
		klog.Errorf("Bootstrap failed: %v", err)
		os.Exit(1)
	}
	operatorNs, err := util.GetOperatorNamespace()
	if err != nil {
		klog.Errorf("Getting operator namespace failed: %v", err)
		os.Exit(1)
	}

	if err := bs.CheckOperatorCatalog(operatorNs); err != nil {
		klog.Errorf("Checking operator catalog failed: %v", err)
		os.Exit(1)
	}

	// Create master namespace
	if operatorNs != bs.CSData.MasterNs {
		klog.Infof("Creating IBM Common Services master namespace: %s", bs.CSData.MasterNs)
		if err := bs.CreateNamespace(bs.CSData.MasterNs); err != nil {
			klog.Errorf("Failed to create master namespace: %v", err)
			os.Exit(1)
		}

		klog.Info("Creating OperatorGroup for IBM Common Services")
		if err := bs.CreateOperatorGroup(); err != nil {
			klog.Errorf("Failed to create OperatorGroup for IBM Common Services: %v", err)
			os.Exit(1)
		}

		klog.Info("Creating ConfigMap for operators")
		if err := bs.CreateNsScopeConfigmap(); err != nil {
			klog.Errorf("Failed to create Namespace Scope ConfigMap: %v", err)
			os.Exit(1)
		}
	}

	if operatorNs == bs.CSData.MasterNs || operatorNs == constant.ClusterOperatorNamespace {
		klog.Infof("Creating CommonService CR in the namespace %s", bs.CSData.MasterNs)
		if err = bs.CreateCsCR(); err != nil {
			klog.Errorf("Failed to create CommonService CR: %v", err)
			os.Exit(1)
		}

		// Generate Issuer and Certificate CR, integrated into Controller logic
		// go goroutines.DeployCertManagerCR(bs)
		// Check IAM pods status
		go goroutines.CheckIamStatus(bs)
		// Sync up NSS CR
		go goroutines.SyncUpNSSCR(bs)
		// Update CS CR Status
		go goroutines.UpdateCsCrStatus(bs)
		// Create or Update CPP configuration
		go goroutines.CreateUpdateConfig(bs)
		// Clean up decprecated services
		go goroutines.CleanUpDeprecatedServices(bs)

		if err = (&controllers.CommonServiceReconciler{
			Bootstrap: bs,
			Scheme:    mgr.GetScheme(),
			Recorder:  mgr.GetEventRecorderFor("commonservice-controller"),
		}).SetupWithManager(mgr); err != nil {
			klog.Errorf("Unable to create controller CommonService: %v", err)
			os.Exit(1)
		}
		// +kubebuilder:scaffold:builder
	} else {
		klog.Infof("Creating common service operator subscription in namespace %s", bs.CSData.MasterNs)
		if err = bs.CheckCsSubscription(); err != nil {
			klog.Errorf("Failed to check common service operator subscription: %v", err)
			os.Exit(1)
		}
		if err = bs.CreateCsSubscription(); err != nil {
			klog.Errorf("Failed to create common service operator subscription: %v", err)
			os.Exit(1)
		}
		if err = bs.UpdateCsOpApproval(); err != nil {
			klog.Errorf("Failed to update common service operator subscription: %v", err)
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.Errorf("unable to set up health check: %v", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.Errorf("unable to set up ready check: %v", err)
		os.Exit(1)
	}

	klog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Errorf("Problem running manager: %v", err)
		os.Exit(1)
	}
}
