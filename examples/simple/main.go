package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kleeedolinux/gorilix/actor"
	"github.com/kleeedolinux/gorilix/messaging"
	"github.com/kleeedolinux/gorilix/supervisor"
	"github.com/kleeedolinux/gorilix/system"
)

type PingMessage struct {
	Count int
}

type PongMessage struct {
	Count int
	From  string
}

type PingActor struct {
	*actor.DefaultActor
	pongReceived int
}

type PongActor struct {
	*actor.DefaultActor
}

func NewPingActor(id string) *PingActor {
	pa := &PingActor{
		pongReceived: 0,
	}
	pa.DefaultActor = actor.NewActor(id, pa.receive, 10)
	return pa
}

func (p *PingActor) receive(ctx context.Context, msg interface{}) error {
	switch m := msg.(type) {
	case *PingMessage:
		fmt.Printf("PingActor %s received PingMessage: %d\n", p.ID(), m.Count)

		if actorSystem != nil {
			pongRef, err := actorSystem.GetActor("pong")
			if err == nil {
				fmt.Printf("PingActor %s sending ping to PongActor\n", p.ID())
				return pongRef.Send(ctx, m)
			}
		}
	case *PongMessage:
		p.pongReceived++
		fmt.Printf("PingActor %s received PongMessage from %s: %d (total: %d)\n",
			p.ID(), m.From, m.Count, p.pongReceived)
	default:
		fmt.Printf("PingActor %s received unknown message type\n", p.ID())
	}
	return nil
}

func NewPongActor(id string) *PongActor {
	pa := &PongActor{}
	pa.DefaultActor = actor.NewActor(id, pa.receive, 10)
	return pa
}

func (p *PongActor) receive(ctx context.Context, msg interface{}) error {
	switch m := msg.(type) {
	case *PingMessage:
		fmt.Printf("PongActor %s received PingMessage: %d\n", p.ID(), m.Count)

		pong := &PongMessage{
			Count: m.Count,
			From:  p.ID(),
		}

		if actorSystem != nil {
			pingRef, err := actorSystem.GetActor("ping")
			if err == nil {
				fmt.Printf("PongActor %s sending pong to PingActor\n", p.ID())
				return pingRef.Send(ctx, pong)
			}
		}
	default:
		fmt.Printf("PongActor %s received unknown message type\n", p.ID())
	}
	return nil
}

var actorSystem *system.ActorSystem

func main() {

	actorSystem = system.NewActorSystem("ping-pong-example")
	defer actorSystem.Stop()

	pingActor := NewPingActor("ping")
	pingRef := actor.NewActorRef(pingActor)

	pongActor := NewPongActor("pong")
	pongRef := actor.NewActorRef(pongActor)

	actorSystem.SpawnActor("ping", pingActor.receive, 10)
	actorSystem.SpawnActor("pong", pongActor.receive, 10)

	messageBus := messaging.NewMessageBus()

	messageBus.Subscribe("game", pingRef)
	messageBus.Subscribe("game", pongRef)

	strategy := supervisor.NewStrategy(supervisor.OneForOne, 3, 10)
	supActor := supervisor.NewSupervisor("game-supervisor", strategy)

	actorSystem.SpawnActor("game-supervisor", supActor.Receive, 10)

	pingSpec := supervisor.ChildSpec{
		ID: "ping-supervised",
		CreateFunc: func() (actor.Actor, error) {
			return NewPingActor("ping-supervised"), nil
		},
		RestartType: supervisor.Permanent,
	}

	pongSpec := supervisor.ChildSpec{
		ID: "pong-supervised",
		CreateFunc: func() (actor.Actor, error) {
			return NewPongActor("pong-supervised"), nil
		},
		RestartType: supervisor.Permanent,
	}

	_, err := supActor.AddChild(pingSpec)
	if err != nil {
		log.Fatalf("Failed to add ping child: %v", err)
	}

	_, err = supActor.AddChild(pongSpec)
	if err != nil {
		log.Fatalf("Failed to add pong child: %v", err)
	}

	ctx := context.Background()

	pingMessage := &PingMessage{Count: 1}
	err = pingRef.Send(ctx, pingMessage)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	broadcastMsg := messaging.Message{
		Type:      messaging.Broadcast,
		Payload:   "Game started!",
		Timestamp: time.Now(),
		ID:        "broadcast-1",
	}

	err = messageBus.Publish(ctx, "game", broadcastMsg)
	if err != nil {
		log.Printf("Failed to broadcast message: %v", err)
	}

	time.Sleep(2 * time.Second)

	pingMessage = &PingMessage{Count: 2}
	err = pingRef.Send(ctx, pingMessage)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	time.Sleep(1 * time.Second)

	fmt.Println("Example completed successfully")
}
