# Monitoring in Gorilix

Monitoring is a critical feature for building resilient applications. This guide explains how monitoring works in Gorilix.

## What is Monitoring?

Monitoring is a mechanism that allows one actor (the monitor) to be notified when another actor (the monitored) fails or stops. Unlike supervision, monitoring is a non-hierarchical relationship - any actor can monitor any other actor.

Think of it as subscribing to failure notifications:
- When an actor you're monitoring fails, you receive a message
- You can then decide how to react to the failure

## Monitoring vs. Supervision

It's important to understand the difference between monitoring and supervision:

|                | Monitoring                   | Supervision                           |
|----------------|------------------------------|---------------------------------------|
| **Relationship** | Peer-to-peer, non-hierarchical | Parent-child, hierarchical            |
| **Purpose**    | Notification of failures     | Management of failures and recovery   |
| **Control**    | No control over monitored actor | Supervisor controls children's lifecycle |
| **Usage**      | React to failures in other parts of the system | Manage and recover from failures in subsystems |

## Types of Monitoring

Gorilix supports two types of monitoring:

### OneWay Monitoring

Only the monitor is notified of the monitored actor's failure.

```go
actorSystem.Monitor("watcher", "worker", actor.OneWay)
```

### Bidirectional Monitoring

Both actors are notified of each other's failures.

```go
actorSystem.Monitor("actorA", "actorB", actor.Bidirectional)
```

## Setting Up Monitoring

To set up monitoring between actors:

```go
// Create an actor system
actorSystem := system.NewActorSystem("monitoring-example")

// Create actors
watcherActor := NewWatcherActor("watcher")
workerActor := NewWorkerActor("worker")

// Register actors
actorSystem.SpawnActor("watcher", watcherActor.receive, 10)
actorSystem.SpawnActor("worker", workerActor.receive, 10)

// Set up monitoring - watcher monitors worker
err := actorSystem.Monitor("watcher", "worker", actor.OneWay)
if err != nil {
    // Handle error
}
```

## Handling Monitor Notifications

When a monitored actor fails, the monitor receives a `MonitorMessage`:

```go
// Watcher actor that receives monitor notifications
type WatcherActor struct {
    *actor.DefaultActor
}

func (w *WatcherActor) receive(ctx context.Context, msg interface{}) error {
    switch m := msg.(type) {
    case *actor.MonitorMessage:
        fmt.Printf("Actor %s failed with reason: %v\n", m.MonitoredID, m.Reason)
        // Handle the failure
        // Maybe restart the actor, notify someone, or take alternative action
    default:
        // Handle other messages
    }
    return nil
}
```

## Removing Monitoring

You can remove a monitoring relationship when it's no longer needed:

```go
// Remove monitoring relationship
err := actorSystem.Demonitor("watcher", "worker")
if err != nil {
    // Handle error
}
```

## Complete Monitoring Example

Here's a complete example demonstrating monitoring in action:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/kleeedolinux/gorilix/actor"
    "github.com/kleeedolinux/gorilix/system"
)

// WatcherActor monitors other actors
type WatcherActor struct {
    *actor.DefaultActor
    failureCount map[string]int
}

func NewWatcherActor(id string) *WatcherActor {
    w := &WatcherActor{
        failureCount: make(map[string]int),
    }
    w.DefaultActor = actor.NewActor(id, w.receive, 10)
    return w
}

func (w *WatcherActor) receive(ctx context.Context, msg interface{}) error {
    switch m := msg.(type) {
    case *actor.MonitorMessage:
        w.failureCount[m.MonitoredID]++
        count := w.failureCount[m.MonitoredID]
        
        fmt.Printf("Watcher detected failure #%d in actor %s: %v\n", 
            count, m.MonitoredID, m.Reason)
            
        // Take action based on the failure
        if count < 3 {
            fmt.Println("Watcher is requesting actor to be restarted")
        } else {
            fmt.Println("Too many failures, giving up")
        }
        
    default:
        fmt.Printf("Watcher received: %v\n", msg)
    }
    return nil
}

// WorkerActor is an actor that can fail
type WorkerActor struct {
    *actor.DefaultActor
    shouldFail bool
}

func NewWorkerActor(id string) *WorkerActor {
    w := &WorkerActor{}
    w.DefaultActor = actor.NewActor(id, w.receive, 10)
    return w
}

