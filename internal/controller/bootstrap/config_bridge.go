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

package bootstrap

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	apiv3 "github.com/IBM/ibm-common-service-operator/v4/api/v3"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/klog"
)

// ConfigMergerFunc is a function type that merges base config with CommonService configs
// This allows bootstrap to use controller merge logic without import cycles
type ConfigMergerFunc func(baseConfig string, cs *apiv3.CommonService, servicesNs string) (string, error)

// SetConfigMerger sets the configuration merger function on the Bootstrap instance.
// Called during reconciler initialization to inject the merge logic.
func (b *Bootstrap) SetConfigMerger(merger ConfigMergerFunc) {
	b.configMerger = merger
}

// mergeConfigs calls the injected merger function if available
// It also determines the largest size from ALL CommonService CRs and overrides the current CR's size
func (b *Bootstrap) mergeConfigs(baseConfig string, cs *apiv3.CommonService) (string, error) {
	ctx := context.Background()

	// First, determine the largest size from all CommonService CRs
	largestSize, err := b.getLargestSizeFromAllCRs(ctx)
	if err != nil {
		return "", err
	}

	// Override the current CR's size with the largest size if found
	originalSize := cs.Spec.Size
	if largestSize != "" {
		cs.Spec.Size = largestSize
		klog.Infof("Bootstrap: Overriding CommonService CR %s/%s size from '%s' to largest size '%s'", cs.Namespace, cs.Name, originalSize, largestSize)
	}

	if b.configMerger != nil {
		return b.configMerger(baseConfig, cs, b.CSData.ServicesNs)
	}
	// If no merger is set, return base config unchanged
	return baseConfig, nil
}

// getLargestSizeFromAllCRs determines the largest size across all CommonService CRs
func (b *Bootstrap) getLargestSizeFromAllCRs(ctx context.Context) (string, error) {
	csObjectList := &apiv3.CommonServiceList{}
	if err := b.Client.List(ctx, csObjectList); err != nil {
		return "", err
	}

	sizePriority := map[string]int{
		"starterset": 0,
		"starter":    0,
		"small":      1,
		"medium":     2,
		"large":      3,
	}

	largestSize := ""
	largestPriority := -1

	for _, cs := range csObjectList.Items {
		if cs.GetDeletionTimestamp() != nil {
			continue
		}

		csSize := cs.Spec.Size
		if priority, ok := sizePriority[csSize]; ok {
			klog.Infof("Bootstrap: CommonService CR %s/%s has size: %s (priority: %d)", cs.Namespace, cs.Name, csSize, priority)
			if priority > largestPriority {
				largestPriority = priority
				largestSize = csSize
				klog.Infof("Bootstrap: New largest size found: %s (priority: %d)", largestSize, largestPriority)
			}
		} else if csSize != "" {
			klog.Infof("Bootstrap: CommonService CR %s/%s has custom size configuration (not a predefined size)", cs.Namespace, cs.Name)
		}
	}

	if largestSize == "" {
		klog.Info("Bootstrap: No predefined size found in any CommonService CR, will use starterset")
		return "starterset", nil
	}

	klog.Infof("Bootstrap: FINAL DECISION - Largest size across all CommonService CRs is: %s (priority: %d)", largestSize, largestPriority)
	return largestSize, nil
}

