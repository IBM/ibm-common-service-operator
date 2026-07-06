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
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	certmanagerv1 "github.com/ibm/ibm-cert-manager-operator/apis/cert-manager/v1"
)

var (
	restartLabel        = "certmanager.k8s.io/time-restarted"
	noRestartAnnotation = "certmanager.k8s.io/disable-auto-restart"
	t                   = "true"
	// Retry configuration for Secret verification
	secretVerificationRetryDelay    = 5 * time.Second
	secretVerificationMaxRetries    = 12 // 12 retries * 5 seconds = 60 seconds max wait
	secretVerificationBackoffFactor = 1.5
)

// CertificateReconciler reconciles a Certificate object
type PodRefreshReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// //+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch;create;update;patch
// //+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Certificate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *PodRefreshReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logd = log.FromContext(ctx)

	reqLogger := logd.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling podrefresh")

	// Get the certificate that invoked reconciliation
	cert := &certmanagerv1.Certificate{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, cert)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile req
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if cert.Status.NotBefore != nil && cert.Status.NotAfter != nil {
		// Verify that the Secret exists and has been updated before triggering pod restarts
		secretReady, requeueDelay, err := r.verifySecretReady(cert, reqLogger)
		if err != nil {
			reqLogger.Error(err, "Failed to verify Secret readiness", "Secret", cert.Spec.SecretName)
			return ctrl.Result{}, err
		}

		if !secretReady {
			reqLogger.Info("Secret not ready yet, requeuing",
				"Secret", cert.Spec.SecretName,
				"RequeueAfter", requeueDelay)
			return ctrl.Result{RequeueAfter: requeueDelay}, nil
		}

		// Secret is ready, proceed with pod restart
		reqLogger.Info("Secret verified and ready, proceeding with pod restart",
			"Secret", cert.Spec.SecretName,
			"Certificate", cert.Name)

		if err := r.restart(cert.Spec.SecretName, cert.Name, cert.Namespace, cert.Status.NotBefore.Format("2006-1-2.150405")); err != nil {
			reqLogger.Error(err, "Failed to refresh pod")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// requeue the request when certificate status is not ready to
	// ensure we don't lose a certificate update
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// verifySecretReady checks if the Secret referenced by the Certificate exists and has been updated
// Returns: (secretReady bool, requeueDelay time.Duration, error)
func (r *PodRefreshReconciler) verifySecretReady(cert *certmanagerv1.Certificate, logger logr.Logger) (bool, time.Duration, error) {
	secretName := cert.Spec.SecretName
	namespace := cert.Namespace

	// Get the Secret
	secret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Secret not found yet, will retry",
				"Secret", secretName,
				"Certificate", cert.Name)
			return false, secretVerificationRetryDelay, nil
		}
		return false, 0, fmt.Errorf("error getting Secret %s/%s: %w", namespace, secretName, err)
	}

	// Verify the Secret has been updated after the certificate renewal
	// Check if the Secret's creation timestamp or cert-manager annotations indicate it's been updated
	certNotBefore := cert.Status.NotBefore.Time

	// Check if Secret has the cert-manager annotation indicating it was updated
	certManagerAnnotation := secret.Annotations["cert-manager.io/certificate-name"]
	if certManagerAnnotation != cert.Name {
		logger.V(2).Info("Secret does not have matching cert-manager annotation",
			"Secret", secretName,
			"Expected", cert.Name,
			"Got", certManagerAnnotation)
	}

	// Verify Secret contains certificate data
	if len(secret.Data) == 0 {
		logger.Info("Secret exists but has no data yet, will retry",
			"Secret", secretName)
		return false, secretVerificationRetryDelay, nil
	}

	// Check for required certificate fields
	if _, hasTLSCert := secret.Data["tls.crt"]; !hasTLSCert {
		logger.Info("Secret missing tls.crt, will retry",
			"Secret", secretName)
		return false, secretVerificationRetryDelay, nil
	}

	if _, hasTLSKey := secret.Data["tls.key"]; !hasTLSKey {
		logger.Info("Secret missing tls.key, will retry",
			"Secret", secretName)
		return false, secretVerificationRetryDelay, nil
	}

	// Compare Secret's creation/update time with certificate's NotBefore
	// If the Secret was created/updated after the certificate's NotBefore time,
	// it likely contains the new certificate
	secretTime := secret.CreationTimestamp.Time

	// Check if Secret has been updated (look at metadata.resourceVersion or creation time)
	// If Secret is older than the certificate's NotBefore, it might still have old data
	if secretTime.Before(certNotBefore) {
		// Secret exists but might be stale - check if it's been recently modified
		// In Kubernetes, we can't directly get the last modified time, but we can infer
		// from the fact that cert-manager should have updated it
		logger.Info("Secret creation time is before certificate NotBefore, checking if recently updated",
			"Secret", secretName,
			"SecretCreationTime", secretTime,
			"CertNotBefore", certNotBefore)

		// Give cert-manager some time to update the Secret
		// If we've been waiting too long, there might be an issue
		timeSinceCertUpdate := time.Since(certNotBefore)
		if timeSinceCertUpdate < 60*time.Second {
			// Still within reasonable time window, requeue with backoff
			delay := r.calculateBackoffDelay(timeSinceCertUpdate)
			logger.Info("Waiting for Secret to be updated by cert-manager",
				"Secret", secretName,
				"TimeSinceCertUpdate", timeSinceCertUpdate,
				"NextRetryIn", delay)
			return false, delay, nil
		}

		// Been waiting too long - log warning but proceed
		// The Secret might have been updated but we can't detect it reliably
		logger.Info("Secret appears old but proceeding after timeout",
			"Secret", secretName,
			"TimeSinceCertUpdate", timeSinceCertUpdate)
	}

	logger.Info("Secret verified and ready",
		"Secret", secretName,
		"Certificate", cert.Name,
		"SecretCreationTime", secretTime,
		"CertNotBefore", certNotBefore)

	return true, 0, nil
}

