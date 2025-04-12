package system

import (
	"context"
	"sync"
	"time"

	"github.com/kleeedolinux/gorilix/actor"
)

type Registry struct {
	actors            map[string]actor.ActorRef
	actorsByType      map[string][]string
	actorTypes        map[string]string
	actorAliases      map[string]string
	actorTags         map[string][]string
	actorsByTag       map[string][]string
	actorCreationTime map[string]time.Time
	mu                sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		actors:            make(map[string]actor.ActorRef),
		actorsByType:      make(map[string][]string),
		actorTypes:        make(map[string]string),
		actorAliases:      make(map[string]string),
		actorTags:         make(map[string][]string),
		actorsByTag:       make(map[string][]string),
		actorCreationTime: make(map[string]time.Time),
	}
}

func (r *Registry) Register(actorRef actor.ActorRef, actorType string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	actorID := actorRef.ID()
	r.actors[actorID] = actorRef
	r.actorsByType[actorType] = append(r.actorsByType[actorType], actorID)
	r.actorTypes[actorID] = actorType
	r.actorCreationTime[actorID] = time.Now()
}

func (r *Registry) Unregister(actorID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.actors, actorID)

	actualType, exists := r.actorTypes[actorID]
	if exists {
		actorList := r.actorsByType[actualType]
		for i, id := range actorList {
			if id == actorID {
				r.actorsByType[actualType] = append(actorList[:i], actorList[i+1:]...)
				break
			}
		}
		delete(r.actorTypes, actorID)
	}

	for alias, id := range r.actorAliases {
		if id == actorID {
			delete(r.actorAliases, alias)
		}
	}

	tags, exists := r.actorsByTag[actorID]
	if exists {
		for _, tag := range tags {
			taggedActors := r.actorTags[tag]
			for i, id := range taggedActors {
				if id == actorID {
					r.actorTags[tag] = append(taggedActors[:i], taggedActors[i+1:]...)
					break
				}
			}
		}
		delete(r.actorsByTag, actorID)
	}

	delete(r.actorCreationTime, actorID)
}

func (r *Registry) GetActor(actorID string) (actor.ActorRef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actorRef, exists := r.actors[actorID]
	return actorRef, exists
}

func (r *Registry) GetActorByAlias(alias string) (actor.ActorRef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actorID, exists := r.actorAliases[alias]
	if !exists {
		return nil, false
	}

	actorRef, exists := r.actors[actorID]
	return actorRef, exists
}

func (r *Registry) RegisterAlias(actorID, alias string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.actors[actorID]; !exists {
		return false
	}

	r.actorAliases[alias] = actorID
	return true
}

func (r *Registry) TagActor(actorID string, tags ...string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.actors[actorID]; !exists {
		return false
	}

	for _, tag := range tags {

		r.actorTags[tag] = append(r.actorTags[tag], actorID)

		r.actorsByTag[actorID] = append(r.actorsByTag[actorID], tag)
	}

	return true
}

func (r *Registry) GetActorsByType(actorType string) []actor.ActorRef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actorIDs, exists := r.actorsByType[actorType]
	if !exists {
		return nil
	}

	actors := make([]actor.ActorRef, 0, len(actorIDs))
	for _, id := range actorIDs {
		if actorRef, exists := r.actors[id]; exists {
			actors = append(actors, actorRef)
		}
	}

	return actors
}

func (r *Registry) GetActorsByTag(tag string) []actor.ActorRef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actorIDs, exists := r.actorTags[tag]
	if !exists {
		return nil
	}

	actors := make([]actor.ActorRef, 0, len(actorIDs))
	for _, id := range actorIDs {
		if actorRef, exists := r.actors[id]; exists {
			actors = append(actors, actorRef)
		}
	}

	return actors
}

func (r *Registry) BroadcastToType(ctx context.Context, actorType string, message interface{}) error {
	actors := r.GetActorsByType(actorType)
	if len(actors) == 0 {
		return ErrNoActorsOfType
	}

	var wg sync.WaitGroup
	for _, actorRef := range actors {
		wg.Add(1)
		go func(ar actor.ActorRef) {
			defer wg.Done()
			_ = ar.Send(ctx, message)
		}(actorRef)
	}
	wg.Wait()

	return nil
}

func (r *Registry) BroadcastToTag(ctx context.Context, tag string, message interface{}) error {
	actors := r.GetActorsByTag(tag)
	if len(actors) == 0 {
		return ErrNoActorsWithTag
	}

	var wg sync.WaitGroup
	for _, actorRef := range actors {
		wg.Add(1)
		go func(ar actor.ActorRef) {
			defer wg.Done()
			_ = ar.Send(ctx, message)
		}(actorRef)
	}
	wg.Wait()

	return nil
}

func (r *Registry) GetActorType(actorID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actorType, exists := r.actorTypes[actorID]
	return actorType, exists
}

func (r *Registry) GetActorTags(actorID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.actorsByTag[actorID]
}

func (r *Registry) GetActorCreationTime(actorID string) (time.Time, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	creationTime, exists := r.actorCreationTime[actorID]
	return creationTime, exists
}

func (r *Registry) GetAllActors() []actor.ActorRef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actors := make([]actor.ActorRef, 0, len(r.actors))
	for _, actorRef := range r.actors {
		actors = append(actors, actorRef)
	}

	return actors
}
