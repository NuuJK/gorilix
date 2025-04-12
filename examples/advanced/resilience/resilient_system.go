package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kleeedolinux/gorilix/actor"
	"github.com/kleeedolinux/gorilix/supervisor"
)


type UnstableActor struct {
	*actor.DefaultActor
	failureRate float64
	attempts    int
	maxAttempts int
}

func NewUnstableActor(id string, failureRate float64, maxAttempts int) *UnstableActor {
	a := &UnstableActor{
		failureRate: failureRate,
		attempts:    0,
		maxAttempts: maxAttempts,
	}
	a.DefaultActor = actor.NewActor(id, a.processMessage, 100)
	return a
}

func (a *UnstableActor) processMessage(ctx context.Context, msg interface{}) error {
	a.attempts++

	
	if rand.Float64() < a.failureRate {
		log.Printf("[%s] Simulating failure (attempt %d)", a.ID(), a.attempts)
		return fmt.Errorf("simulated failure in actor %s", a.ID())
	}

	
	if a.attempts > a.maxAttempts {
		a.failureRate = 0.1 
	}

	log.Printf("[%s] Successfully processed message: %v", a.ID(), msg)
	return nil
}

func main() {
	
	rand.Seed(time.Now().UnixNano())

	
	supervisorOptions := supervisor.StrategyOptions{
		BackoffType:  supervisor.JitteredExponentialBackoff,
		BaseBackoff:  100 * time.Millisecond,
		MaxBackoff:   10 * time.Second,
		JitterFactor: 0.2,
		CircuitBreakerOptions: &supervisor.CircuitBreakerOptions{
			Enabled:          true,
			TripThreshold:    5,                
			FailureWindow:    30 * time.Second, 
			ResetTimeout:     5 * time.Second,  
			SuccessThreshold: 2,                
		},
		TerminateOnMaxRestarts: false, 
	}

	
	strategy := supervisor.NewStrategyWithOptions(
		supervisor.OneForOne,
		10, 
		60, 
		supervisorOptions,
	)

	
	rootSupervisor := supervisor.NewSupervisor("root", strategy)

	
	actors := []struct {
		id          string
		failureRate float64
		maxAttempts int
	}{
		{"actor-1", 0.7, 3}, 
		{"actor-2", 0.3, 5}, 
		{"actor-3", 0.1, 2}, 
	}

	
	for _, a := range actors {
		actorConfig := a 
		childSpec := supervisor.ChildSpec{
			ID: actorConfig.id,
			CreateFunc: func() (actor.Actor, error) {
				return NewUnstableActor(actorConfig.id, actorConfig.failureRate, actorConfig.maxAttempts), nil
			},
			RestartType: supervisor.Permanent, 
		}

		actorRef, err := rootSupervisor.AddChild(childSpec)
		if err != nil {
			log.Fatalf("Failed to add child %s: %v", a.id, err)
		}

		
		err = actorRef.Send(context.Background(), "Start working")
		if err != nil {
			log.Printf("Failed to send message to %s: %v", a.id, err)
		}
	}

	
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				for _, a := range actors {
					actorRef, err := rootSupervisor.GetChild(a.id)
					if err != nil {
						log.Printf("Failed to get child %s: %v", a.id, err)
						continue
					}

					
					err = actorRef.Send(context.Background(), fmt.Sprintf("Work request at %v", time.Now().Format(time.RFC3339)))
					if err != nil {
						log.Printf("Failed to send message to %s: %v", a.id, err)
					}
				}
			}
		}
	}()

	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	
	err := rootSupervisor.Stop()
	if err != nil {
		log.Printf("Error stopping supervisor: %v", err)
	}

	log.Println("System shut down gracefully")
}
