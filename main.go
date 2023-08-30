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
	"flag"
	"os"
	"strings"

	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/IBM/controller-filtered-cache/filteredcache"
	nssv1 "github.com/IBM/ibm-namespace-scope-operator/api/v1"
	ssv1 "github.com/IBM/ibm-secretshare-operator/api/v1"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"

	certmanagerv1 "github.com/ibm/ibm-cert-manager-operator/apis/cert-manager/v1"
	cmconstants "github.com/ibm/ibm-cert-manager-operator/controllers/resources"

	operatorv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	certmanagerv1controllers "github.com/IBM/ibm-common-service-operator/controllers/cert-manager"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
	"github.com/IBM/ibm-common-service-operator/controllers/goroutines"
	commonservicewebhook "github.com/IBM/ibm-common-service-operator/controllers/webhooks/commonservice"
	operandrequestwebhook "github.com/IBM/ibm-common-service-operator/controllers/webhooks/operandrequest"
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
	utilruntime.Must(certmanagerv1.AddToScheme(scheme))
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
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	watchNamespace := util.GetWatchNamespace()
	gvkLabelMap := map[schema.GroupVersionKind]filteredcache.Selector{
		corev1.SchemeGroupVersion.WithKind("ConfigMap"): {
			LabelSelector: constant.CsManagedLabel,
		},
		corev1.SchemeGroupVersion.WithKind("Secret"): {
			LabelSelector: cmconstants.SecretWatchLabel,
		},
	}
	clusterGVKList := []schema.GroupVersionKind{
		{Group: "admissionregistration.k8s.io", Kind: "MutatingWebhookConfiguration", Version: "v1"},
		{Group: "admissionregistration.k8s.io", Kind: "ValidatingWebhookConfiguration", Version: "v1"},
	}

	var NewCache cache.NewCacheFunc
	watchNamespaceList := strings.Split(watchNamespace, ",")
	NewCache = util.NewCSCache(clusterGVKList, gvkLabelMap, watchNamespaceList)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "be598e12.ibm.com",
		NewCache:               NewCache,
	})
	if err != nil {
		klog.Errorf("Unable to start manager: %v", err)
		os.Exit(1)
	}

	operatorNs, err := util.GetOperatorNamespace()
	klog.Infof("Identifying Common Service Operator Role in the namespace %s", operatorNs)
	if err != nil {
		klog.Errorf("Failed to get operatorNs: %v", err)
		os.Exit(1)
	}

	cpfsNs, err := bootstrap.IdentifyCPFSNs(mgr.GetAPIReader(), operatorNs)
	if err != nil {
		klog.Errorf("Failed to get Common Service deployed namespace: %v", err)
		os.Exit(1)
	}
	// If Common Service Operator Namespace is not in the same as .spec.operatorNamespace(cpfsNs) in default CS CR,
	// this Common Service Operator is not in the operatorNamespace(cpfsNs) under this tenant, and goes dormant.
	if operatorNs == cpfsNs {

		// New bootstrap Object
		bs, err := bootstrap.NewBootstrap(mgr)
		if err != nil {
			klog.Errorf("Bootstrap failed: %v", err)
			os.Exit(1)
		}

		if err := bs.CleanupWebhookResources(); err != nil {
			klog.Errorf("Cleanup Webhook Resources failed: %v", err)
			os.Exit(1)
		}
		// Create or Update CPP configuration
		go goroutines.CreateUpdateConfig(bs)
		// Update CS CR Status
		go goroutines.UpdateCsCrStatus(bs)
		// Create CS CR
		go goroutines.WaitToCreateCsCR(bs)

		if err = (&controllers.CommonServiceReconciler{
			Bootstrap: bs,
			Scheme:    mgr.GetScheme(),
			Recorder:  mgr.GetEventRecorderFor("commonservice-controller"),
		}).SetupWithManager(mgr); err != nil {
			klog.Errorf("Unable to create controller CommonService: %v", err)
			os.Exit(1)
		}
		if err = (&certmanagerv1controllers.CertificateRefreshReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "CertificateRefresh")
			os.Exit(1)
		}
		if err = (&certmanagerv1controllers.PodRefreshReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "PodRefresh")
			os.Exit(1)
		}
		if err = (&certmanagerv1controllers.V1AddLabelReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "V1AddLabel")
			os.Exit(1)
		}
	} else {
		klog.Infof("Common Service Operator goes dormant in the namespace %s", operatorNs)
		klog.Infof("Common Service Operator in the namespace %s takes charge of resource management", cpfsNs)
	}

	// Start up the webhook server if it is ocp
	if err = (&commonservicewebhook.Defaulter{
		Client:    mgr.GetClient(),
		Reader:    mgr.GetAPIReader(),
		IsDormant: operatorNs != cpfsNs,
	}).SetupWebhookWithManager(mgr); err != nil {
		klog.Errorf("Unable to create CommonService webhook: %v", err)
		os.Exit(1)
	}

	if err = (&operandrequestwebhook.Defaulter{
		Client:    mgr.GetClient(),
		Reader:    mgr.GetAPIReader(),
		IsDormant: operatorNs != cpfsNs,
	}).SetupWebhookWithManager(mgr); err != nil {
		klog.Errorf("Unable to create OperandRequest webhook: %v", err)
		os.Exit(1)
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
