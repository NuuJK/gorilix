package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kleeedolinux/gorilix/cluster/bridge"
	"github.com/kleeedolinux/gorilix/messaging"
	"github.com/kleeedolinux/gorilix/system"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <node-name> <port> [seeds]")
		os.Exit(1)
	}

	nodeName := os.Args[1]
	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Invalid port: %v", err)
	}

	
	actorSystem := system.NewActorSystem(nodeName)

	
	actorSystem.SetClusterProvider(bridge.NewClusterProvider())

	
	clusterConfig := &system.ClusterConfig{
		NodeName: nodeName,
		BindAddr: "0.0.0.0",
		BindPort: port,
	}

	
	if len(os.Args) > 3 {
		clusterConfig.Seeds = os.Args[3:]
	}

	err = actorSystem.EnableClustering(clusterConfig)
	if err != nil {
		log.Fatalf("Failed to enable clustering: %v", err)
	}

	
	ctx := context.Background()
	actor, err := actorSystem.SpawnActor("receiver", messageHandler, 100)
	if err != nil {
		log.Fatalf("Failed to spawn actor: %v", err)
	}

	
	err = actorSystem.RegisterName("message-receiver", actor)
	if err != nil {
		log.Fatalf("Failed to register actor: %v", err)
	}

	
	clusterInstance, err := actorSystem.GetCluster()
	if err != nil {
		log.Fatalf("Failed to get cluster: %v", err)
	}

	
	self := clusterInstance.Self()
	fmt.Printf("Node started: %s at %s:%d\n", self.GetName(), self.GetAddress(), self.GetPort())

	go func() {
		for {
			time.Sleep(5 * time.Second)
			members := clusterInstance.Members()
			fmt.Printf("Cluster members (%d):\n", len(members))
			for _, member := range members {
				fmt.Printf("  - %s at %s:%d (Status: %v)\n",
					member.GetName(), member.GetAddress(), member.GetPort(), member.GetStatus())
			}

			
			if len(members) > 1 {
				for _, member := range members {
					if member.GetName() != self.GetName() {
						fmt.Printf("Sending message to %s\n", member.GetName())

						msg := messaging.Message{
							Type:      messaging.Normal,
							Payload:   fmt.Sprintf("Hello from %s at %s", self.GetName(), time.Now().Format(time.RFC3339)),
							Sender:    self.GetName(),
							Receiver:  member.GetName(),
							Timestamp: time.Now(),
							ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
							Headers:   map[string]string{"content-type": "text/plain"},
						}

						
						
						err := actorSystem.SendNamedMessage(ctx, "message-receiver", msg)
						if err != nil {
							fmt.Printf("Error sending message: %v\n", err)
						}
					}
				}
			}
		}
	}()

	
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	
	err = actorSystem.Stop()
	if err != nil {
		log.Printf("Error stopping actor system: %v", err)
	}
	fmt.Println("Node stopped")
}

func messageHandler(ctx context.Context, message interface{}) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		return fmt.Errorf("invalid message type: %T", message)
	}

	fmt.Printf("Received message from %s: %v\n", msg.Sender, msg.Payload)
	return nil
}
