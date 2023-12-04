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

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
	"github.com/IBM/ibm-common-service-operator/controllers/constant"
)

const (
	namespaceScope = "namespaceScope"
	clusterScope   = "clusterScope"
)

var deprecatedServicesMap = map[string][]*bootstrap.Resource{
	"ibm-monitoring-exporters-operator": {
		{
			Name:    "ibm-monitoring",
			Version: "v1alpha1",
			Group:   "monitoring.operator.ibm.com",
			Kind:    "Exporter",
			Scope:   namespaceScope,
		},
		{
			Name:    "monitoring-exporters-operator-request",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandRequest",
			Scope:   namespaceScope,
		},
	},
	"ibm-monitoring-prometheusext-operator": {
		{
			Name:    "ibm-monitoring",
			Version: "v1alpha1",
			Group:   "monitoring.operator.ibm.com",
			Kind:    "PrometheusExt",
			Scope:   namespaceScope,
		},
		{
			Name:    "monitoring-prometheus-ext-operator-request",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandRequest",
			Scope:   namespaceScope,
		},
	},
	"ibm-metering-operator": {
		{
			Name:    "metering",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "Metering",
			Scope:   namespaceScope,
		},
		{
			Name:    "meteringui",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "MeteringUI",
			Scope:   namespaceScope,
		},
		{
			Name:    "meteringreportserver",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "MeteringReportServer",
			Scope:   clusterScope,
		},
		{
			Name:    "ibm-metering-bindinfo",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandBindInfo",
			Scope:   namespaceScope,
		},
		{
			Name:    "ibm-metering-request",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandRequest",
			Scope:   namespaceScope,
		},
	},
	"ibm-elastic-stack-operator": {
		{
			Name:    "logging",
			Version: "v1alpha1",
			Group:   "elasticstack.ibm.com",
			Kind:    "ElasticStack",
			Scope:   namespaceScope,
		},
		{
			Name:    "ibm-elastic-stack-bindinfo",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandBindInfo",
			Scope:   namespaceScope,
		},
		{
			Name:    "ibm-elastic-stack-request",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandRequest",
			Scope:   namespaceScope,
		},
	},
	"ibm-catalog-ui-operator": {
		{
			Name:    "catalog-ui",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "CatalogUI",
			Scope:   namespaceScope,
		},
		{
			Name:    "catalog-ui-request",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandRequest",
			Scope:   namespaceScope,
		},
	},
	"ibm-helm-api-operator": {
		{
			Name:    "helm-api",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "HelmAPI",
			Scope:   namespaceScope,
		},
		{
			Name:    "helm-api-request",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandRequest",
			Scope:   namespaceScope,
		},
	},
	"ibm-helm-repo-operator": {
		{
			Name:    "helm-repo",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "HelmRepo",
			Scope:   namespaceScope,
		},
		{
			Name:    "helm-repo-request",
			Version: "v1alpha1",
			Group:   "operator.ibm.com",
			Kind:    "OperandRequest",
			Scope:   namespaceScope,
		},
	},
}

// CleanUpDeprecatedServices will clean up deprecated services' CRD, operandBindInfo, operandRequest, subscription, CSV
func CleanUpDeprecatedServices(bs *bootstrap.Bootstrap) {
	for {
		opreg := bs.GetOperandRegistry(ctx, constant.MasterCR, bs.CSData.MasterNs)
		if opreg != nil {
			if opreg.GetAnnotations() != nil && opreg.GetAnnotations()["version"] == bs.CSData.Version {
				for service, resourcesList := range deprecatedServicesMap {
					getResourceFailed := false
					for _, resource := range resourcesList {
						operatorNs, err := util.GetOperatorNamespace()
						if err != nil {
							getResourceFailed = true
							continue
						}

						if err := bs.Cleanup(operatorNs, resource); err != nil {
							getResourceFailed = true
							continue
						}
					}

					// delete sub & csv
					if !getResourceFailed {
						if err := DeleteOperator(bs, service, MasterNamespace); err != nil {
							klog.Errorf("Delete subscription failed: %v", err)
							continue
						}
					}
				}
			} else {
				klog.Info("Skipped cleaning deprecated services, wait for latest OperandRegistry common-service ready, retry in 2 minutes.")
			}
		}
		time.Sleep(2 * time.Minute)
	}
}

func DeleteOperator(bs *bootstrap.Bootstrap, name, namespace string) error {
	key := types.NamespacedName{Name: name, Namespace: namespace}
	sub := &olmv1alpha1.Subscription{}
	if err := bs.Reader.Get(context.TODO(), key, sub); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		klog.Errorf("Failed to get subscription %s/%s", namespace, name)
		return client.IgnoreNotFound(err)
	}

	klog.Infof("Deleting subscription %s/%s", namespace, name)

	// Delete csv
	csvName := sub.Status.InstalledCSV
	if csvName != "" {
		csv := &olmv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      csvName,
				Namespace: namespace,
			},
		}
		if err := bs.Client.Delete(context.TODO(), csv); err != nil && !errors.IsNotFound(err) {
			klog.Errorf("Failed to delete Cluster Service Version: %v", err)
			return err
		}
	}

	// Delete subscription
	if err := bs.Client.Delete(context.TODO(), sub); err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Failed to delete subscription: %s", err)
		return err
	}

	return nil
}
