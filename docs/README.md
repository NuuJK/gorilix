# Gorilix Documentation

Welcome to the Gorilix documentation! Gorilix is a Golang library that enhances goroutines with Erlang-like concurrency features.

## Table of Contents

- [Getting Started](getting-started.md)
- [Core Concepts](core-concepts.md)
- [Supervision](supervision.md)
- [Messaging](messaging.md)
- [Named Processes](named-processes.md)
- [GenServer](genserver.md)
- [Monitoring](monitoring.md)
- [Examples](examples.md)

## What is Gorilix?

Gorilix brings Erlang-like actor model concurrency to Go. It adds:

- **Isolated Processes**: Each actor runs independently with its own state
- **Message Passing**: Communication between actors happens through messages
- **Supervision**: Actors can be monitored and restarted if they fail
- **Named Processes**: Actors can be registered and looked up by name
- **GenServer Pattern**: Easy way to create actors with standardized behavior

## Quick Start

Install Gorilix:

```bash
go get github.com/kleeedolinux/gorilix
```

Create a simple actor:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/kleeedolinux/gorilix/actor"
    "github.com/kleeedolinux/gorilix/system"
)

func main() {
    // Create the actor system
    actorSystem := system.NewActorSystem("my-system")
    defer actorSystem.Stop()
    
    // Define a message handler
    handler := func(ctx context.Context, msg interface{}) error {
        fmt.Println("Received message:", msg)
        return nil
    }
    
    // Create and register the actor
    actorRef, err := actorSystem.SpawnActor("my-actor", handler, 10)
    if err != nil {
        panic(err)
    }
    
    // Send a message to the actor
    ctx := context.Background()
    actorRef.Send(ctx, "Hello, world!")
}
```

See the [Getting Started](getting-started.md) guide for more details. 