// calculateBackoffDelay calculates exponential backoff delay based on elapsed time
func (r *PodRefreshReconciler) calculateBackoffDelay(elapsed time.Duration) time.Duration {
	// Calculate which retry attempt this would be
	retryNum := int(elapsed / secretVerificationRetryDelay)
	if retryNum >= secretVerificationMaxRetries {
		retryNum = secretVerificationMaxRetries - 1
	}

	// Calculate exponential backoff: baseDelay * (backoffFactor ^ retryNum)
	delay := secretVerificationRetryDelay
	for i := 0; i < retryNum; i++ {
		delay = time.Duration(float64(delay) * secretVerificationBackoffFactor)
	}

	// Cap at 30 seconds
	maxDelay := 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// pod refresh is enabled. It will edit the deployments, statefulsets, and daemonsets
// that use the secret being updated, which will trigger the pod to be restarted.
func (r *PodRefreshReconciler) restart(secret, cert, namespace string, lastUpdated string) error {
	timeNow := time.Now().Format("2006-1-2.150405")
	deployments := &appsv1.DeploymentList{}
	if err := r.Client.List(context.TODO(), deployments); err != nil {
		return fmt.Errorf("error getting deployments: %v", err)
	}
	deploymentsToUpdate, err := r.getDeploymentsNeedUpdate(secret, namespace, lastUpdated)
	if err != nil {
		return err
	}

	if err := r.updateDeploymentAnnotations(deploymentsToUpdate, cert, secret, timeNow); err != nil {
		return err
	}

	statefulsetsToUpdate, err := r.getStsNeedUpdate(secret, namespace, lastUpdated)
	if err != nil {
		return err
	}
	if err := r.updateStsAnnotations(statefulsetsToUpdate, cert, secret, timeNow); err != nil {
		return err
	}

	daemonsetsToUpdate, err := r.getDaemonSetNeedUpdate(secret, namespace, lastUpdated)
	if err != nil {
		return err
	}
	if err := r.updateDaemonSetAnnotations(daemonsetsToUpdate, cert, secret, timeNow); err != nil {
		return err
	}

	return nil
}

func (r *PodRefreshReconciler) getDeploymentsNeedUpdate(secret, namespace, lastUpdated string) ([]appsv1.Deployment, error) {
	deploymentsToUpdate := make([]appsv1.Deployment, 0)
	deployments := &appsv1.DeploymentList{}
	listOpts := &client.ListOptions{
		Namespace: namespace,
	}
	if err := r.Client.List(context.TODO(), deployments, listOpts); err != nil {
		return deploymentsToUpdate, fmt.Errorf("error getting deployments: %v", err)
	}
NEXT_DEPLOYMENT:
	for _, deployment := range deployments.Items {
		if deployment.ObjectMeta.Labels != nil && deployment.ObjectMeta.Labels[restartLabel] != "" {
			lastUpdatedTime, err := time.Parse("2006-1-2.150405", lastUpdated)
			if err != nil {
				return deploymentsToUpdate, fmt.Errorf("error parsing NotAfter time: %v", err)
			}
			labelTime := deployment.ObjectMeta.Labels[restartLabel]
			if t := strings.Split(labelTime, "."); len(t[len(t)-1]) == 4 {
				labelTime = labelTime + string("00")
			}
			restartedTime, err := time.Parse("2006-1-2.150405", labelTime)
			if err != nil {
				return deploymentsToUpdate, fmt.Errorf("error parsing time-restarted for deployment: %v", err)
			}
			if restartedTime.After(lastUpdatedTime) {
				continue
			}
		}
		for _, container := range deployment.Spec.Template.Spec.Containers {
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secret && deployment.ObjectMeta.Annotations[noRestartAnnotation] != t {
					deploymentsToUpdate = append(deploymentsToUpdate, deployment)
					continue NEXT_DEPLOYMENT
				}
			}
		}
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.Secret != nil && volume.Secret.SecretName != "" && volume.Secret.SecretName == secret && deployment.ObjectMeta.Annotations[noRestartAnnotation] != t {
				deploymentsToUpdate = append(deploymentsToUpdate, deployment)
				continue NEXT_DEPLOYMENT
			}
			if volume.Projected != nil && volume.Projected.Sources != nil && deployment.ObjectMeta.Annotations[noRestartAnnotation] != t {
				for _, source := range volume.Projected.Sources {
					if source.Secret != nil && source.Secret.Name == secret {
						deploymentsToUpdate = append(deploymentsToUpdate, deployment)
						continue NEXT_DEPLOYMENT
					}
				}
			}
		}
	}
	return deploymentsToUpdate, nil
}

