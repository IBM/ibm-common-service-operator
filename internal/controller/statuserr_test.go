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
	"io"
	"net/http"
	"strings"
	"testing"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/bootstrap"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

type statusTestTransport func(*http.Request) (*http.Response, error)

func (f statusTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newStatusTestReconciler(t *testing.T, transport http.RoundTripper, hooks interceptor.Funcs) (*CommonServiceReconciler, *apiv3.CommonService) {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, apiv3.AddToScheme(s))
	require.NoError(t, odlm.AddToScheme(s))
	instance := &apiv3.CommonService{ObjectMeta: metav1.ObjectMeta{
		Name: constant.MasterCR, Namespace: "test-ns", UID: "cs-uid",
	}}
	fc := fake.NewClientBuilder().WithScheme(s).WithStatusSubresource(instance).
		WithObjects(instance, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: instance.Namespace}}).
		WithInterceptorFuncs(hooks).Build()
	require.NoError(t, fc.Get(context.Background(), client.ObjectKeyFromObject(instance), instance))
	return &CommonServiceReconciler{Bootstrap: &bootstrap.Bootstrap{
		Client: fc, Reader: fc,
		Config: &rest.Config{Host: "https://test.invalid", Transport: transport},
		CSData: apiv3.CSData{OperatorNs: instance.Namespace, CPFSNs: instance.Namespace, ServicesNs: instance.Namespace},
	}}, instance
}

// Exercise the production certificate failure branch, including phase-update failure.
func TestDeployCertManagerCRPreservesDeploymentError(t *testing.T) {
	deploymentErr := errors.New("certificate discovery unavailable")
	phaseErr := errors.New("status update unavailable")
	for _, failPhaseUpdate := range []bool{false, true} {
		name := "phase update succeeds"
		if failPhaseUpdate {
			name = "phase update fails"
		}
		t.Run(name, func(t *testing.T) {
			discoveryCalled, phaseUpdateCalled := false, false
			transport := statusTestTransport(func(*http.Request) (*http.Response, error) {
				discoveryCalled = true
				return nil, deploymentErr
			})
			hooks := interceptor.Funcs{SubResourceUpdate: func(ctx context.Context, c client.Client, subresource string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				phaseUpdateCalled = true
				require.Equal(t, "status", subresource)
				require.Equal(t, apiv3.CRFailed, obj.(*apiv3.CommonService).Status.Phase)
				if failPhaseUpdate {
					return phaseErr
				}
				return c.SubResource(subresource).Update(ctx, obj, opts...)
			}}
			r, instance := newStatusTestReconciler(t, transport, hooks)
			err := r.deployCertManagerCR(context.Background(), instance)
			require.ErrorIs(t, err, deploymentErr)
			assert.True(t, discoveryCalled)
			assert.True(t, phaseUpdateCalled)
			if !failPhaseUpdate {
				persisted := &apiv3.CommonService{}
				require.NoError(t, r.Client.Get(context.Background(), client.ObjectKeyFromObject(instance), persisted))
				assert.Equal(t, apiv3.CRFailed, persisted.Status.Phase)
			}
		})
	}
}

// Verify the real master reconcile's deferred condition handling separately.
func TestReconcileMasterCRRecordsErrorCondition(t *testing.T) {
	t.Setenv(constant.OperatorNamespaceEnvVar, "test-ns")
	transport := statusTestTransport(func(req *http.Request) (*http.Response, error) {
		body := `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`
		if req.URL.Path == "/api" {
			body = `{"kind":"APIVersions","apiVersion":"v1","versions":["v1"]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})
	r, instance := newStatusTestReconciler(t, transport, interceptor.Funcs{})
	// No ibm-cpp-config on a non-OCP cluster drives the cluster-type error branch.
	_, err := r.ReconcileMasterCR(context.Background(), instance)
	require.EqualError(t, err, "cluster type specified in the ibm-cpp-config isn't correct")
	persisted := &apiv3.CommonService{}
	require.NoError(t, r.Client.Get(context.Background(), client.ObjectKeyFromObject(instance), persisted))
	assert.Equal(t, apiv3.CRFailed, persisted.Status.Phase)
	var hasError, hasReady bool
	for _, c := range persisted.Status.Conditions {
		if c.Type == apiv3.ConditionTypeError && c.Status == corev1.ConditionTrue {
			hasError = true
			assert.Equal(t, err.Error(), c.Message)
		}
		if c.Type == apiv3.ConditionTypeReady && c.Status == corev1.ConditionTrue {
			hasReady = true
		}
	}
	assert.True(t, hasError)
	assert.False(t, hasReady)
}
