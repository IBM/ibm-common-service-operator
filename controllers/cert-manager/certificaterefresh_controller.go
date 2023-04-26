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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"

	certmanagerv1 "github.com/ibm/ibm-cert-manager-operator/apis/cert-manager/v1"
	res "github.com/ibm/ibm-cert-manager-operator/controllers/resources"
)

var logd = log.Log.WithName("controller_certificaterefresh")

// CertificateReconciler reconciles a Certificate object
type CertificateRefreshReconciler struct {
	*bootstrap.Bootstrap
	client.Client
	Scheme *runtime.Scheme
}

// //+kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete;deletecollection
// //+kubebuilder:rbac:groups=cert-manager.io,resources=certificates/status,verbs=get;update;patch
// //+kubebuilder:rbac:groups=cert-manager.io,resources=certificates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Certificate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *CertificateRefreshReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logd = log.FromContext(ctx)

	masterCR := &apiv3.CommonService{}
	if err := r.Bootstrap.Client.Get(ctx, types.NamespacedName{Namespace: r.Bootstrap.CSData.OperatorNs, Name: "common-service"}, masterCR); err != nil {
		return ctrl.Result{}, err
	}

	if !masterCR.Spec.License.Accept {
		klog.Info("Accept license by changing .spec.license.accept to true in the CertManagerConfig CR. Operator will not proceed until then")
		return ctrl.Result{Requeue: true}, nil
	}

	reqLogger := logd.WithValues("req.Namespace", req.Namespace, "req.Name", req.Name)
	reqLogger.Info("Reconciling CertificateRefresh")

	// Get the certificate that invoked reconciliation is a CA in the listOfCAs
	secret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile req
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// found ca cert or ca secret
	foundCA := false
	// check this secret has refresh label or not
	// if this secret has refresh label
	if secret.GetLabels()[res.RefreshCALabel] == "true" {
		foundCA = true
	} else {
		// Get the certificate by this secret in the same namespace
		cert, err := r.getCertificateBySecret(secret)
		foundCert := true
		if err != nil {
			if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			logd.Info("Failed to find backing Certificate object for secret", "name:", secret.Name, "namespace:", secret.Namespace)
			foundCert = false
		}

		// if we found this certificate in the same namespace
		if foundCert {
			// check this certificate has refresh label or not
			if cert.Labels[res.RefreshCALabel] == "true" {
				foundCA = true
			}
		}
	}

	if !foundCA {
		//if certificate not in the list, disregard i.e. return and don't requeue
		logd.Info("Certificate Secret doesn't need its leaf certs refreshed. Disregarding.", "Secret.Name", secret.Name, "Secret.Namespace", secret.Namespace)
		return ctrl.Result{}, nil
	}

	logd.Info("Certificate Secret is a CA, its leaf should be refreshed", "Secret.Name", secret.Name, "Secret.Namespace", secret.Namespace)

	//Get tls.crt of the CA
	tlsValueOfCA := secret.Data["tls.crt"]

	// Fetch issuers
	issuers, err := r.findIssuersBasedOnCA(secret)
	if err != nil {
		return ctrl.Result{}, err
	}

	// // Fetch all the secrets of leaf certificates issued by these issuers/clusterissuers
	var leafSecrets []*corev1.Secret

	v1LeafCerts, err := r.findV1Certs(issuers)
	if err != nil {
		logd.Error(err, "Error reading the leaf certificates for issuer - requeue the request")
		return ctrl.Result{}, err
	}

	leafSecrets, err = r.findLeafSecrets(v1LeafCerts)
	if err != nil {
		logd.Error(err, "Error finding secrets from v1 leaf certificates - requeue the request")
		return ctrl.Result{}, err
	}

	// Compare ca.crt in leaf with tls.crt of CA
	// If the values don't match, delete the secret; if error, requeue else don't requeue
	for _, leafSecret := range leafSecrets {
		if string(leafSecret.Data["ca.crt"]) != string(tlsValueOfCA) {
			logd.Info("Deleting leaf secret " + leafSecret.Name + " as ca.crt value has changed")
			if err := r.Client.Delete(context.TODO(), leafSecret); err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				return ctrl.Result{}, err
			}
		}
	}

	logd.Info("All leaf certificates refreshed for", "Secret.Name", secret.Name, "Secret.Namespace", secret.Namespace)
	return ctrl.Result{}, nil
}

// Get the certificate by secret in the same namespace
func (r *CertificateRefreshReconciler) getCertificateBySecret(secret *corev1.Secret) (*certmanagerv1.Certificate, error) {
	certName := secret.GetAnnotations()["cert-manager.io/certificate-name"]
	namespace := secret.GetNamespace()
	cert := &certmanagerv1.Certificate{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: certName, Namespace: namespace}, cert)

	return cert, err
}

// getSecret finds corresponding secret of the certificate
func (r *CertificateRefreshReconciler) getSecret(cert *certmanagerv1.Certificate) (*corev1.Secret, error) {
	secretName := cert.Spec.SecretName
	secret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: cert.Namespace}, secret)

	return secret, err
}

// findIssuersBasedOnCA finds issuers that are based on the given CA secret
func (r *CertificateRefreshReconciler) findIssuersBasedOnCA(caSecret *corev1.Secret) ([]certmanagerv1.Issuer, error) {

	var issuers []certmanagerv1.Issuer

	issuerList := &certmanagerv1.IssuerList{}
	err := r.Client.List(context.TODO(), issuerList, &client.ListOptions{Namespace: caSecret.Namespace})
	if err == nil {
		for _, issuer := range issuerList.Items {
			if issuer.Spec.CA != nil && issuer.Spec.CA.SecretName == caSecret.Name {
				issuers = append(issuers, issuer)
			}
		}
	}

	return issuers, err
}

func (r *CertificateRefreshReconciler) findV1Certs(issuers []certmanagerv1.Issuer) ([]certmanagerv1.Certificate, error) {
	var leafCerts []certmanagerv1.Certificate
	for _, i := range issuers {
		certList := &certmanagerv1.CertificateList{}
		err := r.Client.List(context.TODO(), certList, &client.ListOptions{Namespace: i.Namespace})
		if err != nil {
			return leafCerts, err
		}

		for _, c := range certList.Items {
			if c.Spec.IssuerRef.Name == i.Name {
				leafCerts = append(leafCerts, c)
			}
		}
	}
	return leafCerts, nil
}

// findLeafSecrets finds issuers that are based on the given CA secret
func (r *CertificateRefreshReconciler) findLeafSecrets(v1Certs []certmanagerv1.Certificate) ([]*corev1.Secret, error) {

	var leafSecrets []*corev1.Secret

	for _, cert := range v1Certs {
		leafSecret, err := r.getSecret(&cert)
		if err != nil {
			if errors.IsNotFound(err) {
				logd.V(2).Info("Secret not found for cert " + cert.Name)
				continue
			}
			return leafSecrets, err
		}
		leafSecrets = append(leafSecrets, leafSecret)
	}

	return leafSecrets, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateRefreshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	klog.Infof("Set up")

	// Create a new controller
	c, err := controller.New("certificaterefresh-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Certificates in the cluster
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}
