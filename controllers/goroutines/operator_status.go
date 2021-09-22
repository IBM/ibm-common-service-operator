//
// Copyright 2021 IBM Corporation
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
	"fmt"
	"strings"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
)

// UpdateCsCrStatus will update cs cr status according to each bedrock operator
func UpdateCsCrStatus(bs *bootstrap.Bootstrap) {
	for {
		instance := &apiv3.CommonService{}
		if err := bs.Reader.Get(ctx, types.NamespacedName{Name: "common-service", Namespace: MasterNamespace}, instance); err != nil {
			klog.Warningf("Getting Common-service CR with error: %s", err)
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

		opreg := bs.GetOperandRegistry(ctx, "common-service", bs.CSData.MasterNs)
		if opreg == nil {
			klog.Warning("OperandRegistry common-service is not ready, retry in 5 seconds")
			time.Sleep(5 * time.Second)
			continue
		}

		for i := range opreg.Spec.Operators {
			operatorsName = append(operatorsName, opreg.Spec.Operators[i].Name)
		}
		operatorsName = append(operatorsName, []string{
			"ibmcloud-operator",
			"ibm-crossplane-operator-app",
			"ibm-namespace-scope-operator",
			"operand-deployment-lifecycle-manager-app"}...)

		for _, name := range operatorsName {
			var opt apiv3.BedrockOperator
			var err error

			if bs.MultiInstancesEnable && (name == "ibm-cert-manager-operator" || name == "ibm-licensing-operator") {
				opt, err = getBedrockOperator(bs, name, bs.CSData.ControlNs)
			} else {
				opt, err = getBedrockOperator(bs, name, bs.CSData.MasterNs)
			}

			if err == nil {
				operatorSlice = append(operatorSlice, opt)
			} else if !errors.IsNotFound(err) {
				klog.Errorf("Failed to check operator %s: %v", name, err)
			}
		}

		instance.Status.BedrockOperators = operatorSlice
		if err := bs.Client.Status().Update(ctx, instance); err != nil {
			klog.Warning(err)
		}

		time.Sleep(2 * time.Minute)
	}
}

func getBedrockOperator(bs *bootstrap.Bootstrap, name, namespace string) (apiv3.BedrockOperator, error) {
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
		klog.Warningf("Failed to get %s CSV: %s", name, err)
	} else {
		if len(csv.Status.Conditions) > 0 {
			csvStatus := csv.Status.Conditions[len(csv.Status.Conditions)-1].Phase
			opt.OperatorStatus = fmt.Sprintf("%v", csvStatus)
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

	return opt, nil
}
