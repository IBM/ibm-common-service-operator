# CS Operator Performance Monitoring System

This document explains how to use the CS Operator performance monitoring system to measure and log the execution time of various functions.

## Overview

The performance monitoring system provides three main types of timers:

1. **FunctionTimer**: Simple function execution time monitoring
2. **DetailedTimer**: Detailed step-level time monitoring
3. **Decorator Functions**: Automated function wrappers

## Usage

### 1. Simple Timer (FunctionTimer)

Suitable for measuring simple function execution time:

```go
import util "github.com/IBM/ibm-common-service-operator/v4/internal/controller/common"

func myFunction(ctx context.Context) error {
    // Create timer
    timer := util.NewFunctionTimer(ctx, "MyFunction")
    defer timer.Stop()
    
    // Your business logic
    doSomething()
    
    return nil
}
```

**Log Output Example:**
```
I1234 12:00:00.000000 [PERFORMANCE] Function=MyFunction Duration=150ms File=/path/to/file.go Line=123
```

### 2. Detailed Timer (DetailedTimer)

Suitable for complex functions that need to monitor multiple steps:

```go
func complexFunction(ctx context.Context) error {
    // Create detailed timer
    timer := util.NewDetailedTimer(ctx, "ComplexFunction")
    defer timer.Stop()
    
    // Step 1: Initialize
    timer.StartStep("Initialize")
    initializeSomething()
    timer.EndStep()
    
    // Step 2: Process data
    timer.StartStep("ProcessData")
    processData()
    timer.EndStep()
    
    // Step 3: Save results
    timer.StartStep("SaveResults")
    saveResults()
    timer.EndStep()
    
    return nil
}
```

**Log Output Example:**
```
I1234 12:00:00.000000 [STEP_PERFORMANCE] Function=ComplexFunction Step=Initialize Duration=50ms
I1234 12:00:00.050000 [STEP_PERFORMANCE] Function=ComplexFunction Step=ProcessData Duration=100ms
I1234 12:00:00.150000 [STEP_PERFORMANCE] Function=ComplexFunction Step=SaveResults Duration=30ms
I1234 12:00:00.180000 [PERFORMANCE_REPORT] Function=ComplexFunction TotalDuration=180ms StepsCount=3
I1234 12:00:00.180000 [STEP_DETAIL] Function=ComplexFunction Step=Initialize Duration=50ms Percentage=27.8%
I1234 12:00:00.180000 [STEP_DETAIL] Function=ComplexFunction Step=ProcessData Duration=100ms Percentage=55.6%
I1234 12:00:00.180000 [STEP_DETAIL] Function=ComplexFunction Step=SaveResults Duration=30ms Percentage=16.7%
```

### 3. Decorator Functions

Automated function wrappers suitable for simple function monitoring:

```go
// Function with no return value
err := util.TimeFunction(ctx, "MyFunction", func() error {
    doSomething()
    return nil
})

// Function with return value
result, err := util.TimeFunctionWithResult(ctx, "MyFunctionWithResult", func() (string, error) {
    return "success", nil
})
```

## Performance Warnings

The system automatically detects functions that take too long to execute and logs warnings:

- **> 10 seconds**: Logs INFO level
- **> 30 seconds**: Logs WARNING level

**Warning Log Examples:**
```
I1234 12:00:00.000000 [PERFORMANCE_INFO] Function=SlowFunction took longer than expected: 15s
W1234 12:00:00.000000 [PERFORMANCE_WARNING] Function=VerySlowFunction took too long: 45s
```

## Usage in Controllers

### Reconcile Method Example

```go
func (r *CommonServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Create performance monitoring timer
    timer := util.NewDetailedTimer(ctx, fmt.Sprintf("Reconcile_%s_%s", req.Namespace, req.Name))
    defer timer.Stop()

    // Step 1: Fetch instance
    timer.StartStep("FetchInstance")
    instance := &apiv3.CommonService{}
    if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
        timer.EndStep()
        return ctrl.Result{}, err
    }
    timer.EndStep()

    // Step 2: Validate
    timer.StartStep("Validate")
    if !instance.Spec.License.Accept {
        timer.EndStep()
        return ctrl.Result{}, fmt.Errorf("license not accepted")
    }
    timer.EndStep()

    // Step 3: Reconcile
    timer.StartStep("Reconcile")
    result, err := r.doReconcile(ctx, instance)
    timer.EndStep()

    return result, err
}
```

### Bootstrap Function Example

```go
func (b *Bootstrap) InitResources(instance *apiv3.CommonService, forceUpdateODLMCRs bool) error {
    // Create performance monitoring timer
    timer := util.NewDetailedTimer(context.Background(), fmt.Sprintf("InitResources_%s_%s", instance.Namespace, instance.Name))
    defer timer.Stop()

    // Step: Validate configuration
    timer.StartStep("ValidateConfig")
    if err := b.validateConfig(instance); err != nil {
        timer.EndStep()
        return err
    }
    timer.EndStep()

    // Step: Initialize resources
    timer.StartStep("InitializeResources")
    if err := b.initializeResources(instance); err != nil {
        timer.EndStep()
        return err
    }
    timer.EndStep()

    return nil
}
```

