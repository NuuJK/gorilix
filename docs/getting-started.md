# Getting Started with Gorilix

This guide will help you start using Gorilix in your Go applications.

## Installation

Install Gorilix using Go modules:

```bash
go get github.com/kleeedolinux/gorilix
```

## Basic Concepts

Gorilix is based on the **Actor Model** of concurrent computation. Here are the key concepts:

1. **Actors**: Independent units of computation that have their own state
2. **Messages**: How actors communicate with each other
3. **Actor System**: Manages the lifecycle of actors
4. **Supervision**: Controls how actors recover from failures

## Your First Gorilix Application

Let's create a simple application with two actors that send messages to each other.

### 1. Create the project

```bash
mkdir hello-gorilix
cd hello-gorilix
go mod init hello-gorilix
```

### 2. Create a main.go file

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kleeedolinux/gorilix/actor"
	"github.com/kleeedolinux/gorilix/system"
)

// Define our message types
type HelloMessage struct {
	Name string
}

type ReplyMessage struct {
	Greeting string
}

// Greeter actor
type GreeterActor struct {
	*actor.DefaultActor
}

func NewGreeterActor(id string) *GreeterActor {
	g := &GreeterActor{}
	g.DefaultActor = actor.NewActor(id, g.receive, 10)
	return g
}

func (g *GreeterActor) receive(ctx context.Context, msg interface{}) error {
	switch m := msg.(type) {
	case *HelloMessage:
		fmt.Printf("Greeter received Hello from: %s\n", m.Name)
		return nil
	default:
		fmt.Println("Greeter received unknown message")
		return nil
	}
}

// Responder actor
type ResponderActor struct {
	*actor.DefaultActor
}

func NewResponderActor(id string) *ResponderActor {
	r := &ResponderActor{}
	r.DefaultActor = actor.NewActor(id, r.receive, 10)
	return r
}

func (r *ResponderActor) receive(ctx context.Context, msg interface{}) error {
	switch m := msg.(type) {
	case *HelloMessage:
		fmt.Printf("Responder received Hello from: %s\n", m.Name)
		
		// Send a reply
		reply := &ReplyMessage{
			Greeting: fmt.Sprintf("Hello, %s!", m.Name),
		}
		
		// Find the greeter actor and send a message back
		if actorSystem != nil {
			greeterRef, err := actorSystem.GetActor("greeter")
			if err == nil {
				return greeterRef.Send(ctx, reply)
			}
		}
		return nil
	default:
		fmt.Println("Responder received unknown message")
		return nil
	}
}

// Global actor system for this example
var actorSystem *system.ActorSystem

func main() {
	// Create actor system
	actorSystem = system.NewActorSystem("hello-system")
	defer actorSystem.Stop()
	
	// Create actors
	greeterActor := NewGreeterActor("greeter")
	responderActor := NewResponderActor("responder")
	
	// Get actor references
	greeterRef := actor.NewActorRef(greeterActor)
	responderRef := actor.NewActorRef(responderActor)
	
	// Register actors in the system
	actorSystem.SpawnActor("greeter", greeterActor.receive, 10)
	actorSystem.SpawnActor("responder", responderActor.receive, 10)
	
	// Send a message from greeter to responder
	ctx := context.Background()
	helloMsg := &HelloMessage{Name: "Gorilix"}
	
	err := responderRef.Send(ctx, helloMsg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	
	// Wait for processing to complete
	time.Sleep(1 * time.Second)
	
	fmt.Println("Application completed")
}
```

### 3. Run the application

```bash
go run main.go
```

You should see output similar to:

```
Responder received Hello from: Gorilix
Application completed
```

## Understanding the Code

1. We defined two actors: `GreeterActor` and `ResponderActor`
2. Each actor has its own message handler function (`receive`)
3. We created an `ActorSystem` to manage the actors
4. We registered the actors with the system
5. We sent a message from one actor to another

## Next Steps

Now that you have a basic understanding of Gorilix, you can explore more advanced features:

- [Core Concepts](core-concepts.md) - Learn about the fundamental building blocks
- [Supervision](supervision.md) - Discover how to handle actor failures
- [Examples](examples.md) - See more complex examples

Congratulations! You've created your first Gorilix application. 