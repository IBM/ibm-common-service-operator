//
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
//

package certmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	certmanagerv1 "github.com/ibm/ibm-cert-manager-operator/apis/cert-manager/v1"

	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
)

func TestV1AddLabelReconcilerAddsCommonServiceOwnerToCSCASecret(t *testing.T) {
	const namespace = "test-common-service"

	testScheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(testScheme))
	assert.NoError(t, certmanagerv1.AddToScheme(testScheme))

	certificate := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constant.CSCACertificate,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: constant.APIVersion,
				Kind:       constant.KindCR,
				Name:       constant.MasterCR,
				UID:        types.UID("common-service-uid"),
			}},
		},
		Spec: certmanagerv1.CertificateSpec{SecretName: constant.CSCACertificateSecret},
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      constant.CSCACertificateSecret,
		Namespace: namespace,
	}}

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(certificate, secret).Build()
	reconciler := &V1AddLabelReconciler{Client: fakeClient, Reader: fakeClient, Scheme: testScheme}
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: certificate.Name, Namespace: namespace}}

	result, err := reconciler.Reconcile(context.Background(), request)
	assert.NoError(t, err)
	assert.Zero(t, result.RequeueAfter)

	actual := &corev1.Secret{}
	assert.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: secret.Name, Namespace: namespace}, actual))
	assert.Contains(t, actual.Labels, constant.SecretWatchLabel)
	if assert.Len(t, actual.OwnerReferences, 1) {
		owner := actual.OwnerReferences[0]
		assert.Equal(t, types.UID("common-service-uid"), owner.UID)
		assert.NotNil(t, owner.Controller)
		assert.True(t, *owner.Controller)
	}

	// Reconciliation is idempotent and does not duplicate the owner reference.
	_, err = reconciler.Reconcile(context.Background(), request)
	assert.NoError(t, err)
	assert.NoError(t, fakeClient.Get(context.Background(), types.NamespacedName{Name: secret.Name, Namespace: namespace}, actual))
	assert.Len(t, actual.OwnerReferences, 1)
}

func TestV1AddLabelReconcilerRequeuesUntilCertificateSecretExists(t *testing.T) {
	testScheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(testScheme))
	assert.NoError(t, certmanagerv1.AddToScheme(testScheme))

	certificate := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: constant.CSCACertificate, Namespace: "test-common-service"},
		Spec:       certmanagerv1.CertificateSpec{SecretName: constant.CSCACertificateSecret},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(certificate).Build()
	reconciler := &V1AddLabelReconciler{Client: fakeClient, Reader: fakeClient, Scheme: testScheme}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{
		Name: certificate.Name, Namespace: certificate.Namespace,
	}})
	assert.NoError(t, err)
	assert.Positive(t, result.RequeueAfter)
}

func TestCSCASecretWithoutManagedCertificateOwnerRemainsUserOwned(t *testing.T) {
	certificate := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: constant.CSCACertificate, Namespace: "test-common-service"},
		Spec:       certmanagerv1.CertificateSpec{SecretName: constant.CSCACertificateSecret},
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name: constant.CSCACertificateSecret, Namespace: certificate.Namespace,
	}}

	changed, err := ensureCSCACertificateSecretOwnerReference(certificate, secret)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Empty(t, secret.OwnerReferences)
}
