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
	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"k8s.io/klog"
)

const (
	PauseRequestAnnoKey     = "commonservices.operator.ibm.com/pause"
	SelfPauseRequestAnnoKey = "commonservices.operator.ibm.com/self-pause"
	PauseRequestValue       = "true"
)

func (r *CommonServiceReconciler) reconcilePauseRequest(instance *apiv3.CommonService) bool {

	klog.Info("Request Stage: reconcilePauseRequest")

	// if the given CommnService CR has not been existing
	if instance == nil {
		klog.Warningf("CommonService CR %s/%s is not existing", instance.Name, instance.Namespace)
		return false
	}

	// check if there is a pause request annotation in the CommonService CR
	return r.pauseRequestExists(instance)

	// future implementation: TO DO
	// check and set pauseExpire annotation
	// if the time is expired, remove the pause annotation
}

func (r *CommonServiceReconciler) pauseRequestExists(instance *apiv3.CommonService) bool {
	klog.Info("Request Stage: Checking annotations for pause request")

	// if there is pause or self-pause request annotation in the CommonService CR, pause request takes precedence over self-pause request
	var pauseRequestFound bool
	var selfpauseRequestFound bool
	if instance.ObjectMeta.Annotations != nil {
		for key := range instance.ObjectMeta.Annotations {
			if key == PauseRequestAnnoKey {
				pauseRequestFound = true
				klog.Infof("Found pause request annotation: %v", instance.ObjectMeta.Annotations[PauseRequestAnnoKey])
			} else if key == SelfPauseRequestAnnoKey {
				selfpauseRequestFound = true
				klog.Infof("Found self-pause request annotation: %v", instance.ObjectMeta.Annotations[SelfPauseRequestAnnoKey])
			}
		}
		// Pause request takes precedence over self-pause request
		if pauseRequestFound {
			return instance.ObjectMeta.Annotations[PauseRequestAnnoKey] == PauseRequestValue
		} else if selfpauseRequestFound {
			return instance.ObjectMeta.Annotations[SelfPauseRequestAnnoKey] == PauseRequestValue
		} else {
			return false
		}
	}
	return false
}
