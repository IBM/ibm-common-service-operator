//
// Copyright 2024 IBM Corporation
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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	v3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	util "github.com/IBM/ibm-common-service-operator/v4/controllers/common"
	"github.com/IBM/ibm-common-service-operator/v4/controllers/constant"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/v4/api/v1alpha1"
)

func (r *CommonServiceReconciler) updateOperatorConfig(ctx context.Context, configList []v3.OperatorConfig) (bool, error) {
	klog.Info("Applying OperatorConfig")

	if configList == nil {
		if err := r.clearAllUserManaged(ctx); err != nil {
			return false, err
		}
		return true, nil
	}

	// TODO: remove when this feature is generalized to all other operators
	for _, c := range configList {
		config := c
		packageName, err := r.fetchPackageNameFromOpReg(ctx, config.Name)
		if err != nil {
			return false, err
		}
		if packageName != "cloud-native-postgresql" {
			return false, errors.New("failed to update OperatorConfig. This feature is only available for cloud-native-postgresql operator")
		}
		if err := r.updateUserManaged(ctx, config.Name, config.UserManaged); err != nil {
			return false, err
		}
		if config.Replicas == nil {
			return true, nil
		}
	}

	operatorConfig := &odlm.OperatorConfig{}
	if err := r.Reader.Get(ctx, types.NamespacedName{
		Name:      "test-operator-config",
		Namespace: r.Bootstrap.CSData.ServicesNs,
	}, operatorConfig); err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("failed to get OperatorConfig %s/%s: %v", operatorConfig.GetNamespace(), operatorConfig.GetName(), err)
			return true, err
		}
	}
	replicas := *configList[0].Replicas
	replacer := strings.NewReplacer("placeholder-size", fmt.Sprintf("%d", replicas))
	updatedConfig := replacer.Replace(constant.PostGresOperatorConfig)
	klog.V(2).Infof("OperatorConfig to be applied will be: %v", updatedConfig)

	if err := r.Bootstrap.InstallOrUpdateOperatorConfig(updatedConfig, true); err != nil {
		return false, err
	}
	return false, nil
}

func (r *CommonServiceReconciler) fetchPackageNameFromOpReg(ctx context.Context, name string) (string, error) {
	registry, err := r.GetOperandRegistry(ctx, "common-service", r.CSData.ServicesNs)
	if err != nil {
		return "", err
	}

	for _, r := range registry.Spec.Operators {
		operator := r
		if operator.Name == name {
			return operator.PackageName, nil
		}
	}
	return "", nil
}

func (r *CommonServiceReconciler) updateUserManaged(ctx context.Context, operatorName string, value bool) error {
	opreg := &odlm.OperandRegistry{}
	if err := r.Reader.Get(ctx, types.NamespacedName{Namespace: util.GetServicesNamespace(r.Reader), Name: "common-service"}, opreg); err != nil {
		return err
	}
	if err := util.UpdateOpRegUserManaged(opreg, operatorName, value); err != nil {
		return err
	}
	if err := r.Client.Update(ctx, opreg); err != nil {
		return err
	}
	return nil
}

func (r *CommonServiceReconciler) clearAllUserManaged(ctx context.Context) error {
	opreg := &odlm.OperandRegistry{}
	if err := r.Reader.Get(ctx, types.NamespacedName{Namespace: util.GetServicesNamespace(r.Reader), Name: "common-service"}, opreg); err != nil {
		return err
	}
	for i := range opreg.Spec.Operators {
		i := i
		opreg.Spec.Operators[i].UserManaged = false
	}
	return r.Client.Update(ctx, opreg)
}
