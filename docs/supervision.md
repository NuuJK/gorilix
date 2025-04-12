# Supervision in Gorilix

Supervision is a critical part of building fault-tolerant systems. This guide explains how supervision works in Gorilix.

## What is Supervision?

Supervision is a way to handle failures in your application. It's based on the idea that some actors (supervisors) watch over other actors (children). When a child actor fails, the supervisor decides how to handle the failure.

Think of it like this:
- A supervisor is like a manager in a workplace
- When an employee (child actor) has a problem, the manager decides what to do:
  - Let the employee try again
  - Replace the employee
  - Shut down the entire team
  - etc.

## Why Use Supervision?

Supervision helps you build systems that can recover from failures automatically. Instead of your entire application crashing when something goes wrong, supervisors can:
- Restart failed components
- Isolate failures to prevent them from affecting the whole system
- Make your system self-healing

## Supervision Strategies

Gorilix supports different strategies for handling failures:

### OneForOne

When a child fails, only that child is restarted.

```go
strategy := supervisor.NewStrategy(supervisor.OneForOne, 3, 60)
```

This is useful when failures in one child don't affect other children. The supervisor will restart only the failed child.

### OneForAll

When a child fails, all children are restarted.

```go
strategy := supervisor.NewStrategy(supervisor.OneForAll, 3, 60)
```

This is useful when children are interdependent, and a failure in one affects others. The supervisor will restart all children when any child fails.

### RestForOne

When a child fails, that child and all children created after it are restarted.

```go
strategy := supervisor.NewStrategy(supervisor.RestForOne, 3, 60)
```

This is useful when children have dependencies on each other in a specific order.

## Creating a Supervisor

You can create a supervisor with a specific strategy:

```go
// Create a supervision strategy
strategy := supervisor.NewStrategy(supervisor.OneForOne, 3, 60)

// Create a supervisor
sup := supervisor.NewSupervisor("my-supervisor", strategy)

// Register the supervisor in the actor system
actorSystem.SpawnActor("my-supervisor", sup.Receive, 10)
```

## Adding Children to a Supervisor

You can add child actors to a supervisor:

```go
// Define how to create a child
childSpec := supervisor.ChildSpec{
    ID: "child1",
    CreateFunc: func() (actor.Actor, error) {
        return NewWorkerActor("child1"), nil
    },
    RestartType: supervisor.Permanent,
}

// Add the child to the supervisor
childRef, err := sup.AddChild(childSpec)
if err != nil {
    // Handle error
}
```

## Restart Types

When adding a child to a supervisor, you can specify how it should be restarted:

### Permanent

The child will always be restarted when it fails.

```go
RestartType: supervisor.Permanent
```

### Temporary

The child will never be restarted when it fails.

```go
RestartType: supervisor.Temporary
```

### Transient

The child will be restarted only if it terminates abnormally (returns an error).

```go
RestartType: supervisor.Transient
```

## Handling Failures

When a child actor returns an error from its message handler, the supervisor will be notified and will handle the failure according to its strategy:

```go
func (a *MyActor) receive(ctx context.Context, msg interface{}) error {
    // Something bad happened
    return fmt.Errorf("an error occurred")
    
    // This will trigger the supervisor's failure handling
}
```

## Complete Supervision Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/kleeedolinux/gorilix/actor"
    "github.com/kleeedolinux/gorilix/supervisor"
    "github.com/kleeedolinux/gorilix/system"
)

// Worker actor that might fail
type WorkerActor struct {
    *actor.DefaultActor
    failureCount int
}

func NewWorkerActor(id string) *WorkerActor {
    w := &WorkerActor{}
    w.DefaultActor = actor.NewActor(id, w.receive, 10)
    return w
}

func (w *WorkerActor) receive(ctx context.Context, msg interface{}) error {
    if msg == "fail" {
        w.failureCount++
        fmt.Printf("Worker %s failing for the %d time\n", w.ID(), w.failureCount)
        return fmt.Errorf("simulated failure")
    }
    
    fmt.Printf("Worker %s received: %v\n", w.ID(), msg)
    return nil
}

func main() {
    // Create actor system
    actorSystem := system.NewActorSystem("supervision-example")
    defer actorSystem.Stop()
    
    // Create a supervision strategy
    strategy := supervisor.NewStrategy(supervisor.OneForOne, 3, 60)
    
    // Create a supervisor
    sup := supervisor.NewSupervisor("main-supervisor", strategy)
    
    // Register the supervisor
    actorSystem.SpawnActor("main-supervisor", sup.Receive, 10)
    
    // Add a worker child to the supervisor
    childSpec := supervisor.ChildSpec{
        ID: "worker1",
        CreateFunc: func() (actor.Actor, error) {
            return NewWorkerActor("worker1"), nil
        },
        RestartType: supervisor.Permanent,
    }
    
    childRef, err := sup.AddChild(childSpec)
    if err != nil {
        fmt.Printf("Error adding child: %v\n", err)
        return
    }
    
    // Send some messages to the worker
    ctx := context.Background()
    
    // Send a normal message
    childRef.Send(ctx, "hello")
    time.Sleep(500 * time.Millisecond)
    
    // Send a failure message
    childRef.Send(ctx, "fail")
    time.Sleep(500 * time.Millisecond)
    
    // Send another normal message
    // The worker should have been restarted by the supervisor
    childRef.Send(ctx, "hello again")
    time.Sleep(500 * time.Millisecond)
    
    fmt.Println("Example completed")
}
```

## Next Steps

- Learn about [Monitoring](monitoring.md) for a more reactive approach to handling failures.
- Explore [GenServer](genserver.md) for a more structured way to define actor behavior. 