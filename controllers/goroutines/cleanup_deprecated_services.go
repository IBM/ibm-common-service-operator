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

const (
	namespaceScope = "namespaceScope"
	clusterScope   = "clusterScope"
)

type Resource struct {
	name    string
	version string
	group   string
	kind    string
	scope   string
}

var deprecatedServicesMap = map[string][]*Resource{
	"ibm-monitoring-exporters-operator": {
		{
			name:    "ibm-monitoring",
			version: "v1alpha1",
			group:   "monitoring.operator.ibm.com",
			kind:    "Exporter",
			scope:   namespaceScope,
		},
		{
			name:    "monitoring-exporters-operator-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
			scope:   namespaceScope,
		},
	},
	"ibm-monitoring-prometheusext-operator": {
		{
			name:    "ibm-monitoring",
			version: "v1alpha1",
			group:   "monitoring.operator.ibm.com",
			kind:    "PrometheusExt",
			scope:   namespaceScope,
		},
		{
			name:    "monitoring-prometheus-ext-operator-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
			scope:   namespaceScope,
		},
	},
	"ibm-metering-operator": {
		{
			name:    "metering",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "Metering",
			scope:   namespaceScope,
		},
		{
			name:    "meteringui",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "MeteringUI",
			scope:   namespaceScope,
		},
		{
			name:    "meteringreportserver",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "MeteringReportServer",
			scope:   clusterScope,
		},
		{
			name:    "ibm-metering-bindinfo",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandBindInfo",
			scope:   namespaceScope,
		},
		{
			name:    "ibm-metering-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
			scope:   namespaceScope,
		},
	},
	"ibm-elastic-stack-operator": {
		{
			name:    "logging",
			version: "v1alpha1",
			group:   "elasticstack.ibm.com",
			kind:    "ElasticStack",
			scope:   namespaceScope,
		},
		{
			name:    "ibm-elastic-stack-bindinfo",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandBindInfo",
			scope:   namespaceScope,
		},
		{
			name:    "ibm-elastic-stack-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
			scope:   namespaceScope,
		},
	},
	"ibm-catalog-ui-operator": {
		{
			name:    "catalog-ui",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "CatalogUI",
			scope:   namespaceScope,
		},
		{
			name:    "catalog-ui-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
			scope:   namespaceScope,
		},
	},
	"ibm-helm-api-operator": {
		{
			name:    "helm-api",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "HelmAPI",
			scope:   namespaceScope,
		},
		{
			name:    "helm-api-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
			scope:   namespaceScope,
		},
	},
	"ibm-helm-repo-operator": {
		{
			name:    "helm-repo",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "HelmRepo",
			scope:   namespaceScope,
		},
		{
			name:    "helm-repo-request",
			version: "v1alpha1",
			group:   "operator.ibm.com",
			kind:    "OperandRequest",
			scope:   namespaceScope,
		},
	},
}

// CleanUpDeprecatedServices will clean up deprecated services' CRD, operandBindInfo, operandRequest, subscription, CSV
func CleanUpDeprecatedServices(bs *bootstrap.Bootstrap) {
	for {
		opreg := bs.GetOperandRegistry(ctx, "common-service", bs.CSData.MasterNs)
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

						if err := cleanup(bs, operatorNs, resource); err != nil {
							getResourceFailed = true
							continue
						}
					}

					// delete sub & csv
					if !getResourceFailed {
						if err := deleteSubscription(bs, service, MasterNamespace); err != nil {
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

func cleanup(bs *bootstrap.Bootstrap, operatorNs string, resource *Resource) error {
	deprecated := &unstructured.Unstructured{}
	deprecated.SetGroupVersionKind(schema.GroupVersionKind{Group: resource.group, Version: resource.version, Kind: resource.kind})
	deprecated.SetName(resource.name)
	if resource.scope == namespaceScope {
		deprecated.SetNamespace(operatorNs)
	}
	if err := bs.Client.Delete(context.TODO(), deprecated); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	klog.Infof("Deleting resource %s/%s", operatorNs, resource.name)
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
