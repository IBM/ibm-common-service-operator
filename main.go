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

package main

import (
	"flag"
	"os"

	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"

	operatorv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	"github.com/IBM/ibm-common-service-operator/controllers/check"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(operatorv3.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme

	utilruntime.Must(olmv1alpha1.AddToScheme(scheme))
	utilruntime.Must(olmv1.AddToScheme(scheme))
}

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "be598e12.ibm.com",
	})
	if err != nil {
		klog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	klog.Info("check Helm based IBM Common Services installation")
	exist, err := check.OriginalCs(mgr)
	if err != nil {
		klog.Error(err)
		os.Exit(1)
	}
	if exist {
		klog.Error("The Helm based IBM Common Services must be uninstalled before performing operator based installation")
		os.Exit(1)
	}

	operatorNs, err := bootstrap.GetOperatorNamespace()
	if err != nil {
		klog.Error("Get operator namespace failed: ", err)
		os.Exit(1)
	}
	if operatorNs == "ibm-common-services" || operatorNs == "openshift-operators" {
		klog.Info("start installing ODLM operator and initialize IBM Common Services")
		if err = bootstrap.InitResources(mgr); err != nil {
			klog.Error("InitResources failed: ", err)
			os.Exit(1)
		}
		klog.Info("finish installing ODLM operator and initialize IBM Common Services")

		// Check IAM pods status
		go check.IamStatus(mgr)

		if err = (&controllers.CommonServiceReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("CommonService"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			klog.Error(err, "unable to create controller", "controller", "CommonService")
			os.Exit(1)
		}
		// +kubebuilder:scaffold:builder
	} else {
		if err = bootstrap.CreateNamespace(mgr); err != nil {
			klog.Error("Create ibm-common-services namespace failed: ", err)
			os.Exit(1)
		}
		klog.Info("start create common service operator")
		if err = bootstrap.CreateCsSubscription(mgr); err != nil {
			klog.Error("Create common service operator subscription failed: ", err)
			os.Exit(1)
		}
		klog.Info("Finish create common service operator")
	}

	klog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