// MergeAllCommonServiceCRs merges all CommonService CRs with conflict resolution
// Returns: merged CR, list of conflicts, list of source CRs, error
func (b *Bootstrap) MergeAllCommonServiceCRs(ctx context.Context, currentCR *apiv3.CommonService) (*apiv3.CommonService, []apiv3.MergeConflict, []apiv3.SourceCR, error) {
	// List all CommonService CRs
	csObjectList := &apiv3.CommonServiceList{}
	if err := b.Client.List(ctx, csObjectList); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to list CommonService CRs: %v", err)
	}

	// Filter out deleted CRs
	var activeCRs []apiv3.CommonService
	for _, cs := range csObjectList.Items {
		if cs.GetDeletionTimestamp() == nil {
			activeCRs = append(activeCRs, cs)
		}
	}

	if len(activeCRs) == 0 {
		klog.Info("No active CommonService CRs found")
		return currentCR.DeepCopy(), nil, nil, nil
	}

	if len(activeCRs) == 1 {
		klog.Infof("Only one CommonService CR found: %s/%s", activeCRs[0].Namespace, activeCRs[0].Name)
		sourceCRs := []apiv3.SourceCR{{
			Namespace: activeCRs[0].Namespace,
			Name:      activeCRs[0].Name,
			Priority:  b.getCRPriority(activeCRs[0]),
		}}
		return activeCRs[0].DeepCopy(), nil, sourceCRs, nil
	}

	// Sort CRs by priority
	sortedCRs := b.sortCRsByPriority(activeCRs)

	// Build source CR list
	sourceCRs := make([]apiv3.SourceCR, len(sortedCRs))
	for i, cr := range sortedCRs {
		sourceCRs[i] = apiv3.SourceCR{
			Namespace: cr.Namespace,
			Name:      cr.Name,
			Priority:  b.getCRPriority(cr),
		}
	}

	klog.Infof("Merging %d CommonService CRs in priority order", len(sortedCRs))
	for i, cr := range sortedCRs {
		klog.Infof("  %d. %s/%s (priority: %d)", i+1, cr.Namespace, cr.Name, b.getCRPriority(cr))
	}

	// Start with the highest priority CR as base
	mergedCR := sortedCRs[0].DeepCopy()
	var conflicts []apiv3.MergeConflict

	// Merge each subsequent CR
	for i := 1; i < len(sortedCRs); i++ {
		cr := sortedCRs[i]
		crName := fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)
		baseName := fmt.Sprintf("%s/%s", mergedCR.Namespace, mergedCR.Name)

		klog.Infof("Merging CR %s into base %s", crName, baseName)

		// Merge each field with conflict resolution
		b.mergeSize(mergedCR, &cr, baseName, crName, &conflicts)
		b.mergeInstallPlanApproval(mergedCR, &cr, baseName, crName, &conflicts)
		b.mergeBooleans(mergedCR, &cr, baseName, crName, &conflicts)
		b.mergeStrings(mergedCR, &cr, baseName, crName, &conflicts)
		b.mergeLabels(mergedCR, &cr, baseName, crName, &conflicts)
		b.mergeFeatures(mergedCR, &cr, baseName, crName, &conflicts)
		b.mergeHugePagesField(mergedCR, &cr, baseName, crName, &conflicts)
		b.mergeOperatorConfigsField(mergedCR, &cr, baseName, crName, &conflicts)
		b.mergeServicesField(mergedCR, &cr, baseName, crName, &conflicts)
	}

	klog.Infof("Merge complete. Total conflicts: %d", len(conflicts))
	return mergedCR, conflicts, sourceCRs, nil
}

// sortCRsByPriority sorts CRs by priority (master > oldest > alphabetical)
func (b *Bootstrap) sortCRsByPriority(crs []apiv3.CommonService) []apiv3.CommonService {
	sorted := make([]apiv3.CommonService, len(crs))
	copy(sorted, crs)

	sort.Slice(sorted, func(i, j int) bool {
		priI := b.getCRPriority(sorted[i])
		priJ := b.getCRPriority(sorted[j])

		if priI != priJ {
			return priI > priJ // Higher priority first
		}

		// Same priority, sort by creation time (older first)
		timeI := sorted[i].CreationTimestamp.Time
		timeJ := sorted[j].CreationTimestamp.Time
		if !timeI.Equal(timeJ) {
			return timeI.Before(timeJ)
		}

		// Same time, sort alphabetically by namespace/name
		nameI := sorted[i].Namespace + "/" + sorted[i].Name
		nameJ := sorted[j].Namespace + "/" + sorted[j].Name
		return nameI < nameJ
	})

	return sorted
}

// getCRPriority returns priority for a CR (master CR = 100, others = 0)
func (b *Bootstrap) getCRPriority(cr apiv3.CommonService) int {
	if cr.Namespace == b.CSData.OperatorNs && cr.Name == "common-service" {
		return 100 // Master CR has highest priority
	}
	return 0
}

