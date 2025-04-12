# Gorilix

[![Go Reference](https://pkg.go.dev/badge/github.com/kleeedolinux/gorilix.svg)](https://pkg.go.dev/github.com/kleeedolinux/gorilix)

Gorilix is a fault-tolerant actor model framework for Go, inspired by the principles behind Erlang and Elixir. It was born out of the need to create systems that are not only resilient but also capable of scaling seamlessly, even in asynchronous and distributed environments.

## Why Gorilix?

As someone who’s passionate about Go, I wanted to bring the power of the actor model to my favorite stack. The actor model has long been a go-to solution for building robust distributed systems, and frameworks like Erlang and Elixir have shown just how effective it can be for fault tolerance and system reliability. By implementing the actor model in Go, Gorilix aims to offer developers the same reliability and scalability in their Go applications.

The idea was simple: create a lightweight system where actors (essentially independent units of computation) could communicate via message passing, much like Erlang's approach, but tailored for Go’s concurrency model. This allows for building systems that can recover gracefully from failures while maintaining smooth operation at scale.

## Key Features

- **Actor-based concurrency** - Develop systems using isolated actors that communicate via message passing, enabling easy parallelism and isolation.
- **Supervisors** - Manage actor lifecycles with customizable supervision strategies to ensure system reliability.
- **Fault tolerance** - Build systems that automatically recover from failures with adjustable policies.
- **Circuit breakers** - Prevent cascading failures in distributed environments by automatically halting malfunctioning components.
- **Backoff strategies** - Control how retries happen in case of failures with linear, exponential, or jittered backoff strategies.

## Installation

```bash
go get github.com/kleeedolinux/gorilix
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/kleeedolinux/gorilix/actor"
    "github.com/kleeedolinux/gorilix/supervisor"
)

// Define an actor
type GreeterActor struct {
    *actor.DefaultActor
}

func NewGreeterActor(id string) *GreeterActor {
    g := &GreeterActor{}
    g.DefaultActor = actor.NewActor(id, g.processMessage, 100)
    return g
}

func (g *GreeterActor) processMessage(ctx context.Context, msg interface{}) error {
    if name, ok := msg.(string); ok {
        fmt.Printf("Hello, %s!\n", name)
    }
    return nil
}

func main() {
    // Create a simple strategy
    strategy := supervisor.NewStrategy(supervisor.OneForOne, 3, 5)
    
    // Create a supervisor
    sup := supervisor.NewSupervisor("root", strategy)
    
    // Add a child actor
    childSpec := supervisor.ChildSpec{
        ID: "greeter",
        CreateFunc: func() (actor.Actor, error) {
            return NewGreeterActor("greeter"), nil
        },
        RestartType: supervisor.Permanent,
    }
    
    greeterRef, _ := sup.AddChild(childSpec)
    
    // Send a message to the actor
    greeterRef.Send(context.Background(), "World")
    
    // Wait a moment to see the output
    time.Sleep(time.Second)
}
```

## Core Concepts

### Actors

Actors are the building blocks of a Gorilix application. Each actor:
- Has its own state, completely isolated from other actors
- Processes messages one at a time
- Can create other actors
- Can send messages to other actors

### Supervisors

Supervisors monitor actors and decide how to handle failures based on predefined strategies.

### Supervision Strategies

- **OneForOne** - When a child fails, only that child is restarted
- **OneForAll** - When a child fails, all children are restarted
- **RestForOne** - When a child fails, that child and all children started after it are restarted

### Fault Tolerance

Gorilix is designed to handle faults gracefully:

- **Retry with backoff** - Automatically retries failed operations with configurable backoff
- **Circuit breakers** - Stop operation attempts when a system component is failing, avoiding cascading issues
- **Failure isolation** - Contain failures to prevent them from affecting the rest of the system

## Advanced Features

### Circuit Breakers

Circuit breakers prevent cascading failures by halting operations when a system component is failing.

```go
options := supervisor.StrategyOptions{
    CircuitBreakerOptions: &supervisor.CircuitBreakerOptions{
        Enabled:          true,
        TripThreshold:    5,                // Open after 5 failures
        FailureWindow:    30 * time.Second, // Count failures in a 30s window
        ResetTimeout:     5 * time.Second,  // After 5s, try again
        SuccessThreshold: 2,                // Close after 2 consecutive successes
    },
}
```

### Backoff Strategies

Control retry timing with various backoff strategies:

- **NoBackoff** - No delay between retries
- **LinearBackoff** - Delay increases linearly with each retry
- **ExponentialBackoff** - Delay doubles with each retry
- **JitteredExponentialBackoff** - Exponential backoff with random jitter to prevent overloading systems

```go
options := supervisor.StrategyOptions{
    BackoffType:  supervisor.JitteredExponentialBackoff,
    BaseBackoff:  100 * time.Millisecond,
    MaxBackoff:   10 * time.Second,
    JitterFactor: 0.2,
}
```

## Documentation

For more detailed information, check out the [fault tolerance guide](docs/fault_tolerance.md).

## Examples

Feel free to explore the `examples/` directory for more usage examples.

## License

This project is licensed under the terms of the license included in the repository.

## Contributing

Contributions are always welcome! Please feel free to submit a Pull Request.