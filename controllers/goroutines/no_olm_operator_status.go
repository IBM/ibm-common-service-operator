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

package goroutines

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/bootstrap"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/constant"
)

var ctx_NoOLM = context.Background()

// UpdateCsCrStatus will update cs cr status according to each bedrock operator
func UpdateNoOLMCsCrStatus(bs *bootstrap.Bootstrap) {
	for {
		instance := &apiv3.CommonService{}
		if err := bs.Reader.Get(ctx_NoOLM, types.NamespacedName{Name: "common-service", Namespace: bs.CSData.OperatorNs}, instance); err != nil {
			if !errors.IsNotFound(err) {
				klog.Warningf("Faild to get CommonService CR %v/%v: %v", instance.GetNamespace(), instance.GetName(), err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		var operatorSlice []apiv3.BedrockOperator

		for _, name := range constant.DeploymentsName {
			var opt apiv3.BedrockOperator
			var err error

			opt, err = getNoOLMBedrockOperator(bs, name, bs.CSData.CPFSNs)

			if err == nil {
				operatorSlice = append(operatorSlice, opt)
			} else if !errors.IsNotFound(err) {
				klog.Errorf("Failed to check operator %s: %v", name, err)
			}
		}

		// update status for each operators: BedrockOperators list
		instance.Status.BedrockOperators = operatorSlice

		// update operators overall status: OverallStatus
		instance.Status.OverallStatus = "Succeeded"
		for _, opt := range operatorSlice {
			if opt.OperatorStatus != "Succeeded" {
				instance.Status.OverallStatus = "NotReady"
				break
			}
		}

		if err := bs.Client.Status().Update(ctx_NoOLM, instance); err != nil {
			klog.Warning(err)
		}

		time.Sleep(2 * time.Minute)
	}
}

func getNoOLMBedrockOperator(bs *bootstrap.Bootstrap, name, namespace string) (apiv3.BedrockOperator, error) {
	var opt apiv3.BedrockOperator
	opt.Name = name

	// fetch subscription
	deployment := &appsv1.Deployment{}
	deploymentKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	if err := bs.Client.Get(ctx_NoOLM, deploymentKey, deployment); err != nil {
		return opt, err
	}

	if deployment.Status.ReadyReplicas != *deployment.Spec.Replicas {
		opt.Troubleshooting = "Operator status is not healthy, please check " + constant.GeneralTroubleshooting + " for more information"
	}

	return opt, nil
}
