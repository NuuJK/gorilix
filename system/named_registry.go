package system

import (
	"errors"
	"fmt"
	"sync"

	"github.com/kleeedolinux/gorilix/actor"
)

type NamedRegistry struct {
	nameToActor map[string]actor.ActorRef
	actorToName map[string]string
	mu          sync.RWMutex
}

func NewNamedRegistry() *NamedRegistry {
	return &NamedRegistry{
		nameToActor: make(map[string]actor.ActorRef),
		actorToName: make(map[string]string),
	}
}

func (r *NamedRegistry) Register(name string, actorRef actor.ActorRef) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nameToActor[name]; exists {
		return fmt.Errorf("name '%s' is already registered", name)
	}

	actorID := actorRef.ID()

	if existingName, exists := r.actorToName[actorID]; exists {
		return fmt.Errorf("actor is already registered with name '%s'", existingName)
	}

	r.nameToActor[name] = actorRef
	r.actorToName[actorID] = name
	return nil
}

func (r *NamedRegistry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	actorRef, exists := r.nameToActor[name]
	if !exists {
		return false
	}

	delete(r.nameToActor, name)
	delete(r.actorToName, actorRef.ID())
	return true
}

func (r *NamedRegistry) UnregisterActor(actorID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	name, exists := r.actorToName[actorID]
	if !exists {
		return false
	}

	delete(r.nameToActor, name)
	delete(r.actorToName, actorID)
	return true
}

func (r *NamedRegistry) Lookup(name string) (actor.ActorRef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actorRef, exists := r.nameToActor[name]
	return actorRef, exists
}

func (r *NamedRegistry) LookupName(actorID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name, exists := r.actorToName[actorID]
	return name, exists
}

func (r *NamedRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.nameToActor[name]
	return exists
}

func (r *NamedRegistry) GetAllNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.nameToActor))
	for name := range r.nameToActor {
		names = append(names, name)
	}
	return names
}

func (r *NamedRegistry) Where(predicate func(string, actor.ActorRef) bool) map[string]actor.ActorRef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]actor.ActorRef)
	for name, actorRef := range r.nameToActor {
		if predicate(name, actorRef) {
			result[name] = actorRef
		}
	}
	return result
}

var ErrNameAlreadyRegistered = errors.New("name already registered")