func (r *PodRefreshReconciler) getStsNeedUpdate(secret, namespace, lastUpdated string) ([]appsv1.StatefulSet, error) {
	statefulsetsToUpdate := make([]appsv1.StatefulSet, 0)
	statefulsets := &appsv1.StatefulSetList{}
	listOpts := &client.ListOptions{
		Namespace: namespace,
	}
	err := r.Client.List(context.TODO(), statefulsets, listOpts)
	if err != nil {
		return statefulsetsToUpdate, fmt.Errorf("error getting statefulsets: %v", err)
	}
NEXT_STATEFULSET:
	for _, statefulset := range statefulsets.Items {
		if statefulset.ObjectMeta.Labels != nil && statefulset.ObjectMeta.Labels[restartLabel] != "" {
			lastUpdatedTime, err := time.Parse("2006-1-2.150405", lastUpdated)
			if err != nil {
				return statefulsetsToUpdate, fmt.Errorf("error parsing NotAfter time: %v", err)
			}
			labelTime := statefulset.ObjectMeta.Labels[restartLabel]
			if t := strings.Split(labelTime, "."); len(t[len(t)-1]) == 4 {
				labelTime = labelTime + string("00")
			}
			restartedTime, err := time.Parse("2006-1-2.150405", labelTime)
			if err != nil {
				return statefulsetsToUpdate, fmt.Errorf("error parsing time-restarted for statefulSet: %v", err)
			}
			if restartedTime.After(lastUpdatedTime) {
				continue
			}
		}
		for _, container := range statefulset.Spec.Template.Spec.Containers {
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secret && statefulset.ObjectMeta.Annotations[noRestartAnnotation] != t {
					statefulsetsToUpdate = append(statefulsetsToUpdate, statefulset)
					continue NEXT_STATEFULSET
				}
			}
		}
		for _, volume := range statefulset.Spec.Template.Spec.Volumes {
			if volume.Secret != nil && volume.Secret.SecretName != "" && volume.Secret.SecretName == secret && statefulset.ObjectMeta.Annotations[noRestartAnnotation] != t {
				statefulsetsToUpdate = append(statefulsetsToUpdate, statefulset)
				continue NEXT_STATEFULSET
			}
			if volume.Projected != nil && volume.Projected.Sources != nil && statefulset.ObjectMeta.Annotations[noRestartAnnotation] != t {
				for _, source := range volume.Projected.Sources {
					if source.Secret != nil && source.Secret.Name == secret {
						statefulsetsToUpdate = append(statefulsetsToUpdate, statefulset)
						continue NEXT_STATEFULSET
					}
				}
			}
		}
	}
	return statefulsetsToUpdate, nil
}

