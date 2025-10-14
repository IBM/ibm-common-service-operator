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

package main

import (
	"context"
	"fmt"
	"time"

	util "github.com/IBM/ibm-common-service-operator/v4/internal/controller/common"
	"k8s.io/klog"
)

// Example: How to use the performance monitoring system

func main() {
	ctx := context.Background()

	// Example 1: Using simple function timer
	simpleTimerExample(ctx)

	// Example 2: Using detailed step timer
	detailedTimerExample(ctx)

	// Example 3: Using decorator functions
	decoratorExample(ctx)
}

// Example 1: Simple function timer
func simpleTimerExample(ctx context.Context) {
	klog.Info("=== Simple Timer Example ===")

	// Create timer
	timer := util.NewFunctionTimer(ctx, "SimpleFunction")

	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	// Stop timing and record
	timer.Stop()

	// Timer with error
	timer2 := util.NewFunctionTimer(ctx, "FunctionWithError")
	time.Sleep(50 * time.Millisecond)
	timer2.StopWithError(fmt.Errorf("simulated error"))
}

// Example 2: Detailed step timer
func detailedTimerExample(ctx context.Context) {
	klog.Info("=== Detailed Timer Example ===")

	// Create detailed timer
	timer := util.NewDetailedTimer(ctx, "ComplexFunction")

	// Step 1: Initialize
	timer.StartStep("Initialize")
	time.Sleep(50 * time.Millisecond)
	timer.EndStep()

	// Step 2: Data processing
	timer.StartStep("DataProcessing")
	time.Sleep(100 * time.Millisecond)
	timer.EndStep()

	// Step 3: Validation
	timer.StartStep("Validation")
	time.Sleep(30 * time.Millisecond)
	timer.EndStep()

	// Step 4: Save
	timer.StartStep("Save")
	time.Sleep(80 * time.Millisecond)
	timer.EndStep()

	// Stop timing and output report
	timer.Stop()

	// Get performance metrics
	metrics := timer.GetMetrics()
	klog.Infof("Total execution time: %v", metrics.TotalTime)
	klog.Infof("Number of steps: %d", len(metrics.Steps))
}

// Example 3: Using decorator functions
func decoratorExample(ctx context.Context) {
	klog.Info("=== Decorator Functions Example ===")

	// Using TimeFunction decorator
	err := util.TimeFunction(ctx, "DecoratedFunction", func() error {
		time.Sleep(75 * time.Millisecond)
		return nil
	})

	if err != nil {
		klog.Errorf("Function execution failed: %v", err)
	}

	// Using TimeFunctionWithResult decorator
	result, err := util.TimeFunctionWithResult(ctx, "FunctionWithResult", func() (string, error) {
		time.Sleep(60 * time.Millisecond)
		return "success", nil
	})

	if err != nil {
		klog.Errorf("Function execution failed: %v", err)
	} else {
		klog.Infof("Function returned result: %s", result)
	}
}

// Example 4: Using performance monitoring in controllers
func controllerExample(ctx context.Context) {
	klog.Info("=== Controller Example ===")

	// Simulate controller method
	reconcileExample(ctx, "example-namespace", "example-cr")
}

func reconcileExample(ctx context.Context, namespace, name string) {
	// Create detailed timer
	timer := util.NewDetailedTimer(ctx, fmt.Sprintf("Reconcile_%s_%s", namespace, name))
	defer timer.Stop()

	// Step 1: Get resource
	timer.StartStep("GetResource")
	time.Sleep(20 * time.Millisecond)
	timer.EndStep()

	// Step 2: Validate resource
	timer.StartStep("ValidateResource")
	time.Sleep(15 * time.Millisecond)
	timer.EndStep()

	// Step 3: Update status
	timer.StartStep("UpdateStatus")
	time.Sleep(30 * time.Millisecond)
	timer.EndStep()

	// Step 4: Process sub-resources
	timer.StartStep("ProcessSubResources")
	time.Sleep(100 * time.Millisecond)
	timer.EndStep()

	klog.Info("Reconciliation completed")
}