// mergeSize merges size field (takes largest)
func (b *Bootstrap) mergeSize(merged, cr *apiv3.CommonService, baseName, crName string, conflicts *[]apiv3.MergeConflict) {
	if cr.Spec.Size == "" {
		return
	}

	if merged.Spec.Size == "" {
		merged.Spec.Size = cr.Spec.Size
		return
	}

	if merged.Spec.Size == cr.Spec.Size {
		return
	}

	// Different sizes - take largest
	largestSize, hadConflict := b.getLargestSize(merged.Spec.Size, cr.Spec.Size)
	if hadConflict {
		winner := baseName
		if largestSize == cr.Spec.Size {
			winner = crName
		}
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.size",
			Values:     []string{merged.Spec.Size, cr.Spec.Size},
			Resolution: "largest",
			Winner:     winner,
		})
		klog.Infof("Size conflict: %s vs %s -> using %s", merged.Spec.Size, cr.Spec.Size, largestSize)
	}
	merged.Spec.Size = largestSize
}

// getLargestSize returns the largest size and whether there was a conflict
func (b *Bootstrap) getLargestSize(size1, size2 string) (string, bool) {
	priority1 := b.getSizePriority(size1)
	priority2 := b.getSizePriority(size2)

	if priority1 == priority2 {
		return size1, false
	}

	if priority1 > priority2 {
		return size1, true
	}
	return size2, true
}

// getSizePriority returns numeric priority for size
func (b *Bootstrap) getSizePriority(size string) int {
	priorities := map[string]int{
		"starterset": 0,
		"starter":    0,
		"small":      1,
		"medium":     2,
		"large":      3,
	}
	if p, ok := priorities[size]; ok {
		return p
	}
	return 0
}

// mergeInstallPlanApproval merges installPlanApproval (Manual > Automatic)
func (b *Bootstrap) mergeInstallPlanApproval(merged, cr *apiv3.CommonService, baseName, crName string, conflicts *[]apiv3.MergeConflict) {
	if cr.Spec.InstallPlanApproval == "" {
		return
	}

	if merged.Spec.InstallPlanApproval == "" {
		merged.Spec.InstallPlanApproval = cr.Spec.InstallPlanApproval
		return
	}

	if merged.Spec.InstallPlanApproval == cr.Spec.InstallPlanApproval {
		return
	}

	// Manual is more restrictive than Automatic
	if merged.Spec.InstallPlanApproval == olmv1alpha1.ApprovalManual || cr.Spec.InstallPlanApproval == olmv1alpha1.ApprovalManual {
		winner := baseName
		if cr.Spec.InstallPlanApproval == olmv1alpha1.ApprovalManual {
			winner = crName
			merged.Spec.InstallPlanApproval = olmv1alpha1.ApprovalManual
		}
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.installPlanApproval",
			Values:     []string{string(merged.Spec.InstallPlanApproval), string(cr.Spec.InstallPlanApproval)},
			Resolution: "most_restrictive",
			Winner:     winner,
		})
		klog.Infof("InstallPlanApproval conflict: using Manual (most restrictive)")
	}
}

// mergeBooleans merges all boolean fields (OR logic - true wins)
func (b *Bootstrap) mergeBooleans(merged, cr *apiv3.CommonService, baseName, crName string, conflicts *[]apiv3.MergeConflict) {
	// FipsEnabled
	if !merged.Spec.FipsEnabled && cr.Spec.FipsEnabled {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.fipsEnabled",
			Values:     []string{"false", "true"},
			Resolution: "logical_or",
			Winner:     crName,
		})
		merged.Spec.FipsEnabled = true
		klog.Infof("FipsEnabled conflict: false vs true -> using true (OR logic)")
	}

	// ManualManagement
	if !merged.Spec.ManualManagement && cr.Spec.ManualManagement {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.manualManagement",
			Values:     []string{"false", "true"},
			Resolution: "logical_or",
			Winner:     crName,
		})
		merged.Spec.ManualManagement = true
	}

	// BYOCACertificate
	if !merged.Spec.BYOCACertificate && cr.Spec.BYOCACertificate {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.BYOCACertificate",
			Values:     []string{"false", "true"},
			Resolution: "logical_or",
			Winner:     crName,
		})
		merged.Spec.BYOCACertificate = true
	}

	// DisableManageCertRotation
	if !merged.Spec.DisableManageCertRotation && cr.Spec.DisableManageCertRotation {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.disableManageCertRotation",
			Values:     []string{"false", "true"},
			Resolution: "logical_or",
			Winner:     crName,
		})
		merged.Spec.DisableManageCertRotation = true
	}

	// AutoScaleConfig
	if !merged.Spec.AutoScaleConfig && cr.Spec.AutoScaleConfig {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.autoScaleConfig",
			Values:     []string{"false", "true"},
			Resolution: "logical_or",
			Winner:     crName,
		})
		merged.Spec.AutoScaleConfig = true
	}

	// EnableInstanaMetricCollection
	if !merged.Spec.EnableInstanaMetricCollection && cr.Spec.EnableInstanaMetricCollection {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.enableInstanaMetricCollection",
			Values:     []string{"false", "true"},
			Resolution: "logical_or",
			Winner:     crName,
		})
		merged.Spec.EnableInstanaMetricCollection = true
	}
}

