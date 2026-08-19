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
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// EnsureControllerOwnerReference ensures that object has exactly one reference
// to owner and that the reference is controlling. Existing non-controller owner
// references are preserved.
func EnsureControllerOwnerReference(object metav1.Object, owner metav1.OwnerReference) (bool, error) {
	owner.Controller = pointer.Bool(true)
	owner.BlockOwnerDeletion = pointer.Bool(true)

	existing := object.GetOwnerReferences()
	updated := make([]metav1.OwnerReference, 0, len(existing)+1)
	foundOwner := false
	changed := false

	for _, ref := range existing {
		isOwner := ref.UID == owner.UID ||
			(ref.APIVersion == owner.APIVersion && ref.Kind == owner.Kind && ref.Name == owner.Name)
		if isOwner {
			if foundOwner {
				changed = true
				continue
			}
			foundOwner = true
			updated = append(updated, owner)
			if !equality.Semantic.DeepEqual(ref, owner) {
				changed = true
			}
			continue
		}

		if ref.Controller != nil && *ref.Controller {
			return false, fmt.Errorf("cannot set controller owner reference on %s/%s: object is already controlled by %s %s",
				object.GetNamespace(), object.GetName(), ref.Kind, ref.Name)
		}
		updated = append(updated, ref)
	}

	if !foundOwner {
		updated = append(updated, owner)
		changed = true
	}

	if changed {
		object.SetOwnerReferences(updated)
	}
	return changed, nil
}
