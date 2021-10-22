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

package configurationcollector

import (
	"context"
	"strings"

	storagev1 "k8s.io/api/storage/v1"

	"github.com/IBM/ibm-common-service-operator/controllers/bootstrap"
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
