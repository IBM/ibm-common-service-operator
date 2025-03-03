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
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
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
		Reader: fakeClient,
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
		Reader: fakeClient,
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

func TestFetchSubscription(t *testing.T) {
	// Setup
	ctx := context.TODO()
	operatorNs := "test-ns"
	packageManifest := "test-package"

	// Create a fake client
	fakeClient := fake.NewClientBuilder().Build()

	// Create a Bootstrap instance with the fake client as both Client and Reader
	bootstrap := &Bootstrap{
		Client: fakeClient,
		Reader: fakeClient,
	}

	// Test case 1: Success - subscription found by name
	sub1 := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sub1",
			Namespace: operatorNs,
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel: "v1.0",
			Package: packageManifest,
		},
	}
	err := fakeClient.Create(ctx, sub1)
	assert.NoError(t, err)

	result, err := bootstrap.fetchSubscription("sub1", packageManifest, operatorNs)
	assert.NoError(t, err)
	assert.Equal(t, "sub1", result.Name)

	// Test case 2: Fallback to list - subscription found by label
	sub2 := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sub2",
			Namespace: operatorNs,
			Labels: map[string]string{
				"operators.coreos.com/" + packageManifest + "." + operatorNs: "",
			},
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel: "v1.0",
			Package: packageManifest,
		},
	}
	err = fakeClient.Create(ctx, sub2)
	assert.NoError(t, err)

	result, err = bootstrap.fetchSubscription("non-existent", packageManifest, operatorNs)
	assert.NoError(t, err)
	assert.Equal(t, "sub2", result.Name)

	// Test case 3: Multiple subscriptions found by label
	sub3 := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sub3",
			Namespace: operatorNs,
			Labels: map[string]string{
				"operators.coreos.com/" + packageManifest + "." + operatorNs: "",
			},
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel: "v1.0",
			Package: packageManifest,
		},
	}
	err = fakeClient.Create(ctx, sub3)
	assert.NoError(t, err)

	result, err = bootstrap.fetchSubscription("non-existent", packageManifest, operatorNs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple subscriptions found")
	assert.Nil(t, result)

	// Clean up for next test
	err = fakeClient.Delete(ctx, sub2)
	assert.NoError(t, err)
	err = fakeClient.Delete(ctx, sub3)
	assert.NoError(t, err)

	// Test case 4: No subscriptions found by label
	result, err = bootstrap.fetchSubscription("non-existent", packageManifest, operatorNs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no subscription found")
	assert.Nil(t, result)

	// Test case 5: Different namespace
	sub4 := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sub4",
			Namespace: "different-ns",
			Labels: map[string]string{
				"operators.coreos.com/" + packageManifest + ".different-ns": "",
			},
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel: "v1.0",
			Package: packageManifest,
		},
	}
	err = fakeClient.Create(ctx, sub4)
	assert.NoError(t, err)

	result, err = bootstrap.fetchSubscription("sub4", packageManifest, operatorNs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no subscription found")
	assert.Nil(t, result)
}

