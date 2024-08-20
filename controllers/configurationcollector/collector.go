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

package configurationcollector

import (
	"context"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	"github.com/IBM/ibm-common-service-operator/v4/controllers/bootstrap"
	util "github.com/IBM/ibm-common-service-operator/v4/controllers/common"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/constant"
)

func Buildconfig(config map[string]string, bs *bootstrap.Bootstrap) map[string]string {
	builder := configbuilder{data: config, bs: bs}
	updatedConfig := builder.setDefaultStorageClass()
	return updatedConfig.data
}

type configbuilder struct {
	data map[string]string
	bs   *bootstrap.Bootstrap
}

func (b *configbuilder) setDefaultStorageClass() *configbuilder {
	scList := &storagev1.StorageClassList{}
	err := b.bs.Reader.List(context.TODO(), scList)
	if err != nil {
		return b
	}
	if len(scList.Items) == 0 {
		return b
	}

	var defaultSC string
	var defaultSCList []string
	var allSCList []string

	for _, sc := range scList.Items {
		if defaultSC == "" {
			defaultSC = sc.Name
		}
		if sc.ObjectMeta.GetAnnotations()["storageclass.kubernetes.io/is-default-class"] == "true" {
			defaultSCList = append(defaultSCList, sc.Name)
		}
		if sc.Provisioner == "kubernetes.io/no-provisioner" {
			continue
		}
		allSCList = append(allSCList, sc.GetName())
	}

	if b.data == nil {
		b.data = make(map[string]string)
	}
	if defaultSC != "" {
		b.data["storageclass.default"] = defaultSC
	}

	if len(defaultSCList) != 1 {
		b.data["storageclass.default.list"] = strings.Join(defaultSCList, ",")
	}

	if len(allSCList) != 0 {
		b.data["storageclass.list"] = strings.Join(allSCList, ",")
	}

	return b
}

// CreateUpdateConfig deploys config builder for global cpp configmap
func CreateUpdateConfig(bs *bootstrap.Bootstrap) error {
	config := &corev1.ConfigMap{}
	if err := bs.Reader.Get(context.TODO(), types.NamespacedName{Name: constant.IBMCPPCONFIG, Namespace: bs.CSData.ServicesNs}, config); err != nil && !errors.IsNotFound(err) {
		klog.Errorf("Failed to get ConfigMap %s/%s: %v", bs.CSData.ServicesNs, constant.IBMCPPCONFIG, err)
		return err
	} else if errors.IsNotFound(err) {
		config.ObjectMeta.Name = constant.IBMCPPCONFIG
		config.ObjectMeta.Namespace = bs.CSData.ServicesNs
		config.Data = make(map[string]string)
		config.Data = Buildconfig(config.Data, bs)
		if !(config.Labels != nil && config.Labels[constant.CsManagedLabel] == "true") {
			util.EnsureLabelsForConfigMap(config, map[string]string{
				constant.CsManagedLabel: "true",
			})
		}
		if err := bs.Client.Create(context.TODO(), config); err != nil {
			klog.Errorf("Failed to create ConfigMap %s/%s: %v", bs.CSData.ServicesNs, constant.IBMCPPCONFIG, err)
			return err
		}
		klog.Infof("Global CPP config %s/%s is created", bs.CSData.ServicesNs, constant.IBMCPPCONFIG)
	} else {
		orgConfig := config.DeepCopy()
		config.Data = Buildconfig(config.Data, bs)
		if !(config.Labels != nil && config.Labels[constant.CsManagedLabel] == "true") {
			util.EnsureLabelsForConfigMap(config, map[string]string{
				constant.CsManagedLabel: "true",
			})
		}
		if !reflect.DeepEqual(orgConfig, config) {
			if err := bs.Client.Update(context.TODO(), config); err != nil {
				klog.Errorf("Failed to update ConfigMap %s/%s: %v", bs.CSData.ServicesNs, constant.IBMCPPCONFIG, err)
				return err
			}
			klog.Infof("Global CPP config %s/%s is updated", bs.CSData.ServicesNs, constant.IBMCPPCONFIG)
		}
	}

	if err := bs.PropagateCPPConfig(config); err != nil {
		klog.Error(err)
		return err
	}
	return nil
}
