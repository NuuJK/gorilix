# Clustering in Gorilix

This guide explains how to set up and use the clustering features in Gorilix to build distributed actor systems.

## Overview

Gorilix's clustering functionality is built on top of [Hashicorp's Memberlist](https://github.com/hashicorp/memberlist), a Go library that provides a complete, lightweight gossip protocol implementation. This enables nodes in your Gorilix system to automatically discover each other, maintain membership information, and detect node failures.

The key benefits of Gorilix's clustering support include:

- **Automatic node discovery** - Nodes can join a cluster automatically using seed nodes
- **Failure detection** - The cluster can detect when nodes fail and update membership information
- **Distributed message passing** - Send messages to actors on other nodes in the cluster
- **Efficient gossip protocol** - Uses an efficient SWIM-based gossip protocol for member communication
- **Low bandwidth usage** - Minimal network overhead for maintaining cluster state

## Setup

To use clustering in your application, follow these steps:

### 1. Import the Required Packages

```go
import (
    "github.com/kleeedolinux/gorilix/cluster/bridge"
    "github.com/kleeedolinux/gorilix/system"
)
```

### 2. Create an Actor System

```go
// Create the actor system with a node name
actorSystem := system.NewActorSystem("node1")
```

### 3. Configure and Enable Clustering

```go
// Set the cluster provider
actorSystem.SetClusterProvider(bridge.NewClusterProvider())

// Configure clustering
clusterConfig := &system.ClusterConfig{
    NodeName: "node1",      // Name of this node
    BindAddr: "0.0.0.0",    // IP to bind to (0.0.0.0 for all interfaces)
    BindPort: 7946,         // Port for cluster communication
    Seeds:    []string{     // Optional list of seed nodes for discovery
        "othernode1:7946",
        "othernode2:7946",
    },
}

// Enable clustering
err := actorSystem.EnableClustering(clusterConfig)
if err != nil {
    log.Fatalf("Failed to enable clustering: %v", err)
}
```

## Cluster Operations

Once you have a cluster running, you can perform various operations:

### Get Cluster Status

```go
// Get the cluster instance
clusterInstance, err := actorSystem.GetCluster()
if err != nil {
    log.Fatalf("Failed to get cluster: %v", err)
}

// Get information about this node
self := clusterInstance.Self()
fmt.Printf("Node: %s at %s:%d\n", 
    self.GetName(), self.GetAddress(), self.GetPort())

// List all cluster members
members := clusterInstance.Members()
for _, member := range members {
    fmt.Printf("Member: %s at %s:%d (Status: %v)\n", 
        member.GetName(), member.GetAddress(), member.GetPort(), member.GetStatus())
}
```

### Join an Existing Cluster

If you didn't provide seed nodes in the initial configuration, you can join an existing cluster later:

```go
// Join a cluster using seed nodes
joinedCount, err := actorSystem.JoinCluster([]string{"othernode1:7946", "othernode2:7946"})
if err != nil {
    log.Fatalf("Failed to join cluster: %v", err)
}
fmt.Printf("Successfully joined %d nodes\n", joinedCount)
```

### Leave a Cluster

When shutting down gracefully, you can leave the cluster:

```go
err := actorSystem.LeaveCluster()
if err != nil {
    log.Printf("Error leaving cluster: %v", err)
}
```

## Distributed Messaging

The real power of clustering comes from the ability to communicate between actors on different nodes.

### Sending Messages to Remote Actors

To send messages between nodes, you typically:

1. Register actors with a well-known name on both nodes
2. Use Protocol Buffers to serialize the messages
3. Send the message to the remote node using the cluster's messaging capabilities

```go
// On Node 1
actorSystem.RegisterName("worker", workerActor)

// On Node 2
actorSystem.RegisterName("worker", workerActor)

// Sending a message from Node 1 to Node 2's "worker" actor
// This is the high-level concept - actual implementation would use
// protocol buffers and the cluster's message passing facilities
```

## Advanced Configuration

For advanced use cases, you can customize various aspects of the clustering behavior:

```go
// Create a more customized cluster configuration
customConfig := &cluster.ClusterConfig{
    NodeName:     "specialnode",
    BindAddr:     "192.168.1.10",      // Specific interface IP
    BindPort:     8765,                // Custom port
    GossipNodes:  5,                   // Number of nodes to gossip with directly
    GossipPort:   8765,                // Port for gossip communication
    PushInterval: 2 * time.Second,     // How often to push state to other nodes
    PullInterval: 5 * time.Second,     // How often to pull state from other nodes
    Seeds:        []string{"seed1:8765"},
}

// You'd then adapt this to use with the system.ClusterConfig:
systemConfig := &system.ClusterConfig{
    NodeName: customConfig.NodeName,
    BindAddr: customConfig.BindAddr,
    BindPort: customConfig.BindPort,
    Seeds:    customConfig.Seeds,
}
```

## Error Handling

Common errors and how to handle them:

- **"clustering not enabled"**: Ensure you've called `EnableClustering` before using any cluster functions
- **"no cluster provider set"**: You need to call `SetClusterProvider` before enabling clustering
- **"failed to join cluster"**: Check that your seed nodes are correct and reachable
- **"node not found"**: The node you're trying to communicate with is not in the cluster

## Real-World Example

Here's a complete example showing node discovery and monitoring:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/kleeedolinux/gorilix/cluster/bridge"
    "github.com/kleeedolinux/gorilix/system"
)

func main() {
    // Get node name and port from environment or arguments
    nodeName := os.Getenv("NODE_NAME")
    if nodeName == "" {
        nodeName = "node1"
    }
    
    // Create the actor system
    actorSystem := system.NewActorSystem(nodeName)
    
    // Enable clustering
    actorSystem.SetClusterProvider(bridge.NewClusterProvider())
    clusterConfig := &system.ClusterConfig{
        NodeName: nodeName,
        BindAddr: "0.0.0.0",
        BindPort: 7946,
    }
    
    // Add seed nodes if provided
    seedsEnv := os.Getenv("SEED_NODES")
    if seedsEnv != "" {
        clusterConfig.Seeds = []string{seedsEnv}
    }
    
    err := actorSystem.EnableClustering(clusterConfig)
    if err != nil {
        log.Fatalf("Failed to enable clustering: %v", err)
    }
    
    // Get cluster information
    clusterInstance, _ := actorSystem.GetCluster()
    self := clusterInstance.Self()
    fmt.Printf("Node started: %s at %s:%d\n", 
        self.GetName(), self.GetAddress(), self.GetPort())
    
    // Background routine to monitor cluster membership
    go func() {
        for {
            time.Sleep(5 * time.Second)
            members := clusterInstance.Members()
            fmt.Printf("\nCluster members (%d):\n", len(members))
            for _, member := range members {
                fmt.Printf("  - %s at %s:%d (Status: %v)\n",
                    member.GetName(), member.GetAddress(), member.GetPort(), member.GetStatus())
            }
        }
    }()
    
    // Wait for interrupt signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    
    // Clean shutdown
    fmt.Println("\nShutting down...")
    actorSystem.LeaveCluster()
    actorSystem.Stop()
    fmt.Println("Node stopped")
}
```

## Next Steps

- Check out the [Protocol Buffer Serialization Guide](serialization.md) to learn how to efficiently serialize messages for cluster communication
- Explore the examples in `examples/cluster/` for more usage patterns 