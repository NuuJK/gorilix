# Fault Tolerance in Gorilix

This document provides an overview of the fault tolerance features in Gorilix, including retry mechanisms, backoff strategies, and circuit breakers.

## Overview

Gorilix provides a robust fault tolerance system for building resilient applications. The primary components include:

1. **Supervisor Strategies** - Defines how failures are handled
2. **Backoff Mechanisms** - Controls retry timing
3. **Circuit Breaker** - Prevents cascading failures

## Supervisor Strategies

Gorilix supports three supervision strategies:

- **OneForOne** - When a child fails, only that child is restarted
- **OneForAll** - When a child fails, all children are restarted
- **RestForOne** - When a child fails, that child and all children started after it are restarted

## Backoff Strategies

Backoff strategies control the timing of retry attempts:

- **NoBackoff** - No delay between retries
- **LinearBackoff** - Delay increases linearly with each retry
- **ExponentialBackoff** - Delay doubles with each retry (e.g., 100ms, 200ms, 400ms...)
- **JitteredExponentialBackoff** - Exponential backoff with random jitter to prevent thundering herd problem

## Circuit Breaker

The circuit breaker pattern prevents a failing component from consuming resources. It has three states:

1. **Closed** - Normal operation, requests flow through
2. **Open** - Failing, requests are rejected
3. **HalfOpen** - Testing if the system has recovered

### Configuration Options

Circuit breakers can be configured with:

- **Trip Threshold** - Number of failures before opening the circuit
- **Failure Window** - Time window for counting failures
- **Reset Timeout** - Time before transitioning from Open to HalfOpen
- **Success Threshold** - Number of consecutive successes before closing the circuit

## Example Usage

```go
// Create a resilient strategy with circuit breaker and exponential backoff
options := supervisor.StrategyOptions{
    BackoffType: supervisor.JitteredExponentialBackoff,
    BaseBackoff: 100 * time.Millisecond,
    MaxBackoff:  10 * time.Second,
    CircuitBreakerOptions: &supervisor.CircuitBreakerOptions{
        Enabled:          true,
        TripThreshold:    5,                // Open after 5 failures
        FailureWindow:    30 * time.Second, // Count failures in 30s window
        ResetTimeout:     5 * time.Second,  // After 5s, try again
        SuccessThreshold: 2,                // Close after 2 consecutive successes
    },
}

// Create a strategy
strategy := supervisor.NewStrategyWithOptions(
    supervisor.OneForOne,
    10,            // Max restarts
    60,            // Time interval in seconds
    options,
)

// Create a supervisor with this strategy
mySupervisor := supervisor.NewSupervisor("root", strategy)
```

## Best Practices

1. **Choose the Right Strategy** - OneForOne is often sufficient for independent actors
2. **Configure Backoff Properly** - Start with small values and increase exponentially
3. **Use Circuit Breakers** - Especially for external service calls
4. **Set Appropriate Thresholds** - Based on your application's failure tolerance
5. **Monitor Recovery** - Log circuit breaker state changes

## Advanced Usage

See the `examples/resilient_system.go` for a complete example of fault tolerance in action. 