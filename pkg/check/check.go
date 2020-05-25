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

package check

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// CheckOriginalCs check the old version common services if installed
func CheckOriginalCs(mgr manager.Manager) (exist bool, err error) {
	reader := mgr.GetAPIReader()
	secret, err := getSecret(reader)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Get the tiller secret annotations
	annotations := secret.GetAnnotations()

	if _, ok := annotations["ibm.com/iam-service.id"]; ok {
		return true, nil
	}
	if _, ok := annotations["ibm.com/iam-service.api-key"]; ok {
		return true, nil
	}
	if _, ok := annotations["ibm.com/iam-service.name"]; ok {
		return true, nil
	}

	return false, nil
}

func getSecret(reader client.Reader) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	secretName := "tiller-secret"
	secretNs := "kube-system"

	err := reader.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: secretNs}, secret)
	return secret, err
}