func (w *WorkerActor) receive(ctx context.Context, msg interface{}) error {
    fmt.Printf("Worker %s received: %v\n", w.ID(), msg)
    
    if msg == "fail" || w.shouldFail {
        w.shouldFail = true
        return fmt.Errorf("simulated failure in worker %s", w.ID())
    }
    
    if msg == "reset" {
        w.shouldFail = false
    }
    
    return nil
}

func main() {
    // Create actor system
    actorSystem := system.NewActorSystem("monitoring-example")
    defer actorSystem.Stop()
    
    // Create actors
    watcherActor := NewWatcherActor("watcher")
    workerActor := NewWorkerActor("worker")
    
    // Register actors
    actorSystem.SpawnActor("watcher", watcherActor.receive, 10)
    actorSystem.SpawnActor("worker", workerActor.receive, 10)
    
    // Set up monitoring
    err := actorSystem.Monitor("watcher", "worker", actor.OneWay)
    if err != nil {
        fmt.Printf("Error setting up monitoring: %v\n", err)
        return
    }
    
    // Get references to actors
    watcherRef, _ := actorSystem.GetActor("watcher")
    workerRef, _ := actorSystem.GetActor("worker")
    
    ctx := context.Background()
    
    // Send normal message to worker
    workerRef.Send(ctx, "hello")
    time.Sleep(100 * time.Millisecond)
    
    // Cause the worker to fail
    fmt.Println("\nCausing worker to fail...")
    workerRef.Send(ctx, "fail")
    time.Sleep(100 * time.Millisecond)
    
    // Notify system about the failure
    actorSystem.NotifyFailure(ctx, "worker", fmt.Errorf("worker failed"))
    time.Sleep(100 * time.Millisecond)
    
    // Create a new worker (simulating restart)
    newWorker := NewWorkerActor("worker")
    actorSystem.SpawnActor("worker", newWorker.receive, 10)
    
    // Send message to new worker
    newWorkerRef, _ := actorSystem.GetActor("worker")
    newWorkerRef.Send(ctx, "hello after restart")
    time.Sleep(100 * time.Millisecond)
    
    // Cause the worker to fail again
    fmt.Println("\nCausing worker to fail again...")
    newWorkerRef.Send(ctx, "fail")
    time.Sleep(100 * time.Millisecond)
    
    // Notify system about the second failure
    actorSystem.NotifyFailure(ctx, "worker", fmt.Errorf("worker failed again"))
    time.Sleep(100 * time.Millisecond)
    
    fmt.Println("\nExercise monitoring API...")
    // Get monitors for an actor
    monitors := actorSystem.MonitorRegistry.GetMonitors("worker")
    fmt.Printf("Actors monitoring 'worker': %v\n", monitors)
    
    // Remove monitoring
    actorSystem.Demonitor("watcher", "worker")
    
    // Verify monitoring was removed
    monitors = actorSystem.MonitorRegistry.GetMonitors("worker")
    fmt.Printf("Actors monitoring 'worker' after demonitor: %v\n", monitors)
    
    fmt.Println("\nExample completed")
}
```

## When to Use Monitoring

Use monitoring when:
- You need to react to failures in actors you don't own or control
- You want to implement custom recovery strategies beyond what supervisors provide
- You need to be notified about failures across different parts of your system
- You want to implement complex failure detection patterns

## Common Monitoring Patterns

### Heartbeat Monitoring

One actor regularly sends "heartbeat" messages to another. If heartbeats stop arriving, the monitoring actor can take action:

```go
func (a *HeartbeatMonitor) receive(ctx context.Context, msg interface{}) error {
    switch msg.(type) {
    case *HeartbeatMessage:
        a.lastHeartbeat = time.Now()
    case *CheckHeartbeatMessage:
        if time.Since(a.lastHeartbeat) > a.timeout {
            // Take action - the monitored actor is not responding
        }
    }
    return nil
}
```

### Circuit Breaker

Monitor failures and stop sending messages to an actor if it fails too frequently:

```go
func (a *CircuitBreakerActor) receive(ctx context.Context, msg interface{}) error {
    switch m := msg.(type) {
    case *actor.MonitorMessage:
        a.failures++
        if a.failures > a.threshold {
            a.circuitOpen = true
            // Stop sending messages to the failing actor
        }
    case *ResetCircuitMessage:
        a.failures = 0
        a.circuitOpen = false
    }
    return nil
}
```

## Next Steps

- Learn about [Named Processes](named-processes.md) to make actor discovery easier
- Explore practical [Examples](examples.md) showing monitoring in action with real-world scenarios 