## Log Analysis

### Using grep to analyze performance logs

```bash
# View all performance logs
kubectl logs -n ibm-common-services -l app.kubernetes.io/name=ibm-common-service-operator | grep "PERFORMANCE"

# View specific function performance
kubectl logs -n ibm-common-services -l app.kubernetes.io/name=ibm-common-service-operator | grep "Reconcile_"

# View performance warnings
kubectl logs -n ibm-common-services -l app.kubernetes.io/name=ibm-common-service-operator | grep "PERFORMANCE_WARNING"

# View step details
kubectl logs -n ibm-common-services -l app.kubernetes.io/name=ibm-common-service-operator | grep "STEP_DETAIL"
```

### Using jq to analyze structured logs

```bash
# Extract performance data (if using structured logging)
kubectl logs -n ibm-common-services -l app.kubernetes.io/name=ibm-common-service-operator | jq 'select(.msg | contains("PERFORMANCE"))'
```

## Best Practices

### 1. Timer Naming

- Use descriptive function names
- Include namespace and resource names (for controller methods)
- Avoid using dynamically generated names

```go
// Good naming
timer := util.NewDetailedTimer(ctx, "Reconcile_ibm-common-services_common-service")

// Avoid
timer := util.NewDetailedTimer(ctx, "func1")
```

### 2. Step Naming

- Use verb-starting step names
- Keep step names concise but descriptive
- Avoid overly deep nesting

```go
// Good step naming
timer.StartStep("FetchInstance")
timer.StartStep("ValidateLicense")
timer.StartStep("UpdateStatus")

// Avoid
timer.StartStep("step1")
timer.StartStep("do_stuff")
```

### 3. Error Handling

```go
func myFunction(ctx context.Context) error {
    timer := util.NewDetailedTimer(ctx, "MyFunction")
    defer timer.Stop()
    
    timer.StartStep("Step1")
    if err := doSomething(); err != nil {
        timer.EndStep()
        timer.StopWithError(err) // Log error
        return err
    }
    timer.EndStep()
    
    return nil
}
```

### 4. Performance Thresholds

Set reasonable performance thresholds based on your application needs:

```go
// These thresholds can be adjusted in timer.go
const (
    INFO_WARNING_THRESHOLD  = 10 * time.Second
    WARNING_THRESHOLD       = 30 * time.Second
    STEP_WARNING_THRESHOLD  = 5 * time.Second
)
```

## Monitoring and Alerting

### Prometheus Metrics

Performance data can be exposed through Prometheus metrics:

```promql
# Function execution time distribution
histogram_quantile(0.95, sum(rate(cs_function_duration_seconds_bucket[5m])) by (le, function_name))

# Average execution time
sum(rate(cs_function_duration_seconds_sum[5m])) by (function_name) / 
sum(rate(cs_function_duration_seconds_count[5m])) by (function_name)
```

### Alert Rules

```yaml
groups:
- name: cs-operator-performance
  rules:
  - alert: FunctionExecutionTimeHigh
    expr: histogram_quantile(0.95, sum(rate(cs_function_duration_seconds_bucket[5m])) by (le, function_name)) > 30
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "CS Operator function execution time is high"
      description: "Function {{ $labels.function_name }} is taking longer than expected"
```

## Troubleshooting

### Common Issues

1. **Timer not outputting logs**
   - Check if `defer timer.Stop()` is called correctly
   - Confirm log level settings are correct

2. **Inaccurate step timing**
   - Ensure each `StartStep` has a corresponding `EndStep`
   - Check for early returns

3. **Too much performance data**
   - Consider using detailed timers only in critical functions
   - Use simple timers instead of detailed timers

### Debugging Tips

```bash
# Real-time performance log viewing
kubectl logs -n ibm-common-services -l app.kubernetes.io/name=ibm-common-service-operator -f | grep "PERFORMANCE"

# Count performance logs
kubectl logs -n ibm-common-services -l app.kubernetes.io/name=ibm-common-service-operator | grep "PERFORMANCE" | wc -l

# View slowest functions
kubectl logs -n ibm-common-services -l app.kubernetes.io/name=ibm-common-service-operator | grep "PERFORMANCE_WARNING"
```

## Summary

The performance monitoring system provides comprehensive execution time monitoring capabilities for the CS Operator. By using these tools appropriately, you can:

- Identify performance bottlenecks
- Monitor execution time of critical functions
- Set up performance alerts
- Optimize operator performance

Remember to balance monitoring detail with performance overhead, and use detailed monitoring only on critical paths.