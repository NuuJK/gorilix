package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kleeedolinux/gorilix/actor"
	"github.com/kleeedolinux/gorilix/genserver"
	"github.com/kleeedolinux/gorilix/system"
)

type CounterMessage struct {
	Action string
	Amount int
}

type CounterResponse struct {
	Value int
	Error error
}

var actorSystem *system.ActorSystem

func counterInit(ctx context.Context, args interface{}) (interface{}, error) {

	initialValue := 0
	if args != nil {
		if iv, ok := args.(int); ok {
			initialValue = iv
		}
	}
	return initialValue, nil
}

func counterTerminate(ctx context.Context, reason error, state interface{}) {
	value, _ := state.(int)
	fmt.Printf("Counter terminating with final value: %d, reason: %v\n", value, reason)
}

func counterCall(ctx context.Context, message interface{}, state interface{}) (interface{}, interface{}, error) {
	value, _ := state.(int)
	msg, ok := message.(CounterMessage)
	if !ok {
		return nil, value, fmt.Errorf("invalid message type")
	}

	switch msg.Action {
	case "get":
		return CounterResponse{Value: value, Error: nil}, value, nil
	case "increment":
		newValue := value + msg.Amount
		return CounterResponse{Value: newValue, Error: nil}, newValue, nil
	case "decrement":
		newValue := value - msg.Amount
		return CounterResponse{Value: newValue, Error: nil}, newValue, nil
	default:
		return CounterResponse{Value: value, Error: fmt.Errorf("unknown action")}, value, nil
	}
}

func counterCast(ctx context.Context, message interface{}, state interface{}) (interface{}, error) {
	value, _ := state.(int)
	msg, ok := message.(CounterMessage)
	if !ok {
		return value, nil
	}

	switch msg.Action {
	case "increment":
		return value + msg.Amount, nil
	case "decrement":
		return value - msg.Amount, nil
	default:
		return value, nil
	}
}

type WatcherActor struct {
	*actor.DefaultActor
	mu       sync.Mutex
	failures map[string]error
}

func NewWatcherActor(id string) *WatcherActor {
	wa := &WatcherActor{
		failures: make(map[string]error),
	}
	wa.DefaultActor = actor.NewActor(id, wa.receive, 10)
	return wa
}

func (w *WatcherActor) receive(ctx context.Context, msg interface{}) error {
	switch m := msg.(type) {
	case *actor.MonitorMessage:
		w.mu.Lock()
		w.failures[m.MonitoredID] = m.Reason
		w.mu.Unlock()
		fmt.Printf("Watcher received failure notification: Actor %s failed with reason: %v\n",
			m.MonitoredID, m.Reason)
	default:
		fmt.Printf("Watcher received unknown message type\n")
	}
	return nil
}

func (w *WatcherActor) getFailures() map[string]error {
	w.mu.Lock()
	defer w.mu.Unlock()

	result := make(map[string]error, len(w.failures))
	for k, v := range w.failures {
		result[k] = v
	}
	return result
}

type EchoActor struct {
	*actor.DefaultActor
}

func NewEchoActor(id string) *EchoActor {
	ea := &EchoActor{}
	ea.DefaultActor = actor.NewActor(id, ea.receive, 10)
	return ea
}

func (e *EchoActor) receive(ctx context.Context, msg interface{}) error {
	fmt.Printf("Echo actor %s received: %v\n", e.ID(), msg)

	if actorSystem != nil {
		if senderRef, found := actorSystem.WhereIs("watcher"); found {
			return senderRef.Send(ctx, fmt.Sprintf("Echo response to: %v", msg))
		}
	}
	return nil
}

type CrashActor struct {
	*actor.DefaultActor
}

func NewCrashActor(id string) *CrashActor {
	ca := &CrashActor{}
	ca.DefaultActor = actor.NewActor(id, ca.receive, 10)
	return ca
}

func (c *CrashActor) receive(ctx context.Context, msg interface{}) error {
	fmt.Printf("Crash actor %s received: %v\n", c.ID(), msg)

	if msg == "crash" {
		return fmt.Errorf("intentional crash")
	}

	return nil
}

