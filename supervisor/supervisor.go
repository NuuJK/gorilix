package supervisor

import (
	"context"
	"sync"
	"time"

	"github.com/kleeedolinux/gorilix/actor"
)

type ChildSpec struct {
	ID          string
	CreateFunc  func() (actor.Actor, error)
	RestartType RestartType
	Args        map[string]interface{}
}

type RestartType int

const (
	Permanent RestartType = iota

	Temporary

	Transient
)

type SupervisorStatus int

const (
	Running SupervisorStatus = iota

	Restarting

	Stopping

	Stopped
)

type Supervisor interface {
	actor.Actor

	AddChild(spec ChildSpec) (actor.ActorRef, error)

	RemoveChild(id string) error

	GetChild(id string) (actor.ActorRef, error)

	Strategy() Strategy

	Status() SupervisorStatus
}

type DefaultSupervisor struct {
	*actor.DefaultActor
	strategy       Strategy
	children       map[string]actor.Actor
	childRefs      map[string]actor.ActorRef
	childSpecs     map[string]ChildSpec
	childOrder     []string
	restartHistory []time.Time
	status         SupervisorStatus
	lastFailure    error
	mu             sync.RWMutex
}

type childFailureMessage struct {
	childID string
	err     error
}

func NewSupervisor(id string, strategy Strategy) *DefaultSupervisor {
	s := &DefaultSupervisor{
		strategy:       strategy,
		children:       make(map[string]actor.Actor),
		childRefs:      make(map[string]actor.ActorRef),
		childSpecs:     make(map[string]ChildSpec),
		childOrder:     []string{},
		restartHistory: []time.Time{},
		status:         Running,
	}

	s.DefaultActor = actor.NewActor(id, s.processMessage, 100)
	return s
}

func (s *DefaultSupervisor) processMessage(ctx context.Context, msg interface{}) error {
	switch m := msg.(type) {
	case *childFailureMessage:
		return s.handleChildFailure(ctx, m.childID, m.err)
	default:

		return nil
	}
}

func (s *DefaultSupervisor) NotifyChildFailure(ctx context.Context, childID string, err error) error {
	return s.Receive(ctx, &childFailureMessage{
		childID: childID,
		err:     err,
	})
}

func (s *DefaultSupervisor) AddChild(spec ChildSpec) (actor.ActorRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != Running {
		return nil, ErrSupervisorStopped
	}

	if _, exists := s.children[spec.ID]; exists {
		return nil, actor.ErrInvalidActorID
	}

	child, err := spec.CreateFunc()
	if err != nil {
		return nil, err
	}

	childRef := actor.NewActorRef(child)
	s.children[spec.ID] = child
	s.childRefs[spec.ID] = childRef
	s.childSpecs[spec.ID] = spec
	s.childOrder = append(s.childOrder, spec.ID)

	return childRef, nil
}

func (s *DefaultSupervisor) RemoveChild(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != Running {
		return ErrSupervisorStopped
	}

	child, exists := s.children[id]
	if !exists {
		return actor.ErrActorNotFound
	}

	err := child.Stop()
	if err != nil {
		return err
	}

	delete(s.children, id)
	delete(s.childRefs, id)
	delete(s.childSpecs, id)

	for i, childID := range s.childOrder {
		if childID == id {
			s.childOrder = append(s.childOrder[:i], s.childOrder[i+1:]...)
			break
		}
	}

	return nil
}

func (s *DefaultSupervisor) GetChild(id string) (actor.ActorRef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.status != Running {
		return nil, ErrSupervisorStopped
	}

	ref, exists := s.childRefs[id]
	if !exists {
		return nil, actor.ErrActorNotFound
	}

	return ref, nil
}

func (s *DefaultSupervisor) Strategy() Strategy {
	return s.strategy
}

func (s *DefaultSupervisor) Status() SupervisorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *DefaultSupervisor) shouldRestart(childID string, err error) bool {
	spec, exists := s.childSpecs[childID]
	if !exists {
		return false
	}

	switch spec.RestartType {
	case Permanent:
		return true
	case Temporary:
		return false
	case Transient:
		return err != nil
	default:
		return true
	}
}

func (s *DefaultSupervisor) handleChildFailure(ctx context.Context, childID string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == Stopping || s.status == Stopped {
		return nil
	}

	s.lastFailure = err

	now := time.Now()
	s.restartHistory = append(s.restartHistory, now)

	if s.strategy.MaxRestarts() > 0 {

		timeWindow := time.Duration(s.strategy.TimeInterval()) * time.Second
		cutoff := now.Add(-timeWindow)

		var newHistory []time.Time
		validRestarts := 0

		for _, t := range s.restartHistory {
			if t.After(cutoff) {
				newHistory = append(newHistory, t)
				validRestarts++
			}
		}

		s.restartHistory = newHistory

		if validRestarts > s.strategy.MaxRestarts() {

			if s.strategy.ShouldTerminateOnFailure() {
				s.status = Stopping
				go func() {
					_ = s.Stop()
				}()
				return ErrTooManyRestarts
			}

			backoffDuration := s.strategy.CalculateBackoff(validRestarts)
			if backoffDuration > 0 {
				s.status = Restarting
				go func() {
					time.Sleep(backoffDuration)
					s.mu.Lock()
					s.status = Running
					s.mu.Unlock()

					_ = s.NotifyChildFailure(ctx, childID, err)
				}()
				return nil
			}
		}
	}

	if !s.strategy.CircuitBreaker().ShouldAllow() {

		return ErrCircuitBreakerOpen
	}

	if !s.shouldRestart(childID, err) {

		delete(s.children, childID)
		delete(s.childRefs, childID)

		return nil
	}

	s.status = Restarting
	childrenToRestart, err := s.strategy.HandleFailure(ctx, childID, err)
	if err != nil {
		s.status = Running
		return err
	}

	if len(childrenToRestart) == 0 {
		switch s.strategy.Type() {
		case OneForOne:
			childrenToRestart = []string{childID}
		case OneForAll:
			childrenToRestart = make([]string, len(s.childOrder))
			copy(childrenToRestart, s.childOrder)
		case RestForOne:

			pos := -1
			for i, id := range s.childOrder {
				if id == childID {
					pos = i
					break
				}
			}
			if pos >= 0 {
				childrenToRestart = s.childOrder[pos:]
			}
		}
	}

	for _, id := range childrenToRestart {
		spec, exists := s.childSpecs[id]
		if !exists {
			continue
		}

		if child, exists := s.children[id]; exists {
			_ = child.Stop()
		}

		newChild, err := spec.CreateFunc()
		if err != nil {
			continue
		}

		s.children[id] = newChild
		s.childRefs[id] = actor.NewActorRef(newChild)
	}

	s.strategy.CircuitBreaker().RecordSuccess()
	s.status = Running
	return nil
}

func (s *DefaultSupervisor) Stop() error {
	s.mu.Lock()
	if s.status == Stopped {
		s.mu.Unlock()
		return nil
	}

	s.status = Stopping
	childIDs := make([]string, len(s.childOrder))
	copy(childIDs, s.childOrder)
	s.mu.Unlock()

	for _, id := range childIDs {
		_ = s.RemoveChild(id)
	}

	s.mu.Lock()
	s.status = Stopped
	s.mu.Unlock()

	return s.DefaultActor.Stop()
}

func (s *DefaultSupervisor) GetLastFailure() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastFailure
}
