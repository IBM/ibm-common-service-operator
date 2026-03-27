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
	"runtime"
	"time"

	"k8s.io/klog"
)

// FunctionTimer is used to measure function execution time
type FunctionTimer struct {
	startTime time.Time
	funcName  string
	ctx       context.Context
}

// NewFunctionTimer creates a new function timer
func NewFunctionTimer(ctx context.Context, funcName string) *FunctionTimer {
	return &FunctionTimer{
		startTime: time.Now(),
		funcName:  funcName,
		ctx:       ctx,
	}
}

// Stop stops timing and records execution time
func (ft *FunctionTimer) Stop() {
	elapsed := time.Since(ft.startTime)

	// Get caller information
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Record structured log
	klog.Infof("[PERFORMANCE] Function=%s Duration=%s File=%s Line=%d",
		ft.funcName,
		elapsed.String(),
		file,
		line)

	// If execution time exceeds threshold, record warning
	if elapsed > 30*time.Second {
		klog.Warningf("[PERFORMANCE_WARNING] Function=%s took too long: %s", ft.funcName, elapsed.String())
	} else if elapsed > 10*time.Second {
		klog.Infof("[PERFORMANCE_INFO] Function=%s took longer than expected: %s", ft.funcName, elapsed.String())
	}
}

// StopWithError stops timing and records error information
func (ft *FunctionTimer) StopWithError(err error) {
	elapsed := time.Since(ft.startTime)

	// Get caller information
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	if err != nil {
		klog.Errorf("[PERFORMANCE] Function=%s Duration=%s Status=ERROR Error=%v File=%s Line=%d",
			ft.funcName,
			elapsed.String(),
			err,
			file,
			line)
	} else {
		klog.Infof("[PERFORMANCE] Function=%s Duration=%s Status=SUCCESS File=%s Line=%d",
			ft.funcName,
			elapsed.String(),
			file,
			line)
	}
}

// TimeFunction is a decorator function for automatically measuring function execution time
func TimeFunction(ctx context.Context, funcName string, fn func() error) error {
	timer := NewFunctionTimer(ctx, funcName)
	defer timer.Stop()

	err := fn()
	if err != nil {
		timer.StopWithError(err)
	}
	return err
}

// TimeFunctionWithResult is a decorator function for measuring functions with return values
func TimeFunctionWithResult[T any](ctx context.Context, funcName string, fn func() (T, error)) (T, error) {
	timer := NewFunctionTimer(ctx, funcName)
	defer timer.Stop()

	result, err := fn()
	if err != nil {
		timer.StopWithError(err)
	}
	return result, err
}

// DetailedTimer provides more detailed performance monitoring
type DetailedTimer struct {
	startTime time.Time
	funcName  string
	ctx       context.Context
	steps     []StepTimer
}

// StepTimer records step timing
type StepTimer struct {
	stepName  string
	startTime time.Time
	duration  time.Duration
}

// NewDetailedTimer creates a detailed timer
func NewDetailedTimer(ctx context.Context, funcName string) *DetailedTimer {
	return &DetailedTimer{
		startTime: time.Now(),
		funcName:  funcName,
		ctx:       ctx,
		steps:     make([]StepTimer, 0),
	}
}

// StartStep starts timing a step
func (dt *DetailedTimer) StartStep(stepName string) {
	step := StepTimer{
		stepName:  stepName,
		startTime: time.Now(),
	}
	dt.steps = append(dt.steps, step)
}

// EndStep ends the current step
func (dt *DetailedTimer) EndStep() {
	if len(dt.steps) > 0 {
		lastIndex := len(dt.steps) - 1
		dt.steps[lastIndex].duration = time.Since(dt.steps[lastIndex].startTime)

		step := dt.steps[lastIndex]
		klog.Infof("[STEP_PERFORMANCE] Function=%s Step=%s Duration=%s",
			dt.funcName,
			step.stepName,
			step.duration.String())
	}
}

// Stop stops detailed timing and outputs complete report
func (dt *DetailedTimer) Stop() {
	totalElapsed := time.Since(dt.startTime)

	// Output overall performance
	klog.Infof("[PERFORMANCE_REPORT] Function=%s TotalDuration=%s StepsCount=%d",
		dt.funcName,
		totalElapsed.String(),
		len(dt.steps))

	// Output detailed information for each step
	for _, step := range dt.steps {
		if step.duration > 0 {
			percentage := float64(step.duration) / float64(totalElapsed) * 100
			klog.Infof("[STEP_DETAIL] Function=%s Step=%s Duration=%s Percentage=%.1f%%",
				dt.funcName,
				step.stepName,
				step.duration.String(),
				percentage)
		}
	}

	// If any step takes too long, record warning
	for _, step := range dt.steps {
		if step.duration > 5*time.Second {
			klog.Warningf("[STEP_WARNING] Function=%s Step=%s took too long: %s",
				dt.funcName,
				step.stepName,
				step.duration.String())
		}
	}
}

// GetTotalDuration gets the total execution time
func (dt *DetailedTimer) GetTotalDuration() time.Duration {
	return time.Since(dt.startTime)
}

// GetStepDuration gets the execution time of a specified step
func (dt *DetailedTimer) GetStepDuration(stepName string) time.Duration {
	for _, step := range dt.steps {
		if step.stepName == stepName && step.duration > 0 {
			return step.duration
		}
	}
	return 0
}

// PerformanceMetrics is the performance metrics structure
type PerformanceMetrics struct {
	FunctionName string
	TotalTime    time.Duration
	Steps        map[string]time.Duration
	Success      bool
	Error        error
	Timestamp    time.Time
}

// GetMetrics gets performance metrics
func (dt *DetailedTimer) GetMetrics() PerformanceMetrics {
	metrics := PerformanceMetrics{
		FunctionName: dt.funcName,
		TotalTime:    dt.GetTotalDuration(),
		Steps:        make(map[string]time.Duration),
		Timestamp:    time.Now(),
	}

	for _, step := range dt.steps {
		if step.duration > 0 {
			metrics.Steps[step.stepName] = step.duration
		}
	}

	return metrics
}

// LogPerformanceSummary logs performance summary
func LogPerformanceSummary(metrics PerformanceMetrics) {
	klog.Infof("[PERFORMANCE_SUMMARY] Function=%s TotalTime=%s Success=%t StepsCount=%d",
		metrics.FunctionName,
		metrics.TotalTime.String(),
		metrics.Success,
		len(metrics.Steps))

	if metrics.Error != nil {
		klog.Errorf("[PERFORMANCE_SUMMARY] Function=%s Error=%v",
			metrics.FunctionName,
			metrics.Error)
	}
}