func main() {

	actorSystem = system.NewActorSystem("advanced-example")
	defer actorSystem.Stop()

	watcherActor := NewWatcherActor("watcher")
	watcherRef := actor.NewActorRef(watcherActor)
	_, err := actorSystem.SpawnActor("watcher", watcherActor.receive, 10)
	if err != nil {
		log.Fatalf("Failed to spawn watcher actor: %v", err)
	}

	err = actorSystem.RegisterName("watcher", watcherRef)
	if err != nil {
		log.Fatalf("Failed to register watcher name: %v", err)
	}

	counterOptions := genserver.Options{
		InitFunc:      counterInit,
		TerminateFunc: counterTerminate,
		CallHandler:   counterCall,
		CastHandler:   counterCast,
		BufferSize:    10,
		InitArgs:      100,
		Name:          "counter",
	}

	_, err = actorSystem.SpawnGenServer("counter", counterOptions)
	if err != nil {
		log.Fatalf("Failed to spawn counter GenServer: %v", err)
	}

	crashActor := NewCrashActor("crash")
	crashRef := actor.NewActorRef(crashActor)
	_, err = actorSystem.SpawnActor("crash", crashActor.receive, 10)
	if err != nil {
		log.Fatalf("Failed to spawn crash actor: %v", err)
	}

	err = actorSystem.Monitor("watcher", "crash", actor.OneWay)
	if err != nil {
		log.Printf("Failed to set up monitoring: %v", err)
	}

	echoActor := NewEchoActor("echo")
	echoRef := actor.NewActorRef(echoActor)
	_, err = actorSystem.SpawnActor("echo", echoActor.receive, 10)
	if err != nil {
		log.Fatalf("Failed to spawn echo actor: %v", err)
	}

	err = actorSystem.RegisterName("echo", echoRef)
	if err != nil {
		log.Printf("Failed to register echo actor: %v", err)
	}

	fmt.Println("\n--- Named Process Lookup ---")
	if counterByName, found := actorSystem.WhereIs("counter"); found {
		fmt.Println("Successfully found counter by name")

		ctx := context.Background()
		result, err := genserver.MakeCallSync(ctx, counterByName, CounterMessage{
			Action: "get",
		}, 5*time.Second)

		if err != nil {
			fmt.Printf("Error calling counter: %v\n", err)
		} else if resp, ok := result.(CounterResponse); ok {
			fmt.Printf("Counter value: %d\n", resp.Value)
		}

		result, err = genserver.MakeCallSync(ctx, counterByName, CounterMessage{
			Action: "increment",
			Amount: 50,
		}, 5*time.Second)

		if err != nil {
			fmt.Printf("Error incrementing counter: %v\n", err)
		} else if resp, ok := result.(CounterResponse); ok {
			fmt.Printf("Counter incremented, new value: %d\n", resp.Value)
		}

		err = genserver.MakeCast(ctx, counterByName, CounterMessage{
			Action: "decrement",
			Amount: 25,
		})

		if err != nil {
			fmt.Printf("Error sending cast to counter: %v\n", err)
		} else {
			fmt.Println("Sent decrement cast to counter")
		}

		time.Sleep(100 * time.Millisecond)

		result, err = genserver.MakeCallSync(ctx, counterByName, CounterMessage{
			Action: "get",
		}, 5*time.Second)

		if err != nil {
			fmt.Printf("Error calling counter: %v\n", err)
		} else if resp, ok := result.(CounterResponse); ok {
			fmt.Printf("Counter value after cast: %d\n", resp.Value)
		}
	} else {
		fmt.Println("Could not find counter by name")
	}

	fmt.Println("\n--- Monitoring ---")
	ctx := context.Background()

	err = actorSystem.SendNamedMessage(ctx, "echo", "Hello from main")
	if err != nil {
		fmt.Printf("Error sending to echo actor: %v\n", err)
	}

	fmt.Println("Sending crash message to crash actor...")
	err = crashRef.Send(ctx, "crash")
	if err != nil {
		fmt.Printf("Error sending crash message: %v\n", err)
	}

	time.Sleep(100 * time.Millisecond)

	failures := watcherActor.getFailures()
	fmt.Println("Failures detected by watcher:")
	for actorID, reason := range failures {
		fmt.Printf("  - Actor %s: %v\n", actorID, reason)
	}

	actorSystem.NotifyFailure(ctx, "crash", fmt.Errorf("actor crashed and was handled"))

	fmt.Println("\nExample completed successfully")
}
