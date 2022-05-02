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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/controllers/common"
)

type Resource struct {
	name    string
	version string
	group   string
	kind    string
}

var deprecatedServicesMap = map[string][]*Resource{
	"ibm-monitoring-exporters-operator": {
		{
			name:    "ibm-monitoring",
			version: "v1alpha1",
			group:   "monitoring.operator.ibm.com",
			kind:    "Exporter",
		},
		{
			name:    "monitoring-exporters-operator-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
		},
	},
	"ibm-monitoring-prometheusext-operator": {
		{
			name:    "ibm-monitoring",
			version: "v1alpha1",
			group:   "monitoring.operator.ibm.com",
			kind:    "PrometheusExt",
		},
		{
			name:    "monitoring-prometheus-ext-operator-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
		},
	},
	"ibm-metering-operator": {
		{
			name:    "metering",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "Metering",
		},
		{
			name:    "meteringui",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "MeteringUI",
		},
		{
			name:    "meteringreportserver",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "MeteringReportServer",
		},
		{
			name:    "ibm-metering-bindinfo",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandBindInfo",
		},
		{
			name:    "ibm-metering-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
		},
	},
	"ibm-elastic-stack-operator": {
		{
			name:    "logging",
			version: "v1alpha1",
			group:   "elasticstack.ibm.com",
			kind:    "ElasticStack",
		},
		{
			name:    "ibm-elastic-stack-bindinfo",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandBindInfo",
		},
		{
			name:    "ibm-elastic-stack-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
		},
	},
	"ibm-catalog-ui-operator": {
		{
			name:    "catalog-ui",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "CatalogUI",
		},
		{
			name:    "catalog-ui-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
		},
	},
	"ibm-helm-api-operator": {
		{
			name:    "helm-api",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "HelmAPI",
		},
		{
			name:    "helm-api-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
		},
	},
	"ibm-helm-repo-operator": {
		{
			name:    "helm-repo",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "HelmRepo",
		},
		{
			name:    "helm-repo-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
		},
	},
}

// CleanUpDeprecatedServices will clean up deprecated services' CRD, operandBindInfo, operandRequest, subscription, CSV
func CleanUpDeprecatedServices(bs *bootstrap.Bootstrap) {
	for {
		for service, resourcesList := range deprecatedServicesMap {
			for _, resource := range resourcesList {
				operatorNs, err := util.GetOperatorNamespace()
				if err != nil {
					continue
				}

				if err := cleanup(bs, resource.name, operatorNs, resource.version, resource.group, resource.kind); err != nil {
					continue
				}
			}

			// delete sub & csv
			if err := deleteSubscription(bs, service, MasterNamespace); err != nil {
				klog.Errorf("Delete subscription failed: %v", err)
				continue
			}
		}

		time.Sleep(2 * time.Minute)
	}
}

func cleanup(bs *bootstrap.Bootstrap, name, operatorNs string, version, group, kind string) error {
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Version: version, Kind: kind})
	resource.SetName(name)
	resource.SetNamespace(operatorNs)
	if err := bs.Client.Delete(context.TODO(), resource); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	klog.Infof("Deleting resource %s/%s", operatorNs, name)
	return nil
}

func deleteSubscription(bs *bootstrap.Bootstrap, name, namespace string) error {
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
