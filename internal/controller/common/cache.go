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

package common

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	"github.com/IBM/ibm-common-service-operator/v4/internal/controller/constant"
)

// NewCSCache implements a customized cache with a for CS
func NewCSCache(watchNamespaceList []string, opts ctrl.Options) ctrl.Options {
	configmapSelector := labels.Set{constant.CsManagedLabel: "true"}
	secretSelector := labels.Set{constant.SecretWatchLabel: "true"}
	// set DefaultNamespaces based on watchNamespaces
	// if watchNamespaces is empty, then cache resource in all namespaces
	var cacheDefaultNamespaces map[string]cache.Config
	if len(watchNamespaceList) == 1 && watchNamespaceList[0] == "" {
		// cache resource in all namespaces
		cacheDefaultNamespaces = nil
	} else {
		// cache resource in watchNamespaces, but only for namespaced resources
		cacheNamespaces := make(map[string]cache.Config)
		for _, ns := range watchNamespaceList {
			cacheNamespaces[ns] = cache.Config{}
		}
		cacheDefaultNamespaces = cacheNamespaces
	}

	cacheByObject := map[client.Object]cache.ByObject{
		&corev1.Secret{}:       {Label: secretSelector.AsSelector()},
		&corev1.ConfigMap{}:    {Label: configmapSelector.AsSelector()},
		&apiv3.CommonService{}: {},
	}

	// set cache options
	opts.Cache = cache.Options{
		ByObject:          cacheByObject,
		DefaultNamespaces: cacheDefaultNamespaces,
		Scheme:            opts.Scheme,
	}

	return opts

}
