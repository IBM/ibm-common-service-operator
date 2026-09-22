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

package controllers

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
)

func newEntry(phase string) apiv3.OperationTimingEntry {
	now := metav1.Now()
	return apiv3.OperationTimingEntry{
		StartTime:     now,
		EndTime:       now,
		TotalDuration: "0s",
		Phase:         phase,
	}
}

func TestAppendOperationTiming_PrependsMostRecent(t *testing.T) {
	instance := &apiv3.CommonService{}
	first := newEntry(apiv3.CRSucceeded)
	AppendOperationTiming(instance, first)

	if len(instance.Status.OperationTiming) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(instance.Status.OperationTiming))
	}

	second := newEntry(apiv3.CRFailed)
	AppendOperationTiming(instance, second)

	if len(instance.Status.OperationTiming) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(instance.Status.OperationTiming))
	}
	// Most recent must be first.
	if instance.Status.OperationTiming[0].Phase != apiv3.CRFailed {
		t.Errorf("expected most recent entry (index 0) to have phase %s, got %s",
			apiv3.CRFailed, instance.Status.OperationTiming[0].Phase)
	}
	if instance.Status.OperationTiming[1].Phase != apiv3.CRSucceeded {
		t.Errorf("expected older entry (index 1) to have phase %s, got %s",
			apiv3.CRSucceeded, instance.Status.OperationTiming[1].Phase)
	}
}

func TestAppendOperationTiming_CapsFiveEntries(t *testing.T) {
	instance := &apiv3.CommonService{}

	// Add 6 entries; the oldest (first added) must be dropped.
	for i := 0; i < 6; i++ {
		AppendOperationTiming(instance, newEntry(apiv3.CRSucceeded))
	}

	if len(instance.Status.OperationTiming) != operationTimingMaxEntries {
		t.Errorf("expected %d entries after 6 appends, got %d",
			operationTimingMaxEntries, len(instance.Status.OperationTiming))
	}
}

func TestAppendOperationTiming_ExistingEntriesPreserved(t *testing.T) {
	instance := &apiv3.CommonService{}

	// Pre-populate with 3 entries.
	for i := 0; i < 3; i++ {
		AppendOperationTiming(instance, newEntry(apiv3.CRSucceeded))
	}

	// Adding a Failed entry must keep the 3 Succeeded entries below it.
	AppendOperationTiming(instance, newEntry(apiv3.CRFailed))

	if len(instance.Status.OperationTiming) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(instance.Status.OperationTiming))
	}
	if instance.Status.OperationTiming[0].Phase != apiv3.CRFailed {
		t.Errorf("newest entry should be Failed, got %s", instance.Status.OperationTiming[0].Phase)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{42*time.Minute + 30*time.Second, "42m30s"},
		{16*time.Minute + 500*time.Millisecond, "16m1s"}, // 500ms rounds up to 1s
		{0, "0s"},
		{time.Second, "1s"},
		{time.Hour + 5*time.Minute, "1h5m0s"},
	}
	for _, tc := range cases {
		got := FormatDuration(tc.d)
		if got != tc.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}
