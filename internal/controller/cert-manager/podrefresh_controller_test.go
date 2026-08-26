// Copyright 2026 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package certmanager

import (
	"context"
	"errors"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type staticDaemonSetPermissionChecker struct {
	result daemonSetAccessResult
	err    error
}

func (c staticDaemonSetPermissionChecker) Check(context.Context, string) (daemonSetAccessResult, error) {
	return c.result, c.err
}

type daemonSetListForbiddenClient struct {
	client.Client
}

func (c daemonSetListForbiddenClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if _, ok := list.(*appsv1.DaemonSetList); ok {
		return apierrors.NewForbidden(schema.GroupResource{Group: "apps", Resource: "daemonsets"}, "", errors.New("forbidden"))
	}
	return c.Client.List(ctx, list, opts...)
}

type daemonSetUpdateForbiddenClient struct {
	client.Client
}

func (c daemonSetUpdateForbiddenClient) Update(ctx context.Context, object client.Object, opts ...client.UpdateOption) error {
	if daemonSet, ok := object.(*appsv1.DaemonSet); ok {
		return apierrors.NewForbidden(schema.GroupResource{Group: "apps", Resource: "daemonsets"}, daemonSet.Name, errors.New("forbidden"))
	}
	return c.Client.Update(ctx, object, opts...)
}

type recordingSSARClient struct {
	client.Client
	deniedVerb          string
	evaluationErrorVerb string
	createErr           error
	verbs               []string
}

func (c *recordingSSARClient) Create(ctx context.Context, object client.Object, opts ...client.CreateOption) error {
	review, ok := object.(*authorizationv1.SelfSubjectAccessReview)
	if !ok {
		return c.Client.Create(ctx, object, opts...)
	}
	if c.createErr != nil {
		return c.createErr
	}

	verb := review.Spec.ResourceAttributes.Verb
	c.verbs = append(c.verbs, verb)
	review.Status.Allowed = verb != c.deniedVerb
	if verb == c.evaluationErrorVerb {
		review.Status.EvaluationError = "authorization backend unavailable"
	}
	return nil
}

func TestRestartSkipsDaemonSetsWhenPermissionIsDenied(t *testing.T) {
	ctx := context.Background()
	daemonSet := daemonSetUsingSecret("test-ns", "tls-secret")
	deployment := deploymentUsingSecret("test-ns", "tls-secret")
	statefulSet := statefulSetUsingSecret("test-ns", "tls-secret")
	r := testPodRefreshReconciler(t, daemonSet, deployment, statefulSet)
	r.daemonSetPermissionChecker = staticDaemonSetPermissionChecker{
		result: daemonSetAccessResult{deniedVerb: "update"},
	}

	if err := r.restart(ctx, "tls-secret", "certificate", "test-ns", "2000-1-1.000000"); err != nil {
		t.Fatalf("restart returned an error: %v", err)
	}
	assertDaemonSetNotRestarted(t, ctx, r.Client, daemonSet)
	assertDeploymentRestarted(t, ctx, r.Client, deployment)
	assertStatefulSetRestarted(t, ctx, r.Client, statefulSet)
}

func TestRestartReturnsErrorWhenAccessReviewFails(t *testing.T) {
	ctx := context.Background()
	daemonSet := daemonSetUsingSecret("test-ns", "tls-secret")
	r := testPodRefreshReconciler(t, daemonSet)
	r.daemonSetPermissionChecker = staticDaemonSetPermissionChecker{
		err: errors.New("access review failed"),
	}

	if err := r.restart(ctx, "tls-secret", "certificate", "test-ns", "2000-1-1.000000"); err == nil {
		t.Fatal("expected access review failure to be returned for reconciliation retry")
	}
	assertDaemonSetNotRestarted(t, ctx, r.Client, daemonSet)
}

func TestRestartSkipsDaemonSetsWhenAccessReviewIsForbidden(t *testing.T) {
	ctx := context.Background()
	daemonSet := daemonSetUsingSecret("test-ns", "tls-secret")
	r := testPodRefreshReconciler(t, daemonSet)
	r.daemonSetPermissionChecker = staticDaemonSetPermissionChecker{
		err: apierrors.NewForbidden(
			schema.GroupResource{Group: "authorization.k8s.io", Resource: "selfsubjectaccessreviews"},
			"",
			errors.New("forbidden"),
		),
	}

	if err := r.restart(ctx, "tls-secret", "certificate", "test-ns", "2000-1-1.000000"); err != nil {
		t.Fatalf("restart returned an error: %v", err)
	}
	assertDaemonSetNotRestarted(t, ctx, r.Client, daemonSet)
}

func TestRestartContinuesWhenDaemonSetListBecomesForbidden(t *testing.T) {
	ctx := context.Background()
	daemonSet := daemonSetUsingSecret("test-ns", "tls-secret")
	r := testPodRefreshReconciler(t, daemonSet)
	r.Client = daemonSetListForbiddenClient{Client: r.Client}
	r.daemonSetPermissionChecker = staticDaemonSetPermissionChecker{
		result: daemonSetAccessResult{allowed: true},
	}

	if err := r.restart(ctx, "tls-secret", "certificate", "test-ns", "2000-1-1.000000"); err != nil {
		t.Fatalf("restart returned an error: %v", err)
	}
}

func TestRestartContinuesWhenDaemonSetUpdateBecomesForbidden(t *testing.T) {
	ctx := context.Background()
	daemonSet := daemonSetUsingSecret("test-ns", "tls-secret")
	r := testPodRefreshReconciler(t, daemonSet)
	r.Client = daemonSetUpdateForbiddenClient{Client: r.Client}
	r.daemonSetPermissionChecker = staticDaemonSetPermissionChecker{
		result: daemonSetAccessResult{allowed: true},
	}

	if err := r.restart(ctx, "tls-secret", "certificate", "test-ns", "2000-1-1.000000"); err != nil {
		t.Fatalf("restart returned an error: %v", err)
	}
}

func TestRestartUpdatesDaemonSetsWhenPermissionIsGranted(t *testing.T) {
	ctx := context.Background()
	daemonSet := daemonSetUsingSecret("test-ns", "tls-secret")
	r := testPodRefreshReconciler(t, daemonSet)
	r.daemonSetPermissionChecker = staticDaemonSetPermissionChecker{
		result: daemonSetAccessResult{allowed: true},
	}

	if err := r.restart(ctx, "tls-secret", "certificate", "test-ns", "2000-1-1.000000"); err != nil {
		t.Fatalf("restart returned an error: %v", err)
	}

	updated := &appsv1.DaemonSet{}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(daemonSet), updated); err != nil {
		t.Fatalf("get updated DaemonSet: %v", err)
	}
	if updated.Labels[restartLabel] == "" || updated.Spec.Template.Labels[restartLabel] == "" {
		t.Fatalf("expected restart labels to be set on DaemonSet and pod template")
	}
}

