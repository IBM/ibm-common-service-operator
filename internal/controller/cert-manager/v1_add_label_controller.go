/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package certmanager

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	certmanagerv1 "github.com/ibm/ibm-cert-manager-operator/apis/cert-manager/v1"

	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/common"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
)

// V1AddLabelReconciler reconciles a Certificate object
type V1AddLabelReconciler struct {
	client.Client
	client.Reader
	Scheme *runtime.Scheme
}

// //+kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch
// //+kubebuilder:rbac:groups=cert-manager.io,resources=certificates/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Certificate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *V1AddLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logd = log.FromContext(ctx)

	reqLogger := logd.WithValues("req.Namespace", req.Namespace, "req.Name", req.Name)
	reqLogger.Info("Reconciling CertificateRefresh")

	cert := &certmanagerv1.Certificate{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, cert)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logd.Error(err, "Error getting v1 Certificate")
		return ctrl.Result{}, err
	}

	// Get secret corresponding to the certificate
	secretInstance, err := r.getSecret(cert)
	if err != nil {
		if errors.IsNotFound(err) {
			// Certificate Secrets are generated asynchronously. Requeue so a
			// Certificate event that arrives before its Secret does not permanently
			// miss labels or ownership.
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		logd.Error(err, "Error getting Secret")
		return ctrl.Result{}, err
	}

	changed := false
	labelsMap := secretInstance.GetLabels()
	if labelsMap == nil {
		labelsMap = make(map[string]string)
	}
	if _, ok := labelsMap[constant.SecretWatchLabel]; !ok {
		labelsMap[constant.SecretWatchLabel] = ""
		secretInstance.SetLabels(labelsMap)
		changed = true
	}

	ownerReferenceChanged, err := ensureCSCACertificateSecretOwnerReference(cert, secretInstance)
	if err != nil {
		logd.Error(err, "Error adding CommonService owner reference to Secret")
		return ctrl.Result{}, err
	}
	changed = changed || ownerReferenceChanged

	if !changed {
		return ctrl.Result{}, nil
	}

	if err = r.updateSecret(secretInstance); err != nil {
		logd.Error(err, "Error updating Secret")
		return ctrl.Result{}, err
	}
	if ownerReferenceChanged {
		logd.Info("Added owner reference to Secret", "namespace", secretInstance.Namespace, "name", secretInstance.Name)
	}
	return ctrl.Result{}, nil
}

// ensureCSCACertificateSecretOwnerReference copies the CommonService owner
// from the managed CA Certificate to its generated Secret. A BYO CA Secret has
// no managed Certificate and therefore is intentionally left user-owned.
func ensureCSCACertificateSecretOwnerReference(cert *certmanagerv1.Certificate, secret *corev1.Secret) (bool, error) {
	if cert.Name != constant.CSCACertificate ||
		secret.Name != constant.CSCACertificateSecret ||
		cert.Spec.SecretName != secret.Name {
		return false, nil
	}

	for _, ownerRef := range cert.GetOwnerReferences() {
		if ownerRef.APIVersion == constant.APIVersion && ownerRef.Kind == constant.KindCR {
			return common.EnsureControllerOwnerReference(secret, ownerRef)
		}
	}
	return false, nil
}

// getSecret finds corresponding secret of the certmanagerv1 certificate
func (r *V1AddLabelReconciler) getSecret(cert *certmanagerv1.Certificate) (*corev1.Secret, error) {
	secretName := cert.Spec.SecretName
	secret := &corev1.Secret{}
	err := r.Reader.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: cert.Namespace}, secret)

	return secret, err
}

// updateSecret updates corresponding secret
func (r *V1AddLabelReconciler) updateSecret(secret *corev1.Secret) error {
	return r.Client.Update(context.TODO(), secret)
}

// SetupWithManager sets up the controller with the Manager.
func (r *V1AddLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	klog.V(2).Infof("Set up")

	return ctrl.NewControllerManagedBy(mgr).
		Named("certificate-v1-label").
		For(&certmanagerv1.Certificate{}).
		Complete(r)
}
