# GenServer in Gorilix

The GenServer (Generic Server) pattern is a way to create actors with a standardized structure and behavior. This guide explains how to use GenServer in Gorilix.

## What is GenServer?

GenServer is a pattern borrowed from Erlang/Elixir that provides a standard structure for actors. It simplifies:
- State management
- Message handling
- Synchronous and asynchronous communication
- Initialization and termination

Think of GenServer as a template that helps you create actors with consistent behavior.

## Why Use GenServer?

Using GenServer provides several benefits:
- Standardized structure makes your code more consistent
- Clear separation of concerns
- Supports both synchronous calls and asynchronous casts
- Handles initialization and cleanup automatically

## Basic Concepts

### Calls vs Casts

GenServer supports two main types of messages:

1. **Calls**: Synchronous requests that expect a response
2. **Casts**: Asynchronous messages that don't expect a response

This is similar to the difference between calling a function and getting a return value (call) versus triggering an event and not waiting for a response (cast).

### State Management

GenServer manages state for you:
- Initialize state in the `InitFunc`
- Access and modify state in handler functions
- Clean up state in the `TerminateFunc`

## Creating a GenServer

To create a GenServer, you need to define:
1. An initialization function
2. Handlers for calls and casts
3. Optionally, a termination function

```go
// Create a GenServer
options := genserver.Options{
    InitFunc: func(ctx context.Context, args interface{}) (interface{}, error) {
        // Initialize and return initial state
        return 0, nil  // Start with state = 0
    },
    TerminateFunc: func(ctx context.Context, reason error, state interface{}) {
        // Clean up resources
        fmt.Printf("Terminating with final state: %v\n", state)
    },
    CallHandler: func(ctx context.Context, message interface{}, state interface{}) (interface{}, interface{}, error) {
        // Handle call messages
        // Return: (reply, newState, error)
        currentState := state.(int)
        return currentState + 1, currentState + 1, nil
    },
    CastHandler: func(ctx context.Context, message interface{}, state interface{}) (interface{}, error) {
        // Handle cast messages
        // Return: (newState, error)
        currentState := state.(int)
        return currentState + 1, nil
    },
    BufferSize: 10,  // Mailbox size
}

// Start the GenServer
gsActor, gsRef, err := genserver.Start("counter", options)
if err != nil {
    // Handle error
}

// Register with actor system
actorSystem.RegisterName("counter", gsRef)
```

## Using GenServer Calls

Calls are synchronous requests that expect a response:

```go
// Send a synchronous call to the GenServer
ctx := context.Background()
timeout := 5 * time.Second
result, err := genserver.MakeCallSync(ctx, gsRef, "get_value", timeout)
if err != nil {
    // Handle error
}

// Use the result
fmt.Printf("Result: %v\n", result)
```

## Using GenServer Casts

Casts are asynchronous messages that don't expect a response:

```go
// Send an asynchronous cast to the GenServer
ctx := context.Background()
err := genserver.MakeCast(ctx, gsRef, "increment")
if err != nil {
    // Handle error
}
```

## Complete Counter Example

Here's a complete example of a counter implemented with GenServer:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/kleeedolinux/gorilix/genserver"
    "github.com/kleeedolinux/gorilix/system"
)

// Message types
type CounterMessage struct {
    Action string // "increment", "decrement", "get"
    Value  int
}

// Response type
type CounterResponse struct {
    Value int
    Error error
}

func main() {
    // Create actor system
    actorSystem := system.NewActorSystem("genserver-example")
    defer actorSystem.Stop()

    // Create GenServer options
    options := genserver.Options{
        // Initialize with state = 0
        InitFunc: func(ctx context.Context, args interface{}) (interface{}, error) {
            initialValue := 0
            if args != nil {
                if value, ok := args.(int); ok {
                    initialValue = value
                }
            }
            return initialValue, nil
        },
        
        // Cleanup when terminated
        TerminateFunc: func(ctx context.Context, reason error, state interface{}) {
            value, _ := state.(int)
            fmt.Printf("Counter terminated with value: %d\n", value)
        },
        
        // Handle call messages
        CallHandler: func(ctx context.Context, message interface{}, state interface{}) (interface{}, interface{}, error) {
            value, _ := state.(int)
            
            // Type assertion for message
            msg, ok := message.(CounterMessage)
            if !ok {
                return CounterResponse{Value: value, Error: fmt.Errorf("invalid message")}, value, nil
            }
            
            switch msg.Action {
            case "get":
                return CounterResponse{Value: value, Error: nil}, value, nil
            case "increment":
                newValue := value + msg.Value
                return CounterResponse{Value: newValue, Error: nil}, newValue, nil
            case "decrement":
                newValue := value - msg.Value
                return CounterResponse{Value: newValue, Error: nil}, newValue, nil
            default:
                return CounterResponse{Value: value, Error: fmt.Errorf("unknown action")}, value, nil
            }
        },
        
        // Handle cast messages
        CastHandler: func(ctx context.Context, message interface{}, state interface{}) (interface{}, error) {
            value, _ := state.(int)
            
            // Type assertion for message
            msg, ok := message.(CounterMessage)
            if !ok {
                return value, nil
            }
            
            switch msg.Action {
            case "increment":
                return value + msg.Value, nil
            case "decrement":
                return value - msg.Value, nil
            default:
                return value, nil
            }
        },
        
        BufferSize: 10,
        Name: "counter", // Will be registered with this name
    }
    
    // Start the GenServer
    _, _, err := genserver.Start("counter", options)
    if err != nil {
        fmt.Printf("Error starting GenServer: %v\n", err)
        return
    }
    
    // Get reference by name
    counterRef, found := actorSystem.WhereIs("counter")
    if !found {
        fmt.Println("Counter not found")
        return
    }
    
    // Use the counter with calls and casts
    ctx := context.Background()
    
    // Get current value with a call
    result, err := genserver.MakeCallSync(ctx, counterRef, CounterMessage{
        Action: "get",
    }, 5*time.Second)
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else if resp, ok := result.(CounterResponse); ok {
        fmt.Printf("Current value: %d\n", resp.Value)
    }
    
    // Increment with a call
    result, err = genserver.MakeCallSync(ctx, counterRef, CounterMessage{
        Action: "increment",
        Value: 5,
    }, 5*time.Second)
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else if resp, ok := result.(CounterResponse); ok {
        fmt.Printf("New value after increment: %d\n", resp.Value)
    }
    
    // Decrement with a cast (fire and forget)
    err = genserver.MakeCast(ctx, counterRef, CounterMessage{
        Action: "decrement",
        Value: 2,
    })
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    }
    
    // Wait a bit for the cast to be processed
    time.Sleep(100 * time.Millisecond)
    
    // Get the final value
    result, err = genserver.MakeCallSync(ctx, counterRef, CounterMessage{
        Action: "get",
    }, 5*time.Second)
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else if resp, ok := result.(CounterResponse); ok {
        fmt.Printf("Final value: %d\n", resp.Value)
    }
    
    fmt.Println("Example completed")
}
```

## When to Use GenServer

Use GenServer when:
- You need an actor with persistent state
- You want a clear separation between synchronous and asynchronous message handling
- You need standardized initialization and cleanup

## Next Steps

- Learn about [Named Processes](named-processes.md) to register your GenServer with a name
- Explore [Monitoring](monitoring.md) to handle failures between actors 