func TestSelfSubjectDaemonSetPermissionCheckerChecksRequiredVerbs(t *testing.T) {
	ctx := context.Background()
	baseClient := testClient(t)
	client := &recordingSSARClient{Client: baseClient}
	checker := selfSubjectDaemonSetPermissionChecker{client: client}

	result, err := checker.Check(ctx, "test-ns")
	if err != nil {
		t.Fatalf("check returned an error: %v", err)
	}
	if !result.allowed {
		t.Fatalf("expected all permissions to be allowed, got %+v", result)
	}
	if want := []string{"list", "watch", "update"}; !reflect.DeepEqual(client.verbs, want) {
		t.Fatalf("reviewed verbs = %v, want %v", client.verbs, want)
	}

	client = &recordingSSARClient{Client: baseClient, deniedVerb: "update"}
	checker = selfSubjectDaemonSetPermissionChecker{client: client}
	result, err = checker.Check(ctx, "test-ns")
	if err != nil {
		t.Fatalf("check returned an error: %v", err)
	}
	if result.allowed || result.deniedVerb != "update" {
		t.Fatalf("expected update to be denied, got %+v", result)
	}

	client = &recordingSSARClient{Client: baseClient, evaluationErrorVerb: "watch"}
	checker = selfSubjectDaemonSetPermissionChecker{client: client}
	if _, err = checker.Check(ctx, "test-ns"); err == nil {
		t.Fatal("expected SelfSubjectAccessReview evaluation error to be returned")
	}
}

func testPodRefreshReconciler(t *testing.T, objects ...client.Object) *PodRefreshReconciler {
	t.Helper()
	return &PodRefreshReconciler{Client: testClient(t, objects...)}
}

func testClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add apps scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	if err := authorizationv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add authorization scheme: %v", err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func daemonSetUsingSecret(namespace, secretName string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "uses-secret",
			Namespace: namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: podTemplateUsingSecret(secretName),
		},
	}
}

func deploymentUsingSecret(namespace, secretName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "uses-secret-deployment", Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: podTemplateUsingSecret(secretName),
		},
	}
}

func statefulSetUsingSecret(namespace, secretName string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "uses-secret-statefulset", Namespace: namespace},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: podTemplateUsingSecret(secretName),
		},
	}
}

func podTemplateUsingSecret(secretName string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "test",
			Env: []corev1.EnvVar{{ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
					Key:                  "tls.crt",
				},
			}}},
		}}},
	}
}

func assertDaemonSetNotRestarted(t *testing.T, ctx context.Context, c client.Client, daemonSet *appsv1.DaemonSet) {
	t.Helper()
	updated := &appsv1.DaemonSet{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(daemonSet), updated); err != nil {
		t.Fatalf("get DaemonSet: %v", err)
	}
	if updated.Labels[restartLabel] != "" || updated.Spec.Template.Labels[restartLabel] != "" {
		t.Fatalf("expected DaemonSet to remain unchanged, got labels %#v", updated.Labels)
	}
}

func assertDeploymentRestarted(t *testing.T, ctx context.Context, c client.Client, deployment *appsv1.Deployment) {
	t.Helper()
	updated := &appsv1.Deployment{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(deployment), updated); err != nil {
		t.Fatalf("get Deployment: %v", err)
	}
	if updated.Labels[restartLabel] == "" || updated.Spec.Template.Labels[restartLabel] == "" {
		t.Fatal("expected Deployment restart labels to be set")
	}
}

func assertStatefulSetRestarted(t *testing.T, ctx context.Context, c client.Client, statefulSet *appsv1.StatefulSet) {
	t.Helper()
	updated := &appsv1.StatefulSet{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(statefulSet), updated); err != nil {
		t.Fatalf("get StatefulSet: %v", err)
	}
	if updated.Labels[restartLabel] == "" || updated.Spec.Template.Labels[restartLabel] == "" {
		t.Fatal("expected StatefulSet restart labels to be set")
	}
}
