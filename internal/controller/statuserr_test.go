// Copyright 2026 IBM Corporation
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

package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/bootstrap"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
)

func init() {
	_ = olmv1alpha1.AddToScheme(scheme.Scheme)
	_ = odlm.AddToScheme(scheme.Scheme)
	_ = apiv3.AddToScheme(scheme.Scheme)
}

// minimalAPIServer returns a *httptest.Server that answers Kubernetes discovery
// requests with enough structure to make the client library happy but reports
// no API groups (so isOpenShiftCluster returns false).  All other requests
// return 404 so that CheckCRD and similar helpers fail gracefully.
func minimalAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// /api — core group version list
	mux.HandleFunc("/api", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metav1.APIVersions{TypeMeta: metav1.TypeMeta{
			Kind:       "APIVersions",
			APIVersion: "v1",
		}, Versions: []string{"v1"}})
	})
	// /apis — empty group list → isOpenShiftCluster returns false
	mux.HandleFunc("/apis", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metav1.APIGroupList{TypeMeta: metav1.TypeMeta{
			Kind:       "APIGroupList",
			APIVersion: "v1",
		}})
	})
	// Everything else: 404
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newTestReconciler returns a CommonServiceReconciler backed by a fake client
// and a minimal API server, with only the fields needed for ReconcileMasterCR
// tests that do not require full ODLM / cert-manager infrastructure.
func newTestReconciler(t *testing.T, srv *httptest.Server, objs ...apiv3.CommonService) *CommonServiceReconciler {
	t.Helper()
	builder := fake.NewClientBuilder().WithScheme(scheme.Scheme)
	for i := range objs {
		builder = builder.WithStatusSubresource(&objs[i]).WithRuntimeObjects(&objs[i])
	}
	fc := builder.Build()

	cfg := &rest.Config{Host: srv.URL}

	return &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			Client: fc,
			Reader: fc,
			Config: cfg,
		},
	}
}

// TestReconcileMasterCRErrorPathPreservesStatusErr is the regression test for
// the statusErr variable-overwrite bug.
//
// Before the fix, the DeployCertManagerCR and !typeCorrect error paths wrote:
//
//	if statusErr = r.Bootstrap.DeployCertManagerCR(instance); statusErr != nil {
//	    if statusErr = r.updatePhase(...); statusErr != nil { ... }
//	    return ctrl.Result{}, statusErr   // returned nil when updatePhase succeeded!
//	}
//
// The deferred condition-setter then saw statusErr == nil and wrote
// type:Ready=True instead of type:Error, leaving phase:Failed but
// conditions:Ready=True — the contradiction observed in the cluster.
//
// After the fix, updatePhase is called with a local `err` variable so the
// outer statusErr always retains the original error.
//
// This test exercises the real ReconcileMasterCR code path. It drives the
// reconcile into the !typeCorrect branch (which uses statusErr directly and
// was fixed in the same commit), verifies that:
//   - the reconcile returns the original error (not nil)
//   - the deferred function wrote an Error condition, not Ready
//   - the phase was set to Failed
func TestReconcileMasterCRErrorPathPreservesStatusErr(t *testing.T) {
	const (
		ns   = "test-ns"
		name = "common-service"
	)

	srv := minimalAPIServer(t)

	instance := apiv3.CommonService{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       ns,
			UID:             types.UID("uid-cs"),
			ResourceVersion: "1",
		},
		Spec: apiv3.CommonServiceSpec{
			License: apiv3.LicenseList{Accept: true},
			// operatorNamespace == servicesNamespace: WatchNamespaces is empty,
			// so the code checks for the servicesNamespace Namespace object.
			// Leave it unset; the Namespace will be present in the fake store.
		},
	}

	// The servicesNamespace Namespace must exist so the namespace-check does
	// not short-circuit before reaching CheckClusterType.
	svcNs := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}

	r := newTestReconciler(t, srv, instance)
	// Add the Namespace to the fake store.
	require.NoError(t, r.Client.Create(context.Background(), &svcNs))

	// Point OperatorNs / ServicesNs at the test namespace so the reconcile
	// does not try to look up unrelated objects.
	r.Bootstrap.CSData.OperatorNs = ns
	r.Bootstrap.CSData.ServicesNs = ns

	// Fetch a live copy so resourceVersion is consistent.
	live := &apiv3.CommonService{}
	require.NoError(t, r.Client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ns}, live))

	// Invoke the real reconcile method.
	result, err := r.ReconcileMasterCR(context.Background(), live)

	// The reconcile must return a non-nil error.  Before the fix, updatePhase
	// would overwrite statusErr with nil and the function returned (Result{}, nil).
	require.Error(t, err,
		"ReconcileMasterCR must return the original error, not nil, when it fails")
	assert.Equal(t, result, ctrl.Result{})

	// Read the persisted status.
	persisted := &apiv3.CommonService{}
	require.NoError(t, r.Client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ns}, persisted))

	// Phase must be Failed.
	assert.Equal(t, apiv3.CRFailed, persisted.Status.Phase,
		"phase must be Failed after a reconcile error")

	// The deferred condition-setter must have recorded an Error condition, NOT
	// a Ready condition.  Before the fix, statusErr was nil so Ready=True was
	// written instead.
	var hasError, hasReady bool
	for _, c := range persisted.Status.Conditions {
		switch c.Type {
		case apiv3.ConditionTypeError:
			hasError = c.Status == corev1.ConditionTrue
		case apiv3.ConditionTypeReady:
			hasReady = c.Status == corev1.ConditionTrue
		}
	}
	assert.True(t, hasError,
		"an Error condition must be set when ReconcileMasterCR returns an error")
	assert.False(t, hasReady,
		"Ready=True must NOT be set when ReconcileMasterCR returns an error (pre-fix bug)")
}