func (r *PodRefreshReconciler) getDaemonSetNeedUpdate(secret, namespace, lastUpdated string) ([]appsv1.DaemonSet, error) {
	daemonsetsToUpdate := make([]appsv1.DaemonSet, 0)
	daemonsets := &appsv1.DaemonSetList{}
	listOpts := &client.ListOptions{
		Namespace: namespace,
	}
	if err := r.Client.List(context.TODO(), daemonsets, listOpts); err != nil {
		return daemonsetsToUpdate, fmt.Errorf("error getting daemonsets: %v", err)
	}
NEXT_DAEMONSET:
	for _, daemonset := range daemonsets.Items {
		if daemonset.ObjectMeta.Labels != nil && daemonset.ObjectMeta.Labels[restartLabel] != "" {
			lastUpdatedTime, err := time.Parse("2006-1-2.150405", lastUpdated)
			if err != nil {
				return daemonsetsToUpdate, fmt.Errorf("error parsing NotAfter time: %v", err)
			}
			labelTime := daemonset.ObjectMeta.Labels[restartLabel]
			if t := strings.Split(labelTime, "."); len(t[len(t)-1]) == 4 {
				labelTime = labelTime + string("00")
			}
			restartedTime, err := time.Parse("2006-1-2.150405", labelTime)
			if err != nil {
				return daemonsetsToUpdate, fmt.Errorf("error parsing time-restarted for daemonSet: %v", err)
			}
			if restartedTime.After(lastUpdatedTime) {
				continue
			}
		}
		for _, container := range daemonset.Spec.Template.Spec.Containers {
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secret && daemonset.ObjectMeta.Annotations[noRestartAnnotation] != t {
					daemonsetsToUpdate = append(daemonsetsToUpdate, daemonset)
					continue NEXT_DAEMONSET
				}
			}
		}
		for _, volume := range daemonset.Spec.Template.Spec.Volumes {
			if volume.Secret != nil && volume.Secret.SecretName != "" && volume.Secret.SecretName == secret && daemonset.ObjectMeta.Annotations[noRestartAnnotation] != t {
				daemonsetsToUpdate = append(daemonsetsToUpdate, daemonset)
				continue NEXT_DAEMONSET
			}
			if volume.Projected != nil && volume.Projected.Sources != nil && daemonset.ObjectMeta.Annotations[noRestartAnnotation] != t {
				for _, source := range volume.Projected.Sources {
					if source.Secret != nil && source.Secret.Name == secret {
						daemonsetsToUpdate = append(daemonsetsToUpdate, daemonset)
						continue NEXT_DAEMONSET
					}
				}
			}
		}
	}
	return daemonsetsToUpdate, nil
}

