# Protocol Buffer Serialization in Gorilix

This guide explains how to use Protocol Buffers for efficient message serialization in Gorilix, particularly for cluster communication.

## Overview

Protocol Buffers (protobuf) is a language-neutral, platform-neutral, extensible mechanism for serializing structured data. Gorilix integrates protobuf for several key benefits:

- **Efficiency** - Smaller serialized representation than JSON or XML
- **Type safety** - Strongly typed message definitions
- **Backward/forward compatibility** - Add fields without breaking existing code
- **Language interoperability** - Works with multiple programming languages
- **Performance** - Fast serialization and deserialization

## Setting Up Protocol Buffers

Before using Protocol Buffers with Gorilix, you need to set up the required tools and dependencies:

### 1. Install the Protocol Buffer Compiler

Install the `protoc` compiler from the [official release page](https://github.com/protocolbuffers/protobuf/releases).

### 2. Install Go Protobuf Plugins

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

### 3. Add Dependencies to Your Project

Ensure your `go.mod` file includes the following dependencies:

```
require (
    github.com/golang/protobuf v1.5.3
    google.golang.org/protobuf v1.31.0
)
```

## Defining Message Types

Protocol Buffers use `.proto` files to define message types. Here's how to create message definitions for Gorilix:

### 1. Create a Proto File

Create a directory for your proto files, for example `proto/`:

```bash
mkdir -p proto
```

Then create a `.proto` file, such as `message.proto`:

```protobuf
syntax = "proto3";
package myapp;

option go_package = "github.com/yourusername/yourapp/proto";

import "google/protobuf/timestamp.proto";

// Define your custom message types
message UserMessage {
    string id = 1;
    string user_id = 2;
    string content = 3;
    google.protobuf.Timestamp created_at = 4;
    map<string, string> metadata = 5;
}

// You can define multiple message types in the same file
message SystemMessage {
    string id = 1;
    int32 type = 2;
    bytes payload = 3;
}
```

### 2. Generate Go Code

Generate Go code from your proto file:

```bash
protoc --go_out=. proto/*.proto
```

This will create Go source files with structs and methods for your message types.

## Using Protocol Buffers with Gorilix

Gorilix provides built-in support for using Protocol Buffers in cluster communication:

### 1. Import the Generated Code

```go
import (
    "github.com/kleeedolinux/gorilix/cluster"
    pb "github.com/yourusername/yourapp/proto"
)
```

### 2. Create and Serialize Messages

```go
import (
    "time"
    
    "github.com/golang/protobuf/ptypes"
    "github.com/kleeedolinux/gorilix/cluster"
    "github.com/kleeedolinux/gorilix/messaging"
    pb "github.com/yourusername/yourapp/proto"
)

func createAndSerializeMessage() ([]byte, error) {
    // Create timestamp
    timestamp, err := ptypes.TimestampProto(time.Now())
    if err != nil {
        return nil, err
    }
    
    // Create a protobuf message
    userMsg := &pb.UserMessage{
        Id:        "msg-1234",
        UserId:    "user-5678",
        Content:   "Hello from distributed Gorilix!",
        CreatedAt: timestamp,
        Metadata: map[string]string{
            "priority": "high",
            "source":   "web",
        },
    }
    
    // Wrap in a Gorilix message
    message := &messaging.Message{
        Type:      messaging.Normal,
        Payload:   userMsg,
        Sender:    "node1",
        Receiver:  "node2",
        Timestamp: time.Now(),
        ID:        "msg-1234",
        Headers:   map[string]string{"content-type": "application/protobuf"},
    }
    
    // Serialize for transmission
    serialized, err := cluster.SerializeMessage(message)
    if err != nil {
        return nil, err
    }
    
    return serialized, nil
}
```

### 3. Deserialize and Process Messages

```go
func receiveAndDeserializeMessage(data []byte) error {
    // Deserialize the message
    message, err := cluster.DeserializeMessage(data)
    if err != nil {
        return err
    }
    
    // Access the payload
    if userMsg, ok := message.Payload.(*pb.UserMessage); ok {
        fmt.Printf("Received message from user %s: %s\n", 
            userMsg.UserId, userMsg.Content)
    }
    
    return nil
}
```

## Custom Serialization

For more control over serialization, you can implement custom serialization logic:

```go
import (
    "github.com/golang/protobuf/proto"
    pb "github.com/yourusername/yourapp/proto"
)

func customSerialize(userMsg *pb.UserMessage) ([]byte, error) {
    // Marshal the message directly using protobuf
    data, err := proto.Marshal(userMsg)
    if err != nil {
        return nil, err
    }
    return data, nil
}

func customDeserialize(data []byte) (*pb.UserMessage, error) {
    // Unmarshal the message
    userMsg := &pb.UserMessage{}
    err := proto.Unmarshal(data, userMsg)
    if err != nil {
        return nil, err
    }
    return userMsg, nil
}
```

## Cluster Communication with Protocol Buffers

When using Gorilix clusters, you'll typically use Protocol Buffers for node-to-node communication:

```go
func sendMessageToRemoteNode(clusterInstance *cluster.Cluster, targetNode string, userMsg *pb.UserMessage) error {
    // Serialize the protobuf message
    data, err := proto.Marshal(userMsg)
    if err != nil {
        return err
    }
    
    // Send to the target node
    return clusterInstance.SendToNode(targetNode, data)
}
```

## Advanced Topics

### Message Versioning

For evolving systems, implement versioning to maintain backward compatibility:

```protobuf
message UserMessageV1 {
    string id = 1;
    string content = 2;
}

message UserMessageV2 {
    string id = 1;
    string content = 2;
    string user_id = 3;  // New field
    int32 priority = 4;  // New field
}
```

### Performance Considerations

- **Message Size**: Keep your messages compact by using appropriate field types
- **Optional Fields**: Use optional fields for data that isn't always present
- **Avoid Excessive Nesting**: Deeply nested messages can impact performance
- **Reuse Message Objects**: Consider pooling message objects for frequent communications

## Common Issues and Solutions

### Issue: "No required module provides package..."

**Solution**: Ensure your `go.mod` has the correct dependencies and run `go mod tidy`.

### Issue: Serialization fails with "failed to serialize payload"

**Solution**: Check that your payload implements the required protobuf interfaces.

### Issue: Deserialization returns wrong message type

**Solution**: Verify that you're using the correct message type for deserialization.

## Real-World Example

Here's a complete example showing protocol buffer serialization in a cluster setup:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/golang/protobuf/proto"
    "github.com/golang/protobuf/ptypes"
    "github.com/kleeedolinux/gorilix/cluster/bridge"
    "github.com/kleeedolinux/gorilix/system"
    pb "github.com/yourusername/yourapp/proto"
)

