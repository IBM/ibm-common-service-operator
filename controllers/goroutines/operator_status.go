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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	apiv3 "github.com/IBM/ibm-common-service-operator/api/v3"
	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
)

// UpdateCsCrStatus will update cs cr status according to each bedrock operator
func UpdateCsCrStatus(bs *bootstrap.Bootstrap) {
	for {
		instance := &apiv3.CommonService{}
		if err := bs.Client.Get(ctx, types.NamespacedName{Name: "common-service", Namespace: MasterNamespace}, instance); err != nil {
			continue
		}

		if instance == nil {
			continue
		}

		var operatorSlice []apiv3.BedrockOperator
		operatorsName := []string{
			"ibm-auditlogging-operator",
			"ibm-cert-manager-operator",
			"ibm-commonui-operator",
			"ibm-crossplane-operator-app",
			"ibm-events-operator",
			"ibm-healthcheck-operator",
			"ibm-iam-operator",
			"ibm-ingress-nginx-operator",
			"ibm-licensing-operator",
			"ibm-management-ingress-operator",
			"ibm-mongodb-operator",
			"ibm-monitoring-grafana-operator",
			"ibm-namespace-scope-operator",
			"ibm-platform-api-operator",
			"ibm-zen-operator",
			"operand-deployment-lifecycle-manager-app"}

		for _, name := range operatorsName {
			var opt apiv3.BedrockOperator
			if bs.MultiInstancesEnable && (name == "ibm-cert-manager-operator" || name == "ibm-licensing-operator") {
				opt = getBedrockOperator(bs, name, bs.CSData.ControlNs)
			} else {
				opt = getBedrockOperator(bs, name, bs.CSData.MasterNs)
			}

			if opt.Version != "" && opt.Status != "" {
				operatorSlice = append(operatorSlice, opt)
			}
		}

		instance.Status.BedrockOperators = operatorSlice
		if err := bs.Client.Status().Update(ctx, instance); err != nil {
			klog.Warning(err)
		}

		time.Sleep(2 * time.Minute)
	}
}

func getBedrockOperator(bs *bootstrap.Bootstrap, name, namespace string) apiv3.BedrockOperator {
	var opt apiv3.BedrockOperator

	// fetch subscription
	sub := &olmv1alpha1.Subscription{}
	subKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	if err := bs.Client.Get(ctx, subKey, sub); err != nil {
		return opt
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
		return opt
	}

	// fetch installplan
	installplanName := sub.Status.Install.Name
	if installplanName == "" {
		opt.SubscriptionStatus = "Failed"
	}
	opt.InstallPlanName = installplanName

	// determinate subscription status
	currentCSV := sub.Status.CurrentCSV
	installedIsLarger := util.CompareVersion(installedCSV, currentCSV)
	currentIsLarger := util.CompareVersion(currentCSV, installedCSV)
	if !installedIsLarger && !currentIsLarger {
		// installedCSV == currentCSV
		opt.SubscriptionStatus = "Succeeded"
	} else if currentIsLarger {
		opt.SubscriptionStatus = "Upgrade available"
	} else {
		klog.Warning("installedCSV version is larger than currentCSV")
	}

	if len(csv.Status.Conditions) > 0 {
		csvStatus := csv.Status.Conditions[len(csv.Status.Conditions)-1].Phase
		opt.Status = fmt.Sprintf("%v", csvStatus)
	}

	return opt
}