func (r *PodRefreshReconciler) updateDeploymentAnnotations(deploymentsToUpdate []appsv1.Deployment, cert, secret, timeNow string) error {
	for _, deployment := range deploymentsToUpdate {
		//in case of deployments not having labels section, create the label section
		if deployment.ObjectMeta.Labels == nil {
			deployment.ObjectMeta.Labels = make(map[string]string)
		}
		if deployment.Spec.Template.ObjectMeta.Labels == nil {
			deployment.Spec.Template.ObjectMeta.Labels = make(map[string]string)
		}
		deployment.ObjectMeta.Labels[restartLabel] = timeNow
		deployment.Spec.Template.ObjectMeta.Labels[restartLabel] = timeNow
		err := r.Client.Update(context.TODO(), &deployment)
		if err != nil {
			return fmt.Errorf("error updating deployment: %v", err)
		}
		logd.Info("Cert-Manager Restarting Resource:", "Certificate=", cert, "Secret=", secret, "Deployment=", deployment.ObjectMeta.Name, "TimeNow=", timeNow)
	}
	return nil
}

func (r *PodRefreshReconciler) updateStsAnnotations(statefulsetsToUpdate []appsv1.StatefulSet, cert, secret, timeNow string) error {
	for _, statefulset := range statefulsetsToUpdate {
		if statefulset.ObjectMeta.Labels == nil {
			statefulset.ObjectMeta.Labels = make(map[string]string)
		}
		if statefulset.Spec.Template.ObjectMeta.Labels == nil {
			statefulset.Spec.Template.ObjectMeta.Labels = make(map[string]string)
		}
		statefulset.ObjectMeta.Labels[restartLabel] = timeNow
		statefulset.Spec.Template.ObjectMeta.Labels[restartLabel] = timeNow
		if err := r.Client.Update(context.TODO(), &statefulset); err != nil {
			return fmt.Errorf("error updating statefulset: %v", err)
		}
		logd.Info("Cert-Manager Restarting Resource:", "Certificate=", cert, "Secret=", secret, "StatefulSet=", statefulset.ObjectMeta.Name, "TimeNow=", timeNow)
	}
	return nil
}

func (r *PodRefreshReconciler) updateDaemonSetAnnotations(daemonsetsToUpdate []appsv1.DaemonSet, cert, secret, timeNow string) error {
	for _, daemonset := range daemonsetsToUpdate {
		if daemonset.ObjectMeta.Labels == nil {
			daemonset.ObjectMeta.Labels = make(map[string]string)
		}
		if daemonset.Spec.Template.ObjectMeta.Labels == nil {
			daemonset.Spec.Template.ObjectMeta.Labels = make(map[string]string)
		}
		daemonset.ObjectMeta.Labels[restartLabel] = timeNow
		daemonset.Spec.Template.ObjectMeta.Labels[restartLabel] = timeNow
		if err := r.Client.Update(context.TODO(), &daemonset); err != nil {
			return fmt.Errorf("error updating daemonset: %v", err)
		}
		logd.Info("Cert-Manager Restarting Resource:", "Certificate=", cert, "Secret=", secret, "DaemonSet=", daemonset.ObjectMeta.Name, "TimeNow=", timeNow)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodRefreshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	klog.V(2).Infof("Set up")

	// Create a new controller
	c, err := controller.New("podrefresh-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Certificates in the cluster
	err = c.Watch(source.Kind(mgr.GetCache(), &certmanagerv1.Certificate{}), &handler.EnqueueRequestForObject{}, isExpiredPredicate{})
	if err != nil {
		return err
	}

	return nil
}

type isExpiredPredicate struct{}

func (isExpiredPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (isExpiredPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (isExpiredPredicate) Update(e event.UpdateEvent) bool {
	oldCert := (e.ObjectOld).(*certmanagerv1.Certificate)
	updatedCert := (e.ObjectNew).(*certmanagerv1.Certificate)
	if oldCert.Status.NotAfter == nil && updatedCert.Status.NotAfter != nil {
		return true
	}
	if updatedCert.Status.NotAfter != nil && oldCert.Status.NotAfter != nil &&
		!oldCert.Status.NotAfter.Time.Equal(updatedCert.Status.NotAfter.Time) {
		return true
	}
	return false
}

func (isExpiredPredicate) Generic(e event.GenericEvent) bool {
	return false
}
