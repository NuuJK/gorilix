# Named Processes in Gorilix

Named processes are a powerful feature in Gorilix that allow you to register and look up actors by name instead of by ID. This guide explains how to use named processes in your applications.

## What are Named Processes?

Named processes are actors that are registered with a human-readable name. This allows:
- Easy actor discovery without having to pass actor references around
- Location transparency (you don't need to know where the actor is running)
- Simplified actor communication patterns

Think of named processes like a phone book for actors - instead of needing someone's phone number (actor reference), you can look them up by name.

## Why Use Named Processes?

Using named processes provides several benefits:
- **Simplicity**: Actors can find other actors without having to pass references around
- **Loose coupling**: Components don't need direct references to each other
- **Dynamic behavior**: You can swap implementations behind a name
- **Service discovery**: Easily find actors that provide specific services

## Registering a Named Process

To register an actor with a name:

```go
// Create an actor system
actorSystem := system.NewActorSystem("my-system")

// Create an actor
myActor := NewMyActor("actor-id")
myActorRef := actor.NewActorRef(myActor)

// Register the actor with a name
err := actorSystem.RegisterName("my-service", myActorRef)
if err != nil {
    // Handle error
}
```

## Looking Up a Named Process

To find an actor by name:

```go
// Look up an actor by name
actorRef, found := actorSystem.WhereIs("my-service")
if !found {
    // Actor not found
    return
}

// Use the actor reference
ctx := context.Background()
err := actorRef.Send(ctx, "Hello!")
```

## Unregistering a Named Process

To remove a name registration:

```go
// Unregister a name
success := actorSystem.UnregisterName("my-service")
```

## Sending Messages to Named Processes

Gorilix provides a convenient way to send messages directly to named processes:

```go
// Send a message to a named process
ctx := context.Background()
err := actorSystem.SendNamedMessage(ctx, "my-service", "Hello!")
if err != nil {
    // Handle error
}
```

## Named Process Rules

When working with named processes, keep these rules in mind:
1. Names must be unique within an actor system
2. An actor can have multiple names
3. A name can refer to only one actor at a time
4. If an actor is stopped, its names are not automatically unregistered

## Complete Named Processes Example

Here's a complete example demonstrating named processes:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/kleeedolinux/gorilix/actor"
    "github.com/kleeedolinux/gorilix/system"
)

// ServiceActor provides a service
type ServiceActor struct {
    *actor.DefaultActor
    serviceType string
}

func NewServiceActor(id string, serviceType string) *ServiceActor {
    s := &ServiceActor{
        serviceType: serviceType,
    }
    s.DefaultActor = actor.NewActor(id, s.receive, 10)
    return s
}

func (s *ServiceActor) receive(ctx context.Context, msg interface{}) error {
    fmt.Printf("%s service (%s) received: %v\n", 
        s.serviceType, s.ID(), msg)
    return nil
}

// ClientActor uses services
type ClientActor struct {
    *actor.DefaultActor
    actorSystem *system.ActorSystem
}

func NewClientActor(id string, actorSystem *system.ActorSystem) *ClientActor {
    c := &ClientActor{
        actorSystem: actorSystem,
    }
    c.DefaultActor = actor.NewActor(id, c.receive, 10)
    return c
}

func (c *ClientActor) receive(ctx context.Context, msg interface{}) error {
    fmt.Printf("Client (%s) received: %v\n", c.ID(), msg)
    
    if msg == "use-service" {
        // Look up a service by name
        serviceRef, found := c.actorSystem.WhereIs("database-service")
        if found {
            serviceRef.Send(ctx, "Query: SELECT * FROM users")
        } else {
            fmt.Println("Database service not found")
        }
        
        // Send a message to a named service directly
        err := c.actorSystem.SendNamedMessage(ctx, "logger-service", "Log: Client operation")
        if err != nil {
            fmt.Printf("Error sending to logger: %v\n", err)
        }
    }
    
    return nil
}

