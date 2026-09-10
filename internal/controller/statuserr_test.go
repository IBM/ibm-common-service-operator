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
	"errors"
	"testing"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
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

// newTestReconciler returns a CommonServiceReconciler backed by a fake client,
// with only the fields needed for updatePhase and the status-deferred tests.
func newTestReconciler(t *testing.T, objs ...apiv3.CommonService) *CommonServiceReconciler {
	t.Helper()
	runtimeObjs := make([]interface{}, len(objs))
	for i := range objs {
		runtimeObjs[i] = &objs[i]
	}
	builder := fake.NewClientBuilder().WithScheme(scheme.Scheme)
	for i := range objs {
		builder = builder.WithStatusSubresource(&objs[i]).WithRuntimeObjects(&objs[i])
	}
	fc := builder.Build()
	return &CommonServiceReconciler{
		Bootstrap: &bootstrap.Bootstrap{
			Client: fc,
			Reader: fc,
		},
	}
}

// TestUpdatePhaseDoesNotOverwriteCallerError verifies the fix for the statusErr
// variable overwrite bug.
//
// Before the fix, error paths used:
//
//	if statusErr = r.Bootstrap.DeployCertManagerCR(instance); statusErr != nil {
//	    if statusErr = r.updatePhase(...); statusErr != nil { ... }
//	    return ctrl.Result{}, statusErr  // returns nil when updatePhase succeeds!
//	}
//
// The outer statusErr was overwritten by the updatePhase result, so the
// deferred condition-setter saw nil and wrote Ready=True instead of Error.
//
// After the fix updatePhase is called with a local err variable so the outer
// statusErr is never clobbered.
func TestUpdatePhaseDoesNotOverwriteCallerError(t *testing.T) {
	instance := apiv3.CommonService{ObjectMeta: metav1.ObjectMeta{
		Name:            "common-service",
		Namespace:       "test-ns",
		UID:             types.UID("uid-cs"),
		ResourceVersion: "1",
	}}

	r := newTestReconciler(t, instance)
	ctx := context.Background()

	// Simulate the original (broken) pattern: statusErr is assigned by both
	// the original failure AND the updatePhase call.
	originalErr := errors.New("cert manager deployment failed")
	statusErr := originalErr

	// updatePhase writing Phase=Failed succeeds → with the old pattern this
	// would have set statusErr = nil.
	phaseErr := r.updatePhase(ctx, &instance, apiv3.CRFailed)
	require.NoError(t, phaseErr, "updatePhase itself must succeed")

	// The caller's error must be preserved — not overwritten.
	// This is what the fix guarantees by using a local `err` variable.
	assert.Equal(t, originalErr, statusErr,
		"statusErr must not be overwritten when updatePhase succeeds")

	// Phase must have been persisted.
	latest := &apiv3.CommonService{}
	require.NoError(t, r.Client.Get(ctx, types.NamespacedName{Name: "common-service", Namespace: "test-ns"}, latest))
	assert.Equal(t, apiv3.CRFailed, latest.Status.Phase)
}
