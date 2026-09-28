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
	"time"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
)

const operationTimingMaxEntries = 5

// AppendOperationTiming prepends entry to instance.Status.OperationTiming and
// trims the list to a maximum of operationTimingMaxEntries (5) entries, dropping
// the oldest when the limit is exceeded.
func AppendOperationTiming(instance *apiv3.CommonService, entry apiv3.OperationTimingEntry) {
	updated := append([]apiv3.OperationTimingEntry{entry}, instance.Status.OperationTiming...)
	if len(updated) > operationTimingMaxEntries {
		updated = updated[:operationTimingMaxEntries]
	}
	instance.Status.OperationTiming = updated
}

// FormatDuration returns a human-readable duration string rounded to the nearest
// second (e.g. "42m30s"). This matches the format expected by the operationTiming
// totalDuration and dependencyDuration fields.
func FormatDuration(d time.Duration) string {
	return d.Round(time.Second).String()
}
