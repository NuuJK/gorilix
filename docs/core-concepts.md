# Core Concepts of Gorilix

Gorilix is built around several key concepts that enable robust concurrent programming. This page explains these concepts in simple terms.

## Actors

Actors are the building blocks of Gorilix applications.

### What is an Actor?

An actor is an independent unit of computation with:
- Its own private state that cannot be accessed directly from outside
- The ability to receive and process messages
- The ability to send messages to other actors
- The ability to create new actors

Think of an actor as a worker that has:
- A mailbox to receive messages
- A brain to process messages
- Hands to send messages to others

### Actor Lifecycle

Actors in Gorilix follow a simple lifecycle:
1. **Creation**: An actor is created and initialized
2. **Running**: The actor processes messages from its mailbox
3. **Stopping**: The actor stops processing messages and cleans up resources

## Messages

Communication between actors happens exclusively through messages.

### Message Passing

Instead of directly calling methods or accessing data, actors communicate by sending messages to each other. This provides isolation between actors, making the system more robust.

### Message Types

In Gorilix, you can send any Go value as a message. However, it's best practice to define specific message types for your application:

```go
// Define a message type
type GreetingMessage struct {
    Name    string
    Message string
}

// Send a message to an actor
actorRef.Send(ctx, &GreetingMessage{
    Name: "Alice",
    Message: "Hello, Bob!",
})
```

### Message Processing

When an actor receives a message, it:
1. Takes the message from its mailbox
2. Processes the message using its message handler function
3. Optionally updates its internal state
4. Optionally sends messages to other actors

## Actor System

The actor system is the environment where actors live and interact.

### What is an Actor System?

An actor system:
- Manages the lifecycle of actors
- Provides a way to look up actors by ID or name
- Handles actor failures and recovery
- Provides utilities for actor communication

### Creating an Actor System

```go
// Create a new actor system
actorSystem := system.NewActorSystem("my-system")
defer actorSystem.Stop()
```

### Spawning Actors

You can create and register actors with the system:

```go
// Create an actor
actorRef, err := actorSystem.SpawnActor("worker1", messageHandler, bufferSize)
```

## Actor References

Actor references (ActorRef) provide a way to interact with actors without having direct access to them.

### What is an ActorRef?

An ActorRef is a reference to an actor that allows:
- Sending messages to the actor
- Getting the actor's ID
- Checking if the actor is running

### Using ActorRefs

```go
// Send a message to an actor
err := actorRef.Send(ctx, "Hello!")

// Get actor ID
id := actorRef.ID()

// Check if actor is running
isRunning := actorRef.IsRunning()
```

## Supervision

Supervision is a way to handle failures in actors.

### What is Supervision?

Supervision is a pattern where one actor (the supervisor) monitors and controls the lifecycle of other actors (the children). When a child actor fails, the supervisor decides how to handle the failure based on a supervision strategy.

### Supervision Strategies

Gorilix supports these supervision strategies:
- **OneForOne**: Only the failed child is restarted
- **OneForAll**: All children are restarted when one fails
- **RestForOne**: The failed child and all children created after it are restarted

### Creating a Supervisor

```go
// Create a supervisor with OneForOne strategy
strategy := supervisor.NewStrategy(supervisor.OneForOne, maxRestarts, timeInterval)
supervisor := supervisor.NewSupervisor("my-supervisor", strategy)
```

## Next Steps

Now that you understand the core concepts, learn about:
- [Actor System](actor-system.md) - Detailed information about the actor system
- [Supervision](supervision.md) - In-depth guide to supervision
- [Messaging](messaging.md) - Advanced messaging features 