func main() {
    // Create actor system
    actorSystem := system.NewActorSystem("named-processes-example")
    defer actorSystem.Stop()
    
    // Create service actors
    dbActor := NewServiceActor("db1", "Database")
    loggerActor := NewServiceActor("logger1", "Logger")
    cacheActor := NewServiceActor("cache1", "Cache")
    
    // Register actors with the system
    actorSystem.SpawnActor("db1", dbActor.receive, 10)
    actorSystem.SpawnActor("logger1", loggerActor.receive, 10)
    actorSystem.SpawnActor("cache1", cacheActor.receive, 10)
    
    // Register actors with names
    dbRef := actor.NewActorRef(dbActor)
    loggerRef := actor.NewActorRef(loggerActor)
    cacheRef := actor.NewActorRef(cacheActor)
    
    actorSystem.RegisterName("database-service", dbRef)
    actorSystem.RegisterName("logger-service", loggerRef)
    actorSystem.RegisterName("cache-service", cacheRef)
    
    // Create a client actor
    clientActor := NewClientActor("client1", actorSystem)
    clientRef := actor.NewActorRef(clientActor)
    actorSystem.SpawnActor("client1", clientActor.receive, 10)
    
    // Get all registered names
    fmt.Println("Registered names:")
    names := actorSystem.NamedRegistry.GetAllNames()
    for _, name := range names {
        fmt.Printf("- %s\n", name)
    }
    
    fmt.Println("\nClient using services...")
    // Client uses services
    ctx := context.Background()
    clientRef.Send(ctx, "use-service")
    
    // Wait for messages to be processed
    time.Sleep(100 * time.Millisecond)
    
    // Replace a service implementation
    fmt.Println("\nReplacing database service...")
    newDbActor := NewServiceActor("db2", "New Database")
    actorSystem.SpawnActor("db2", newDbActor.receive, 10)
    newDbRef := actor.NewActorRef(newDbActor)
    
    // Unregister old name and register new implementation
    actorSystem.UnregisterName("database-service")
    actorSystem.RegisterName("database-service", newDbRef)
    
    // Client uses services again (now using the new database)
    clientRef.Send(ctx, "use-service")
    
    // Wait for messages to be processed
    time.Sleep(100 * time.Millisecond)
    
    fmt.Println("\nExample completed")
}
```

## Advanced Named Process Patterns

### Service Discovery

Named processes make it easy to implement service discovery:

```go
// Find all services matching a pattern
services := actorSystem.NamedRegistry.Where(func(name string, _ actor.ActorRef) bool {
    return strings.HasSuffix(name, "-service")
})

// Use the services
for name, ref := range services {
    // Use each service
}
```

### Round-Robin Load Balancing

Register multiple instances with numbered names and rotate between them:

```go
func getNextServiceInstance(actorSystem *system.ActorSystem, servicePrefix string, instanceCount int) actor.ActorRef {
    nextInstance := (currentInstance + 1) % instanceCount
    currentInstance = nextInstance
    
    serviceName := fmt.Sprintf("%s-%d", servicePrefix, nextInstance)
    actorRef, found := actorSystem.WhereIs(serviceName)
    if found {
        return actorRef
    }
    
    return nil
}
```

### Service Registry with Health Checks

Create a service registry actor that monitors registered services:

```go
func (r *RegistryActor) receive(ctx context.Context, msg interface{}) error {
    switch m := msg.(type) {
    case *RegisterServiceMessage:
        // Register service
        r.services[m.Name] = m.ActorRef
        
        // Monitor service health
        actorSystem.Monitor(r.ID(), m.ActorRef.ID(), actor.OneWay)
        
    case *actor.MonitorMessage:
        // Service failed
        for name, actorRef := range r.services {
            if actorRef.ID() == m.MonitoredID {
                delete(r.services, name)
                fmt.Printf("Service %s was removed due to failure\n", name)
                break
            }
        }
    }
    return nil
}
```

## Next Steps

- Learn about [GenServer](genserver.md) to create more structured actors
- Explore [Examples](examples.md) for more real-world use cases of named processes 