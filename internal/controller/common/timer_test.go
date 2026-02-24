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
	"context"
	"errors"
	"testing"
	"time"
)

func TestFunctionTimer(t *testing.T) {
	ctx := context.Background()

	// 测试正常执行
	timer := NewFunctionTimer(ctx, "TestFunction")
	time.Sleep(10 * time.Millisecond)
	timer.Stop()

	// 测试带错误的执行
	timer2 := NewFunctionTimer(ctx, "TestFunctionWithError")
	time.Sleep(5 * time.Millisecond)
	timer2.StopWithError(errors.New("test error"))
}

func TestDetailedTimer(t *testing.T) {
	ctx := context.Background()

	timer := NewDetailedTimer(ctx, "TestDetailedFunction")

	// Add steps
	timer.StartStep("Step1")
	time.Sleep(20 * time.Millisecond)
	timer.EndStep()

	timer.StartStep("Step2")
	time.Sleep(30 * time.Millisecond)
	timer.EndStep()

	// Stop and get metrics
	timer.Stop()
	metrics := timer.GetMetrics()

	if metrics.FunctionName != "TestDetailedFunction" {
		t.Errorf("Expected function name 'TestDetailedFunction', got '%s'", metrics.FunctionName)
	}

	if len(metrics.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(metrics.Steps))
	}

	if metrics.Steps["Step1"] == 0 {
		t.Error("Step1 duration should be > 0")
	}

	if metrics.Steps["Step2"] == 0 {
		t.Error("Step2 duration should be > 0")
	}
}

func TestTimeFunction(t *testing.T) {
	ctx := context.Background()

	// Test normal function
	err := TimeFunction(ctx, "TestTimeFunction", func() error {
		time.Sleep(15 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test function with error
	err = TimeFunction(ctx, "TestTimeFunctionWithError", func() error {
		time.Sleep(10 * time.Millisecond)
		return errors.New("test error")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestTimeFunctionWithResult(t *testing.T) {
	ctx := context.Background()

	// Test normal function
	result, err := TimeFunctionWithResult(ctx, "TestTimeFunctionWithResult", func() (string, error) {
		time.Sleep(12 * time.Millisecond)
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "success" {
		t.Errorf("Expected result 'success', got '%s'", result)
	}

	// Test function with error
	_, err = TimeFunctionWithResult(ctx, "TestTimeFunctionWithResultError", func() (string, error) {
		time.Sleep(8 * time.Millisecond)
		return "", errors.New("test error")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestDetailedTimerSteps(t *testing.T) {
	ctx := context.Background()

	timer := NewDetailedTimer(ctx, "TestStepsFunction")

	// Test multiple steps
	steps := []string{"Init", "Process", "Validate", "Save"}

	for _, step := range steps {
		timer.StartStep(step)
		time.Sleep(5 * time.Millisecond)
		timer.EndStep()
	}

	timer.Stop()
	metrics := timer.GetMetrics()

	if len(metrics.Steps) != len(steps) {
		t.Errorf("Expected %d steps, got %d", len(steps), len(metrics.Steps))
	}

	for _, step := range steps {
		if metrics.Steps[step] == 0 {
			t.Errorf("Step '%s' duration should be > 0", step)
		}
	}
}

func TestPerformanceMetrics(t *testing.T) {
	ctx := context.Background()

	timer := NewDetailedTimer(ctx, "TestMetricsFunction")
	timer.StartStep("TestStep")
	time.Sleep(10 * time.Millisecond)
	timer.EndStep()

	metrics := timer.GetMetrics()

	if metrics.FunctionName != "TestMetricsFunction" {
		t.Errorf("Expected function name 'TestMetricsFunction', got '%s'", metrics.FunctionName)
	}

	if metrics.TotalTime == 0 {
		t.Error("Total time should be > 0")
	}

	if len(metrics.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(metrics.Steps))
	}

	if metrics.Steps["TestStep"] == 0 {
		t.Error("TestStep duration should be > 0")
	}
}