// mergeStrings merges string fields (first non-empty with master priority)
func (b *Bootstrap) mergeStrings(merged, cr *apiv3.CommonService, baseName, crName string, conflicts *[]apiv3.MergeConflict) {
	// RouteHost
	if merged.Spec.RouteHost == "" && cr.Spec.RouteHost != "" {
		merged.Spec.RouteHost = cr.Spec.RouteHost
	} else if merged.Spec.RouteHost != "" && cr.Spec.RouteHost != "" && merged.Spec.RouteHost != cr.Spec.RouteHost {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.routeHost",
			Values:     []string{merged.Spec.RouteHost, cr.Spec.RouteHost},
			Resolution: "first_non_empty",
			Winner:     baseName,
		})
	}

	// StorageClass
	if merged.Spec.StorageClass == "" && cr.Spec.StorageClass != "" {
		merged.Spec.StorageClass = cr.Spec.StorageClass
	} else if merged.Spec.StorageClass != "" && cr.Spec.StorageClass != "" && merged.Spec.StorageClass != cr.Spec.StorageClass {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.storageClass",
			Values:     []string{merged.Spec.StorageClass, cr.Spec.StorageClass},
			Resolution: "first_non_empty",
			Winner:     baseName,
		})
	}

	// ProfileController
	if merged.Spec.ProfileController == "" && cr.Spec.ProfileController != "" {
		merged.Spec.ProfileController = cr.Spec.ProfileController
	} else if merged.Spec.ProfileController != "" && cr.Spec.ProfileController != "" && merged.Spec.ProfileController != cr.Spec.ProfileController {
		// For ProfileController, take the most advanced (non-default)
		if merged.Spec.ProfileController == "default" && cr.Spec.ProfileController != "default" {
			*conflicts = append(*conflicts, apiv3.MergeConflict{
				Field:      "spec.profileController",
				Values:     []string{merged.Spec.ProfileController, cr.Spec.ProfileController},
				Resolution: "most_advanced",
				Winner:     crName,
			})
			merged.Spec.ProfileController = cr.Spec.ProfileController
		} else if merged.Spec.ProfileController != "default" && cr.Spec.ProfileController == "default" {
			*conflicts = append(*conflicts, apiv3.MergeConflict{
				Field:      "spec.profileController",
				Values:     []string{merged.Spec.ProfileController, cr.Spec.ProfileController},
				Resolution: "most_advanced",
				Winner:     baseName,
			})
		} else {
			*conflicts = append(*conflicts, apiv3.MergeConflict{
				Field:      "spec.profileController",
				Values:     []string{merged.Spec.ProfileController, cr.Spec.ProfileController},
				Resolution: "first_non_empty",
				Winner:     baseName,
			})
		}
	}

	// DefaultAdminUser
	if merged.Spec.DefaultAdminUser == "" && cr.Spec.DefaultAdminUser != "" {
		merged.Spec.DefaultAdminUser = cr.Spec.DefaultAdminUser
	} else if merged.Spec.DefaultAdminUser != "" && cr.Spec.DefaultAdminUser != "" && merged.Spec.DefaultAdminUser != cr.Spec.DefaultAdminUser {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.defaultAdminUser",
			Values:     []string{merged.Spec.DefaultAdminUser, cr.Spec.DefaultAdminUser},
			Resolution: "first_non_empty",
			Winner:     baseName,
		})
	}

	// ImagePullSecret
	if merged.Spec.ImagePullSecret == "" && cr.Spec.ImagePullSecret != "" {
		merged.Spec.ImagePullSecret = cr.Spec.ImagePullSecret
	} else if merged.Spec.ImagePullSecret != "" && cr.Spec.ImagePullSecret != "" && merged.Spec.ImagePullSecret != cr.Spec.ImagePullSecret {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.imagePullSecret",
			Values:     []string{merged.Spec.ImagePullSecret, cr.Spec.ImagePullSecret},
			Resolution: "first_non_empty",
			Winner:     baseName,
		})
	}
}

