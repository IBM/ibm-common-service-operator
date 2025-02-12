//
// Copyright 2022 IBM Corporation
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

package bootstrap

import (
	"context"
	"fmt"
	"testing"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	// Add the v1alpha1 version of the operators.coreos.com API to the scheme
	_ = olmv1alpha1.AddToScheme(scheme.Scheme)
}

func TestCheckOperatorCSV(t *testing.T) {
	// Create a fake client
	fakeClient := fake.NewClientBuilder().Build()

	// Create a Bootstrap instance
	bootstrap := &Bootstrap{
		Client: fakeClient,
	}

	// Define the packageManifest and operatorNs
	packageManifest := "ibm-common-service-operator"
	operatorNs := "cpfs-operator-ns"

	// Create a SubscriptionList with a single item
	subList := &olmv1alpha1.SubscriptionList{
		Items: []olmv1alpha1.Subscription{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "subscription-1",
					Namespace: operatorNs,
					Labels: map[string]string{
						"operators.coreos.com/" + packageManifest + "." + operatorNs: "",
					},
				},
				Spec: &olmv1alpha1.SubscriptionSpec{
					Channel: "v1.0",
				},
				Status: olmv1alpha1.SubscriptionStatus{
					InstalledCSV: "ibm-common-service-operator.v1.0.0",
				},
			},
		},
	}

	var err error
	for _, item := range subList.Items {
		err = fakeClient.Create(context.TODO(), &item)
		assert.NoError(t, err)
	}

	// Test case 1: Single subscription found with valid semver
	result, err := bootstrap.checkOperatorCSV("subscription-1", packageManifest, operatorNs)
	assert.True(t, result)
	assert.NoError(t, err)

	// Test case 2: Multiple subscriptions found and not subscription name matched
	err = fakeClient.Create(context.TODO(), &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "subscription-2",
			Namespace: operatorNs,
			Labels: map[string]string{
				"operators.coreos.com/" + packageManifest + "." + operatorNs: "",
			},
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel: "v2.0",
		},
		Status: olmv1alpha1.SubscriptionStatus{
			InstalledCSV: "ibm-common-service-operator.v2.0.0",
		},
	})
	assert.NoError(t, err)

	result, err = bootstrap.checkOperatorCSV("subscription-non-match", packageManifest, operatorNs)
	assert.False(t, result)
	assert.EqualError(t, err, fmt.Sprintf("multiple subscriptions found by packageManifest %s and operatorNs %s", packageManifest, operatorNs))

	// Test case 3: No subscription found
	err = fakeClient.DeleteAllOf(context.TODO(), &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "Subscription",
		},
	}, &client.DeleteAllOfOptions{
		ListOptions: client.ListOptions{
			Namespace: operatorNs,
		},
	})
	assert.NoError(t, err)

	result, err = bootstrap.checkOperatorCSV("subscription-1", packageManifest, operatorNs)
	assert.False(t, result)
	assert.EqualError(t, err, fmt.Sprintf("no subscription found by packageManifest %s and operatorNs %s", packageManifest, operatorNs))

	// Test case 4: Invalid semver in channel
	err = fakeClient.Create(context.TODO(), &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "subscription-3",
			Namespace: operatorNs,
			Labels: map[string]string{
				"operators.coreos.com/" + packageManifest + "." + operatorNs: "",
			},
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel: "invalid-semver",
		},
		Status: olmv1alpha1.SubscriptionStatus{
			InstalledCSV: "ibm-common-service-operator.v1.0.0",
		},
	})
	assert.NoError(t, err)

	result, err = bootstrap.checkOperatorCSV("subscription-3", packageManifest, operatorNs)
	assert.False(t, result)
	assert.NoError(t, err)

	// Test case 5: Small semver in channel
	// InstalledCSV: "ibm-common-service-operator.v1.0.0", Channel: "v0.1"
	subscription := &olmv1alpha1.Subscription{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "subscription-3", Namespace: "cpfs-operator-ns"}, subscription)
	assert.NoError(t, err)

	subscription.Spec.Channel = "v0.1"
	err = fakeClient.Update(context.TODO(), subscription)
	assert.NoError(t, err)

	result, err = bootstrap.checkOperatorCSV("subscription-3", packageManifest, operatorNs)
	assert.True(t, result)
	assert.NoError(t, err)

	// Test case 6: Large semver in channel
	// InstalledCSV: "ibm-common-service-operator.v1.0.0", Channel: "v1.1"
	subscription = &olmv1alpha1.Subscription{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "subscription-3", Namespace: "cpfs-operator-ns"}, subscription)
	assert.NoError(t, err)

	subscription.Spec.Channel = "v1.1"
	err = fakeClient.Update(context.TODO(), subscription)
	assert.NoError(t, err)

	result, err = bootstrap.checkOperatorCSV("subscription-3", packageManifest, operatorNs)
	assert.False(t, result)
	assert.NoError(t, err)

	// Test case 7: same semver in channel and installedCSV
	// InstalledCSV: "ibm-common-service-operator.v1.0.0", Channel: "v1.0"
	subscription = &olmv1alpha1.Subscription{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "subscription-3", Namespace: "cpfs-operator-ns"}, subscription)
	assert.NoError(t, err)

	subscription.Spec.Channel = "v1.0"
	err = fakeClient.Update(context.TODO(), subscription)
	assert.NoError(t, err)

	result, err = bootstrap.checkOperatorCSV("subscription-3", packageManifest, operatorNs)
	assert.True(t, result)
	assert.NoError(t, err)

}

func TestWaitOperatorCSV(t *testing.T) {
	// Create a fake client
	fakeClient := fake.NewClientBuilder().Build()

	// Create a Bootstrap instance
	bootstrap := &Bootstrap{
		Client: fakeClient,
	}

	// Define the packageManifest and operatorNs
	packageManifest := "ibm-common-service-operator"
	operatorNs := "cpfs-operator-ns"

	// Test case 1: Operator CSV is not installed yet
	err := fakeClient.Create(context.TODO(), &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "subscription-1",
			Namespace: operatorNs,
			Labels: map[string]string{
				"operators.coreos.com/" + packageManifest + "." + operatorNs: "",
			},
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel: "v1.2",
		},
		Status: olmv1alpha1.SubscriptionStatus{
			InstalledCSV: "ibm-common-service-operator.v1.0.0",
		},
	})
	assert.NoError(t, err)

	// additional go routine to update the subscription status
	go func() {
		// sleep for 3 seconds to simulate the operator CSV installation
		<-time.After(3 * time.Second)

		subscription := &olmv1alpha1.Subscription{}
		err := fakeClient.Get(context.TODO(), types.NamespacedName{Name: "subscription-1", Namespace: "cpfs-operator-ns"}, subscription)
		assert.NoError(t, err)

		subscription.Status.InstalledCSV = "ibm-common-service-operator.v1.2.0"
		err = fakeClient.Update(context.TODO(), subscription)
		assert.NoError(t, err)
	}()

	isWaiting, err := bootstrap.waitOperatorCSV("subscription-1", packageManifest, operatorNs)
	assert.True(t, isWaiting)
	assert.NoError(t, err)

	// Test case 2: Operator CSV is already installed
	isWaiting, err = bootstrap.waitOperatorCSV("subscription-1", packageManifest, operatorNs)
	assert.False(t, isWaiting)
	assert.NoError(t, err)
}
