package system

import (
	"context"
	"fmt"
	"sync"

	"github.com/kleeedolinux/gorilix/actor"
	"github.com/kleeedolinux/gorilix/genserver"
	"github.com/kleeedolinux/gorilix/supervisor"
)

type ActorSystem struct {
	name            string
	rootSupervisor  supervisor.Supervisor
	registry        map[string]actor.ActorRef
	namedRegistry   *NamedRegistry
	actorRegistry   *Registry
	monitorRegistry *actor.MonitorRegistry
	mu              sync.RWMutex
	running         bool
}

func NewActorSystem(name string) *ActorSystem {

	strategy := supervisor.NewStrategy(supervisor.OneForOne, 10, 60)
	rootSupervisor := supervisor.NewSupervisor("root", strategy)

	return &ActorSystem{
		name:            name,
		rootSupervisor:  rootSupervisor,
		registry:        make(map[string]actor.ActorRef),
		namedRegistry:   NewNamedRegistry(),
		actorRegistry:   NewRegistry(),
		monitorRegistry: actor.NewMonitorRegistry(),
		running:         true,
	}
}

func (s *ActorSystem) SpawnActor(id string, receiver func(context.Context, interface{}) error, bufferSize int) (actor.ActorRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil, ErrSystemStopped
	}

	if _, exists := s.registry[id]; exists {
		return nil, actor.ErrInvalidActorID
	}

	createFunc := func() (actor.Actor, error) {
		return actor.NewActor(id, receiver, bufferSize), nil
	}

	spec := supervisor.ChildSpec{
		ID:          id,
		CreateFunc:  createFunc,
		RestartType: supervisor.Permanent,
	}

	actorRef, err := s.rootSupervisor.AddChild(spec)
	if err != nil {
		return nil, err
	}

	s.registry[id] = actorRef
	return actorRef, nil
}

func (s *ActorSystem) SpawnSupervisor(id string, strategyType supervisor.RestartStrategy,
	maxRestarts, timeInterval int) (supervisor.Supervisor, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil, ErrSystemStopped
	}

	if _, exists := s.registry[id]; exists {
		return nil, actor.ErrInvalidActorID
	}

	strategy := supervisor.NewStrategy(strategyType, maxRestarts, timeInterval)

	createFunc := func() (actor.Actor, error) {
		return supervisor.NewSupervisor(id, strategy), nil
	}

	spec := supervisor.ChildSpec{
		ID:          id,
		CreateFunc:  createFunc,
		RestartType: supervisor.Permanent,
	}

	supRef, err := s.rootSupervisor.AddChild(spec)
	if err != nil {
		return nil, err
	}

	s.registry[id] = supRef

	sup, ok := supRef.(supervisor.Supervisor)
	if !ok {
		return nil, fmt.Errorf("failed to cast actor to supervisor")
	}

	return sup, nil
}

func (s *ActorSystem) SpawnGenServer(id string, options genserver.Options) (actor.ActorRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil, ErrSystemStopped
	}

	if _, exists := s.registry[id]; exists {
		return nil, actor.ErrInvalidActorID
	}

	gs, ref, err := genserver.Start(id, options)
	if err != nil {
		return nil, err
	}

	s.registry[id] = ref

	if options.Name != "" {
		err = s.namedRegistry.Register(options.Name, ref)
		if err != nil {

			_ = gs.Stop()
			delete(s.registry, id)
			return nil, err
		}
	}

	return ref, nil
}

func (s *ActorSystem) GetActor(id string) (actor.ActorRef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return nil, ErrSystemStopped
	}

	ref, exists := s.registry[id]
	if !exists {
		return nil, actor.ErrActorNotFound
	}

	return ref, nil
}

func (s *ActorSystem) RegisterName(name string, actorRef actor.ActorRef) error {
	if !s.running {
		return ErrSystemStopped
	}

	return s.namedRegistry.Register(name, actorRef)
}

func (s *ActorSystem) UnregisterName(name string) bool {
	if !s.running {
		return false
	}

	return s.namedRegistry.Unregister(name)
}

func (s *ActorSystem) WhereIs(name string) (actor.ActorRef, bool) {
	if !s.running {
		return nil, false
	}

	return s.namedRegistry.Lookup(name)
}

func (s *ActorSystem) Monitor(monitorID, monitoredID string, linkType actor.MonitorType) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return ErrSystemStopped
	}

	_, exists1 := s.registry[monitorID]
	_, exists2 := s.registry[monitoredID]

	if !exists1 || !exists2 {
		return actor.ErrActorNotFound
	}

	s.monitorRegistry.Monitor(monitorID, monitoredID, linkType)
	return nil
}

func (s *ActorSystem) Demonitor(monitorID, monitoredID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return ErrSystemStopped
	}

	s.monitorRegistry.Demonitor(monitorID, monitoredID)
	return nil
}

func (s *ActorSystem) Stop() error {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	err := s.rootSupervisor.Stop()

	s.mu.Lock()
	s.registry = make(map[string]actor.ActorRef)
	s.mu.Unlock()

	return err
}

func (s *ActorSystem) SendMessage(ctx context.Context, actorID string, message interface{}) error {
	actorRef, err := s.GetActor(actorID)
	if err != nil {
		return err
	}

	return actorRef.Send(ctx, message)
}

func (s *ActorSystem) SendNamedMessage(ctx context.Context, name string, message interface{}) error {
	actorRef, found := s.namedRegistry.Lookup(name)
	if !found {
		return fmt.Errorf("actor with name '%s' not found", name)
	}

	return actorRef.Send(ctx, message)
}

func (s *ActorSystem) NotifyFailure(ctx context.Context, actorID string, reason error) error {
	if !s.running {
		return ErrSystemStopped
	}

	s.monitorRegistry.NotifyMonitors(ctx, actorID, reason, s)

	s.namedRegistry.UnregisterActor(actorID)
	s.monitorRegistry.CleanupActor(actorID)

	return nil
}