// mergeLabels merges labels (union with conflict resolution)
func (b *Bootstrap) mergeLabels(merged, cr *apiv3.CommonService, baseName, crName string, conflicts *[]apiv3.MergeConflict) {
	if len(cr.Spec.Labels) == 0 {
		return
	}

	if merged.Spec.Labels == nil {
		merged.Spec.Labels = make(map[string]string)
	}

	for key, value := range cr.Spec.Labels {
		if existingValue, exists := merged.Spec.Labels[key]; exists {
			if existingValue != value {
				*conflicts = append(*conflicts, apiv3.MergeConflict{
					Field:      fmt.Sprintf("spec.labels[%s]", key),
					Values:     []string{existingValue, value},
					Resolution: "first_value",
					Winner:     baseName,
				})
			}
		} else {
			merged.Spec.Labels[key] = value
		}
	}
}

// mergeFeatures merges features configuration
func (b *Bootstrap) mergeFeatures(merged, cr *apiv3.CommonService, baseName, crName string, conflicts *[]apiv3.MergeConflict) {
	if cr.Spec.Features == nil {
		return
	}

	if merged.Spec.Features == nil {
		merged.Spec.Features = &apiv3.Features{}
	}

	// Merge Bedrockshim
	if cr.Spec.Features.Bedrockshim != nil {
		if merged.Spec.Features.Bedrockshim == nil {
			merged.Spec.Features.Bedrockshim = cr.Spec.Features.Bedrockshim
		} else {
			// Merge Enabled (OR logic)
			if !merged.Spec.Features.Bedrockshim.Enabled && cr.Spec.Features.Bedrockshim.Enabled {
				*conflicts = append(*conflicts, apiv3.MergeConflict{
					Field:      "spec.features.bedrockshim.enabled",
					Values:     []string{"false", "true"},
					Resolution: "logical_or",
					Winner:     crName,
				})
				merged.Spec.Features.Bedrockshim.Enabled = true
			}
			// Merge CrossplaneProviderRemoval (OR logic)
			if !merged.Spec.Features.Bedrockshim.CrossplaneProviderRemoval && cr.Spec.Features.Bedrockshim.CrossplaneProviderRemoval {
				*conflicts = append(*conflicts, apiv3.MergeConflict{
					Field:      "spec.features.bedrockshim.crossplaneProviderRemoval",
					Values:     []string{"false", "true"},
					Resolution: "logical_or",
					Winner:     crName,
				})
				merged.Spec.Features.Bedrockshim.CrossplaneProviderRemoval = true
			}
		}
	}

	// Merge APICatalog
	if cr.Spec.Features.APICatalog != nil {
		if merged.Spec.Features.APICatalog == nil {
			merged.Spec.Features.APICatalog = cr.Spec.Features.APICatalog
		} else {
			// Merge StorageClass (first non-empty)
			if merged.Spec.Features.APICatalog.StorageClass == "" && cr.Spec.Features.APICatalog.StorageClass != "" {
				merged.Spec.Features.APICatalog.StorageClass = cr.Spec.Features.APICatalog.StorageClass
			} else if merged.Spec.Features.APICatalog.StorageClass != "" && cr.Spec.Features.APICatalog.StorageClass != "" &&
				merged.Spec.Features.APICatalog.StorageClass != cr.Spec.Features.APICatalog.StorageClass {
				*conflicts = append(*conflicts, apiv3.MergeConflict{
					Field:      "spec.features.apiCatalog.storageClass",
					Values:     []string{merged.Spec.Features.APICatalog.StorageClass, cr.Spec.Features.APICatalog.StorageClass},
					Resolution: "first_non_empty",
					Winner:     baseName,
				})
			}
		}
	}
}

