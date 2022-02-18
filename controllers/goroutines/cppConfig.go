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
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
	collector "github.com/IBM/ibm-common-service-operator/controllers/configurationCollector"
)

// CreateUpdateConfig deploys config builder for global cpp configmap
func CreateUpdateConfig(bs *bootstrap.Bootstrap) {

	for {
		config := &corev1.ConfigMap{}
		if err := bs.Client.Get(context.TODO(), types.NamespacedName{Name: "ibm-cpp-config", Namespace: bs.CSData.MasterNs}, config); err != nil && !errors.IsNotFound(err) {
			continue
		} else if errors.IsNotFound(err) {
			config.ObjectMeta.Name = "ibm-cpp-config"
			config.ObjectMeta.Namespace = bs.CSData.MasterNs
			config.Data = make(map[string]string)
			config.Data = collector.Buildconfig(config.Data, bs)
			if err := bs.Client.Create(context.TODO(), config); err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			klog.Infof("Global CPP config %s/%s is created", "ibm-cpp-config", bs.CSData.MasterNs)
		} else {
			orgConfig := config.DeepCopy()
			config.Data = collector.Buildconfig(config.Data, bs)
			if !reflect.DeepEqual(orgConfig, config) {
				if err := bs.Client.Update(context.TODO(), config); err != nil {
					time.Sleep(1 * time.Second)
					continue
				}
				klog.Infof("Global CPP config %s/%s is updated", "ibm-cpp-config", bs.CSData.MasterNs)
			}

		}
		time.Sleep(10 * time.Minute)
	}

}
