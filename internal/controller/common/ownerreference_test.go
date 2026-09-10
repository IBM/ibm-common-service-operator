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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

func newRef(apiVersion, kind, name string, uid types.UID, controller bool) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
		UID:        uid,
		Controller: pointer.Bool(controller),
	}
}

func TestEnsureControllerOwnerReference_NoExisting(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetNamespace("ns")
	obj.SetName("cert")

	owner := metav1.OwnerReference{
		APIVersion: "operator.ibm.com/v3",
		Kind:       "CommonService",
		Name:       "common-service",
		UID:        types.UID("uid-master"),
	}

	changed, err := EnsureControllerOwnerReference(obj, owner, false)
	require.NoError(t, err)
	assert.True(t, changed)
	refs := obj.GetOwnerReferences()
	require.Len(t, refs, 1)
	assert.Equal(t, types.UID("uid-master"), refs[0].UID)
	assert.True(t, *refs[0].Controller)
}

func TestEnsureControllerOwnerReference_Idempotent(t *testing.T) {
	owner := metav1.OwnerReference{
		APIVersion: "operator.ibm.com/v3",
		Kind:       "CommonService",
		Name:       "common-service",
		UID:        types.UID("uid-master"),
	}
	obj := &unstructured.Unstructured{}
	obj.SetNamespace("ns")
	obj.SetName("cert")
	// Pre-set with both Controller and BlockOwnerDeletion as the function would
	// have written them on a previous reconcile.
	obj.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion:         "operator.ibm.com/v3",
		Kind:               "CommonService",
		Name:               "common-service",
		UID:                types.UID("uid-master"),
		Controller:         pointer.Bool(true),
		BlockOwnerDeletion: pointer.Bool(true),
	}})

	changed, err := EnsureControllerOwnerReference(obj, owner, false)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Len(t, obj.GetOwnerReferences(), 1)
}

// TestEnsureControllerOwnerReference_TakeoverFromSameKind verifies that when
// replaceExistingSameKind is true and the object is already controlled by a
// different instance of the same Kind, the stale reference is replaced.
// This models the upgrade path: im-common-service → common-service.
func TestEnsureControllerOwnerReference_TakeoverFromSameKind(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetNamespace("a1")
	obj.SetName("cs-ca-certificate")
	obj.SetOwnerReferences([]metav1.OwnerReference{
		newRef("operator.ibm.com/v3", "CommonService", "im-common-service", "uid-im", true),
	})

	owner := metav1.OwnerReference{
		APIVersion: "operator.ibm.com/v3",
		Kind:       "CommonService",
		Name:       "common-service",
		UID:        types.UID("uid-master"),
	}

	changed, err := EnsureControllerOwnerReference(obj, owner, true)
	require.NoError(t, err)
	assert.True(t, changed)
	refs := obj.GetOwnerReferences()
	require.Len(t, refs, 1, "stale im-common-service ref must be removed")
	assert.Equal(t, types.UID("uid-master"), refs[0].UID)
	assert.Equal(t, "common-service", refs[0].Name)
	assert.True(t, *refs[0].Controller)
}

// TestEnsureControllerOwnerReference_TakeoverBlockedWithoutFlag verifies that
// the same-Kind takeover is rejected when replaceExistingSameKind is false,
// i.e. when the caller is not the designated master CR.
func TestEnsureControllerOwnerReference_TakeoverBlockedWithoutFlag(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetNamespace("a1")
	obj.SetName("cs-ca-certificate")
	obj.SetOwnerReferences([]metav1.OwnerReference{
		newRef("operator.ibm.com/v3", "CommonService", "im-common-service", "uid-im", true),
	})

	owner := metav1.OwnerReference{
		APIVersion: "operator.ibm.com/v3",
		Kind:       "CommonService",
		Name:       "common-service",
		UID:        types.UID("uid-master"),
	}

	_, err := EnsureControllerOwnerReference(obj, owner, false)
	require.Error(t, err, "same-Kind takeover must be rejected when replaceExistingSameKind=false")
	// Object must be unchanged.
	refs := obj.GetOwnerReferences()
	require.Len(t, refs, 1)
	assert.Equal(t, "im-common-service", refs[0].Name, "stale ref must not have been removed")
}

func TestEnsureControllerOwnerReference_RejectsUnrelatedController(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetNamespace("ns")
	obj.SetName("cert")
	// Owned by a completely different Kind.
	obj.SetOwnerReferences([]metav1.OwnerReference{
		newRef("apps/v1", "Deployment", "my-deploy", "uid-deploy", true),
	})

	owner := metav1.OwnerReference{
		APIVersion: "operator.ibm.com/v3",
		Kind:       "CommonService",
		Name:       "common-service",
		UID:        types.UID("uid-master"),
	}

	_, err := EnsureControllerOwnerReference(obj, owner, true)
	assert.Error(t, err, "unrelated-Kind controllers must always be rejected regardless of flag")
}