// mergeHugePagesField merges hugepages configuration
func (b *Bootstrap) mergeHugePagesField(merged, cr *apiv3.CommonService, baseName, crName string, conflicts *[]apiv3.MergeConflict) {
	if cr.Spec.HugePages == nil {
		return
	}

	if merged.Spec.HugePages == nil {
		merged.Spec.HugePages = cr.Spec.HugePages
		return
	}

	// Merge Enable (OR logic)
	if !merged.Spec.HugePages.Enable && cr.Spec.HugePages.Enable {
		*conflicts = append(*conflicts, apiv3.MergeConflict{
			Field:      "spec.hugepages.enable",
			Values:     []string{"false", "true"},
			Resolution: "logical_or",
			Winner:     crName,
		})
		merged.Spec.HugePages.Enable = true
	}

	// Merge HugePagesSizes (union)
	if len(cr.Spec.HugePages.HugePagesSizes) > 0 {
		if merged.Spec.HugePages.HugePagesSizes == nil {
			merged.Spec.HugePages.HugePagesSizes = make(map[string]string)
		}
		for key, value := range cr.Spec.HugePages.HugePagesSizes {
			if existingValue, exists := merged.Spec.HugePages.HugePagesSizes[key]; exists {
				if existingValue != value {
					*conflicts = append(*conflicts, apiv3.MergeConflict{
						Field:      fmt.Sprintf("spec.hugepages.hugePagesSizes[%s]", key),
						Values:     []string{existingValue, value},
						Resolution: "first_value",
						Winner:     baseName,
					})
				}
			} else {
				merged.Spec.HugePages.HugePagesSizes[key] = value
			}
		}
	}
}

// mergeOperatorConfigsField merges operator configs (max replicas per operator)
func (b *Bootstrap) mergeOperatorConfigsField(merged, cr *apiv3.CommonService, baseName, crName string, conflicts *[]apiv3.MergeConflict) {
	if len(cr.Spec.OperatorConfigs) == 0 {
		return
	}

	// Build map of existing configs by name
	configMap := make(map[string]*apiv3.OperatorConfig)
	for i := range merged.Spec.OperatorConfigs {
		configMap[merged.Spec.OperatorConfigs[i].Name] = &merged.Spec.OperatorConfigs[i]
	}

	// Merge each config from cr
	for _, crConfig := range cr.Spec.OperatorConfigs {
		if existingConfig, exists := configMap[crConfig.Name]; exists {
			// Merge replicas (take maximum)
			if crConfig.Replicas != nil {
				if existingConfig.Replicas == nil {
					existingConfig.Replicas = crConfig.Replicas
				} else if *crConfig.Replicas > *existingConfig.Replicas {
					*conflicts = append(*conflicts, apiv3.MergeConflict{
						Field:      fmt.Sprintf("spec.operatorConfigs[%s].replicas", crConfig.Name),
						Values:     []string{fmt.Sprintf("%d", *existingConfig.Replicas), fmt.Sprintf("%d", *crConfig.Replicas)},
						Resolution: "maximum",
						Winner:     crName,
					})
					existingConfig.Replicas = crConfig.Replicas
				} else if *crConfig.Replicas < *existingConfig.Replicas {
					*conflicts = append(*conflicts, apiv3.MergeConflict{
						Field:      fmt.Sprintf("spec.operatorConfigs[%s].replicas", crConfig.Name),
						Values:     []string{fmt.Sprintf("%d", *existingConfig.Replicas), fmt.Sprintf("%d", *crConfig.Replicas)},
						Resolution: "maximum",
						Winner:     baseName,
					})
				}
			}

			// Merge UserManaged (OR logic)
			if !existingConfig.UserManaged && crConfig.UserManaged {
				*conflicts = append(*conflicts, apiv3.MergeConflict{
					Field:      fmt.Sprintf("spec.operatorConfigs[%s].userManaged", crConfig.Name),
					Values:     []string{"false", "true"},
					Resolution: "logical_or",
					Winner:     crName,
				})
				existingConfig.UserManaged = true
			}
		} else {
			// New config, add it
			merged.Spec.OperatorConfigs = append(merged.Spec.OperatorConfigs, crConfig)
			configMap[crConfig.Name] = &merged.Spec.OperatorConfigs[len(merged.Spec.OperatorConfigs)-1]
		}
	}
}