func TestSetOperatorStatus(t *testing.T) {
	// Setup test constants
	const (
		testName            = "test-operator"
		testPackageManifest = "test-package"
		testNamespace       = "test-namespace"
		testInstalledCSV    = "test-operator.v1.0.0"
		testCurrentCSV      = "test-operator.v1.0.0"
		testInstallPlanName = "install-plan-1"
	)

	// Add CSVPhaseSucceeded constant for testing
	CSVPhaseSucceeded := "Succeeded"

	// Test cases
	testCases := []struct {
		name             string
		setupMocks       func(client.Client)
		expectedStatus   apiv3.BedrockOperator
		expectError      bool
		expectedEventMsg string
	}{
		{
			name: "Successfully get operator status",
			setupMocks: func(c client.Client) {
				sub := &olmv1alpha1.Subscription{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testName,
						Namespace: testNamespace,
					},
					Status: olmv1alpha1.SubscriptionStatus{
						InstalledCSV: testInstalledCSV,
						CurrentCSV:   testCurrentCSV,
						Install: &olmv1alpha1.InstallPlanReference{
							Name: testInstallPlanName,
						},
					},
				}
				c.Create(context.TODO(), sub)

				csv := &olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testInstalledCSV,
						Namespace: testNamespace,
					},
					Status: olmv1alpha1.ClusterServiceVersionStatus{
						Conditions: []olmv1alpha1.ClusterServiceVersionCondition{
							{
								Phase: olmv1alpha1.ClusterServiceVersionPhase(CSVPhaseSucceeded),
							},
						},
					},
				}
				c.Create(context.TODO(), csv)
			},
			expectedStatus: apiv3.BedrockOperator{
				Name:               "test-operator",
				Version:            "v1.0.0",
				OperatorStatus:     CSVPhaseSucceeded,
				SubscriptionStatus: "Succeeded", // simulating CRSucceeded
				InstallPlanName:    testInstallPlanName,
				Troubleshooting:    "",
			},
			expectError: false,
		},
		{
			name: "CSV not found",
			setupMocks: func(c client.Client) {
				sub := &olmv1alpha1.Subscription{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testName,
						Namespace: testNamespace,
					},
					Status: olmv1alpha1.SubscriptionStatus{
						InstalledCSV: testInstalledCSV,
						CurrentCSV:   testCurrentCSV,
						Install: &olmv1alpha1.InstallPlanReference{
							Name: testInstallPlanName,
						},
					},
				}
				c.Create(context.TODO(), sub)
				// No CSV created
			},
			expectedStatus: apiv3.BedrockOperator{
				Name:               "test-operator",
				Version:            "v1.0.0",
				OperatorStatus:     "NotReady", // simulating CRNotReady
				SubscriptionStatus: "Succeeded",
				InstallPlanName:    testInstallPlanName,
				Troubleshooting:    fmt.Sprintf("Operator status is not healthy, please check %s for more information", constant.GeneralTroubleshooting),
			},
			expectError:      false,
			expectedEventMsg: "ClusterServiceVersion",
		},
		{
			name: "CSV with no conditions",
			setupMocks: func(c client.Client) {
				sub := &olmv1alpha1.Subscription{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testName,
						Namespace: testNamespace,
					},
					Status: olmv1alpha1.SubscriptionStatus{
						InstalledCSV: testInstalledCSV,
						CurrentCSV:   testCurrentCSV,
						Install: &olmv1alpha1.InstallPlanReference{
							Name: testInstallPlanName,
						},
					},
				}
				c.Create(context.TODO(), sub)

				csv := &olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testInstalledCSV,
						Namespace: testNamespace,
					},
				}
				c.Create(context.TODO(), csv)
			},
			expectedStatus: apiv3.BedrockOperator{
				Name:               "test-operator",
				Version:            "v1.0.0",
				OperatorStatus:     "NotReady",
				SubscriptionStatus: "Succeeded",
				InstallPlanName:    testInstallPlanName,
				Troubleshooting:    fmt.Sprintf("Operator status is not healthy, please check %s for more information", constant.GeneralTroubleshooting),
			},
			expectError:      false,
			expectedEventMsg: "ClusterServiceVersion",
		},
		{
			name: "No InstallPlan",
			setupMocks: func(c client.Client) {
				sub := &olmv1alpha1.Subscription{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testName,
						Namespace: testNamespace,
					},
					Status: olmv1alpha1.SubscriptionStatus{
						InstalledCSV: testInstalledCSV,
						CurrentCSV:   testCurrentCSV,
						// No Install field
					},
				}
				c.Create(context.TODO(), sub)

				csv := &olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testInstalledCSV,
						Namespace: testNamespace,
					},
					Status: olmv1alpha1.ClusterServiceVersionStatus{
						Conditions: []olmv1alpha1.ClusterServiceVersionCondition{
							{
								Phase: olmv1alpha1.ClusterServiceVersionPhase(CSVPhaseSucceeded),
							},
						},
					},
				}
				c.Create(context.TODO(), csv)
			},
			expectedStatus: apiv3.BedrockOperator{
				Name:               "test-operator",
				Version:            "v1.0.0",
				OperatorStatus:     CSVPhaseSucceeded,
				SubscriptionStatus: "Failed", // simulating CRFailed
				InstallPlanName:    "Not Found",
				Troubleshooting:    fmt.Sprintf("Operator status is not healthy, please check %s for more information", constant.GeneralTroubleshooting),
			},
			expectError:      false,
			expectedEventMsg: "Subscription",
		},
		{
			name: "Current CSV doesn't match Installed CSV",
			setupMocks: func(c client.Client) {
				sub := &olmv1alpha1.Subscription{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testName,
						Namespace: testNamespace,
					},
					Status: olmv1alpha1.SubscriptionStatus{
						InstalledCSV: testInstalledCSV,
						CurrentCSV:   "test-operator.v1.1.0", // Different version
						Install: &olmv1alpha1.InstallPlanReference{
							Name: testInstallPlanName,
						},
						State: olmv1alpha1.SubscriptionStateUpgradePending,
					},
				}
				c.Create(context.TODO(), sub)

				csv := &olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testInstalledCSV,
						Namespace: testNamespace,
					},
					Status: olmv1alpha1.ClusterServiceVersionStatus{
						Conditions: []olmv1alpha1.ClusterServiceVersionCondition{
							{
								Phase: olmv1alpha1.ClusterServiceVersionPhase(CSVPhaseSucceeded),
							},
						},
					},
				}
				c.Create(context.TODO(), csv)
			},
			expectedStatus: apiv3.BedrockOperator{
				Name:               "test-operator",
				Version:            "v1.0.0",
				OperatorStatus:     CSVPhaseSucceeded,
				SubscriptionStatus: "UpgradePending",
				InstallPlanName:    testInstallPlanName,
				Troubleshooting:    fmt.Sprintf("Operator status is not healthy, please check %s for more information", constant.GeneralTroubleshooting),
			},
			expectError:      false,
			expectedEventMsg: "Subscription",
		},
		{
			name: "No Installed CSV",
			setupMocks: func(c client.Client) {
				sub := &olmv1alpha1.Subscription{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testName,
						Namespace: testNamespace,
					},
					Status: olmv1alpha1.SubscriptionStatus{
						Install: &olmv1alpha1.InstallPlanReference{
							Name: testInstallPlanName,
						},
						State: olmv1alpha1.SubscriptionStateUpgradePending,
						// No InstalledCSV
					},
				}
				c.Create(context.TODO(), sub)
			},
			expectedStatus: apiv3.BedrockOperator{
				Name:               testName,
				Version:            "",
				OperatorStatus:     "NotReady",
				SubscriptionStatus: olmv1alpha1.SubscriptionStateUpgradePending, // simulating SubscriptionStateUpgradePending
				InstallPlanName:    testInstallPlanName,
				Troubleshooting:    fmt.Sprintf("Operator status is not healthy, please check %s for more information", constant.GeneralTroubleshooting),
			},
			expectError:      false,
			expectedEventMsg: "Subscription",
		},
		{
			name: "Subscription fetch error",
			setupMocks: func(c client.Client) {
				// Don't create any resources to simulate error
			},
			expectedStatus: apiv3.BedrockOperator{
				Name: testName,
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fake client for the test case
			fakeClient := fake.NewClientBuilder().Build()

			// Apply test setup
			tc.setupMocks(fakeClient)

			// Create a mock event recorder
			recorder := record.NewFakeRecorder(5)

			// Create bootstrap instance
			bootstrap := &Bootstrap{
				Client:        fakeClient,
				Reader:        fakeClient,
				EventRecorder: recorder,
			}

			// Create a test instance
			instance := &apiv3.CommonService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-instance",
					Namespace: testNamespace,
				},
			}

			// Mock the apiv3 constants for test
			apiv3CRSucceeded := "Succeeded"
			apiv3CRNotReady := "NotReady"
			apiv3CRFailed := "Failed"

			// Call the function with our test setup
			result, err := bootstrap.setOperatorStatus(instance, testName, testPackageManifest, testNamespace)

			// Check for expected errors
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Convert result to our mock type for comparison
				actualResult := apiv3.BedrockOperator{
					Name:               result.Name,
					Version:            result.Version,
					OperatorStatus:     result.OperatorStatus,
					SubscriptionStatus: result.SubscriptionStatus,
					InstallPlanName:    result.InstallPlanName,
					Troubleshooting:    result.Troubleshooting,
				}

				// Replace constants with our mock values for comparison
				expectedResult := tc.expectedStatus
				if expectedResult.OperatorStatus == "Succeeded" {
					expectedResult.OperatorStatus = apiv3CRSucceeded
				} else if expectedResult.OperatorStatus == "NotReady" {
					expectedResult.OperatorStatus = apiv3CRNotReady
				}

				if expectedResult.SubscriptionStatus == "Succeeded" {
					expectedResult.SubscriptionStatus = apiv3CRSucceeded
				} else if expectedResult.SubscriptionStatus == "Failed" {
					expectedResult.SubscriptionStatus = apiv3CRFailed
				}

				assert.Equal(t, expectedResult, actualResult)

				// Check for events if we expect them
				if tc.expectedEventMsg != "" {
					select {
					case event := <-recorder.Events:
						assert.Contains(t, event, "Warning")
						assert.Contains(t, event, "Bedrock Operator Failed")
						assert.Contains(t, event, tc.expectedEventMsg)
					default:
						if tc.expectedEventMsg != "" {
							t.Error("Expected event but none was recorded")
						}
					}
				}
			}
		})
	}
}