func main() {
    // Set up actor system with clustering
    actorSystem := system.NewActorSystem("node1")
    actorSystem.SetClusterProvider(bridge.NewClusterProvider())
    
    clusterConfig := &system.ClusterConfig{
        NodeName: "node1",
        BindAddr: "0.0.0.0",
        BindPort: 7946,
    }
    
    err := actorSystem.EnableClustering(clusterConfig)
    if err != nil {
        log.Fatalf("Failed to enable clustering: %v", err)
    }
    
    // Get cluster instance
    clusterInstance, _ := actorSystem.GetCluster()
    
    // Create a protobuf message
    timestamp, _ := ptypes.TimestampProto(time.Now())
    userMsg := &pb.UserMessage{
        Id:        "msg-1234",
        UserId:    "user-5678", 
        Content:   "Hello from node1!",
        CreatedAt: timestamp,
    }
    
    // Serialize the message
    data, err := proto.Marshal(userMsg)
    if err != nil {
        log.Fatalf("Failed to serialize message: %v", err)
    }
    
    // Monitor for other nodes and send message when found
    go func() {
        for {
            time.Sleep(5 * time.Second)
            members := clusterInstance.Members()
            
            for _, member := range members {
                // Skip self
                if member.GetName() == "node1" {
                    continue
                }
                
                fmt.Printf("Sending message to %s\n", member.GetName())
                
                // In a real implementation, you would use the adapter pattern
                // to access SendToNode on the system.Cluster interface
                
                // This is conceptual code - actual implementation would use the bridge:
                // bridge.SendToNode(member.GetName(), data)
                
                fmt.Printf("Message sent to %s\n", member.GetName())
            }
        }
    }()
    
    // Block forever
    select {}
}
```

## Next Steps

- Refer to the [Protocol Buffers Language Guide](https://developers.google.com/protocol-buffers/docs/proto3) for more details on .proto file syntax
- Check out the [Clustering Guide](clustering.md) to learn more about Gorilix's cluster capabilities
- Explore the examples in `examples/protobuf/` for more usage patterns 