// mergeServicesField merges services configuration
func (b *Bootstrap) mergeServicesField(merged, cr *apiv3.CommonService, baseName, crName string, conflicts *[]apiv3.MergeConflict) {
	if len(cr.Spec.Services) == 0 {
		return
	}

	// Build map of existing services by name
	serviceMap := make(map[string]*apiv3.ServiceConfig)
	for i := range merged.Spec.Services {
		serviceMap[merged.Spec.Services[i].Name] = &merged.Spec.Services[i]
	}

	// Merge each service from cr
	for _, crService := range cr.Spec.Services {
		if existingService, exists := serviceMap[crService.Name]; exists {
			// Service exists - merge specs (union with first value wins for conflicts)
			for key, value := range crService.Spec {
				if _, exists := existingService.Spec[key]; !exists {
					existingService.Spec[key] = value
				} else {
					// Conflict - keep existing value
					*conflicts = append(*conflicts, apiv3.MergeConflict{
						Field:      fmt.Sprintf("spec.services[%s].spec[%s]", crService.Name, key),
						Values:     []string{"<existing>", "<new>"},
						Resolution: "first_value",
						Winner:     baseName,
					})
				}
			}

			// Merge ManagementStrategy (first non-empty)
			if existingService.ManagementStrategy == "" && crService.ManagementStrategy != "" {
				existingService.ManagementStrategy = crService.ManagementStrategy
			} else if existingService.ManagementStrategy != "" && crService.ManagementStrategy != "" &&
				existingService.ManagementStrategy != crService.ManagementStrategy {
				*conflicts = append(*conflicts, apiv3.MergeConflict{
					Field:      fmt.Sprintf("spec.services[%s].managementStrategy", crService.Name),
					Values:     []string{existingService.ManagementStrategy, crService.ManagementStrategy},
					Resolution: "first_non_empty",
					Winner:     baseName,
				})
			}

			// Merge Resources (append)
			existingService.Resources = append(existingService.Resources, crService.Resources...)
		} else {
			// New service, add it
			merged.Spec.Services = append(merged.Spec.Services, crService)
			serviceMap[crService.Name] = &merged.Spec.Services[len(merged.Spec.Services)-1]
		}
	}
}

// SyncMergedStatusToAllCRs synchronizes the merged status to all CommonService CRs
// This ensures all CRs display identical merge information for transparency
func (b *Bootstrap) SyncMergedStatusToAllCRs(
	ctx context.Context,
	mergedCR *apiv3.CommonService,
	conflicts []apiv3.MergeConflict,
	sourceCRs []apiv3.SourceCR,
	mergedServices []apiv3.MergedServiceSummary,
) error {
	// List all CommonService CRs
	csObjectList := &apiv3.CommonServiceList{}
	if err := b.Client.List(ctx, csObjectList); err != nil {
		return fmt.Errorf("failed to list CommonService CRs for status sync: %v", err)
	}

	klog.Infof("Synchronizing merged status to %d CommonService CRs", len(csObjectList.Items))

	// Update status for each CR
	syncCount := 0
	for _, cs := range csObjectList.Items {
		if cs.GetDeletionTimestamp() != nil {
			klog.V(2).Infof("Skipping deleted CR %s/%s", cs.Namespace, cs.Name)
			continue // Skip deleted CRs
		}

		// Create a copy to update
		updatedCR := cs.DeepCopy()

		// Update merged config status with identical information
		updatedCR.UpdateMergedConfigStatus(mergedCR, conflicts, sourceCRs, mergedServices)

		// Only update if status actually changed
		if !reflect.DeepEqual(cs.Status.AppliedSpec, updatedCR.Status.AppliedSpec) ||
			!reflect.DeepEqual(cs.Status.MergeInfo, updatedCR.Status.MergeInfo) ||
			!reflect.DeepEqual(cs.Status.MergedServices, updatedCR.Status.MergedServices) {

			klog.Infof("Syncing merged status to CR %s/%s", cs.Namespace, cs.Name)
			if err := b.Client.Status().Update(ctx, updatedCR); err != nil {
				klog.Warningf("Failed to sync status to %s/%s: %v", cs.Namespace, cs.Name, err)
				// Continue with other CRs even if one fails
			} else {
				syncCount++
			}
		} else {
			klog.V(2).Infof("Status already up-to-date for CR %s/%s", cs.Namespace, cs.Name)
		}
	}

	klog.Infof("Successfully synced merged status to %d CommonService CRs", syncCount)
	return nil
}
