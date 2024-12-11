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
	"fmt"
	"strings"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/bootstrap"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/constant"
)

var ctx = context.Background()

// UpdateCsCrStatus will update cs cr status according to each bedrock operator
func UpdateCsCrStatus(bs *bootstrap.Bootstrap) {
	for {
		instance := &apiv3.CommonService{}
		if err := bs.Reader.Get(ctx, types.NamespacedName{Name: "common-service", Namespace: bs.CSData.OperatorNs}, instance); err != nil {
			if !errors.IsNotFound(err) {
				klog.Warningf("Faild to get CommonService CR %v/%v: %v", instance.GetNamespace(), instance.GetName(), err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		var operatorSlice []apiv3.BedrockOperator

		operatorsName := []string{}

		// wait ODLM OperandRegistry CR resources
		if err := bs.WaitResourceReady("operator.ibm.com/v1alpha1", "OperandRegistry"); err != nil {
			klog.Error("Failed to wait for resource ready with kind: OperandRegistry, apiGroupVersion: operator.ibm.com/v1alpha1")
			continue
		}

		opreg, err := bs.GetOperandRegistry(ctx, "common-service", bs.CSData.ServicesNs)
		if err != nil || opreg == nil {
			// klog.Warning("OperandRegistry common-service is not ready, retry in 5 seconds")
			time.Sleep(5 * time.Second)
			continue
		}

		for i := range opreg.Spec.Operators {
			operatorsName = append(operatorsName, opreg.Spec.Operators[i].Name)
		}

		for _, name := range operatorsName {
			var opt apiv3.BedrockOperator
			var err error

			opt, err = getBedrockOperator(bs, name, bs.CSData.CPFSNs, instance)

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

		if err := bs.Client.Status().Update(ctx, instance); err != nil {
			klog.Warning(err)
		}

		time.Sleep(2 * time.Minute)
	}
}

func getBedrockOperator(bs *bootstrap.Bootstrap, name, namespace string, reference *apiv3.CommonService) (apiv3.BedrockOperator, error) {
	var opt apiv3.BedrockOperator
	opt.Name = name

	// fetch subscription
	sub := &olmv1alpha1.Subscription{}
	subKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	if err := bs.Client.Get(ctx, subKey, sub); err != nil {
		return opt, err
	}
	installedCSV := sub.Status.InstalledCSV
	if installedCSV != "" {
		opt.Name = installedCSV[:strings.IndexByte(installedCSV, '.')]
		opt.Version = installedCSV[strings.IndexByte(installedCSV, '.')+1:]
	}

	// fetch csv
	csv := &olmv1alpha1.ClusterServiceVersion{}
	csvKey := types.NamespacedName{
		Name:      installedCSV,
		Namespace: namespace,
	}

	if err := bs.Reader.Get(ctx, csvKey, csv); err != nil {
		if installedCSV == "" {
			klog.Warningf("Failed to get %s CSV: installedCSV is not found. Please check Subscription", name)
		} else {
			klog.Warningf("Failed to get %s CSV: %s", name, err)
		}
	} else {
		if len(csv.Status.Conditions) > 0 {
			csvStatus := csv.Status.Conditions[len(csv.Status.Conditions)-1].Phase
			opt.OperatorStatus = fmt.Sprintf("%v", csvStatus)
		} else {
			opt.OperatorStatus = "NotReady"
		}
	}

	// fetch installplanName
	installplanName := ""
	if sub.Status.Install != nil {
		installplanName = sub.Status.Install.Name
	}
	opt.InstallPlanName = installplanName

	// determinate subscription status
	if installplanName == "" {
		opt.SubscriptionStatus = "Failed"
		opt.InstallPlanName = "Not Found"
	} else {
		currentCSV := sub.Status.CurrentCSV
		if installedCSV == currentCSV {
			opt.SubscriptionStatus = "Succeeded"
		} else {
			opt.SubscriptionStatus = fmt.Sprintf("%v", sub.Status.State)
		}
	}

	if opt.OperatorStatus == "" || opt.OperatorStatus != "Succeeded" || opt.SubscriptionStatus == "" || opt.SubscriptionStatus != "Succeeded" {
		opt.Troubleshooting = "Operator status is not healthy, please check " + constant.GeneralTroubleshooting + " for more information"
	}

	if opt.SubscriptionStatus == "" || opt.SubscriptionStatus != "Succeeded" {
		bs.EventRecorder.Eventf(reference, "Warning", "Bedrock Operator Failed", "Subscription %s/%s is not healthy, please check troubleshooting document %s for reasons and solutions", name, installedCSV, constant.GeneralTroubleshooting)
	} else if opt.OperatorStatus == "" || opt.OperatorStatus != "Succeeded" {
		bs.EventRecorder.Eventf(reference, "Warning", "Bedrock Operator Failed", "ClusterServiceVersion %s/%s is not healthy, please check troubleshooting document %s for reasons and solutions", namespace, installedCSV, constant.GeneralTroubleshooting)
	}

	return opt, nil
}
