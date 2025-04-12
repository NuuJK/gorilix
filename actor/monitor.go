package actor

import (
	"context"
	"sync"
	"time"
)

type MonitorType int

const (
	OneWay MonitorType = iota

	Bidirectional
)

type MonitorMessage struct {
	MonitoredID string
	MonitorID   string
	Reason      error
	Timestamp   int64
}

type monitorLink struct {
	monitoredID string
	monitorID   string
	linkType    MonitorType
}

type MonitorRegistry struct {
	monitors map[string]map[string]MonitorType

	monitoring map[string]map[string]MonitorType
	mu         sync.RWMutex
}

func NewMonitorRegistry() *MonitorRegistry {
	return &MonitorRegistry{
		monitors:   make(map[string]map[string]MonitorType),
		monitoring: make(map[string]map[string]MonitorType),
	}
}

func (r *MonitorRegistry) Monitor(monitorID, monitoredID string, linkType MonitorType) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.monitors[monitoredID]; !exists {
		r.monitors[monitoredID] = make(map[string]MonitorType)
	}
	r.monitors[monitoredID][monitorID] = linkType

	if _, exists := r.monitoring[monitorID]; !exists {
		r.monitoring[monitorID] = make(map[string]MonitorType)
	}
	r.monitoring[monitorID][monitoredID] = linkType

	if linkType == Bidirectional {
		if _, exists := r.monitors[monitorID]; !exists {
			r.monitors[monitorID] = make(map[string]MonitorType)
		}
		r.monitors[monitorID][monitoredID] = linkType

		if _, exists := r.monitoring[monitoredID]; !exists {
			r.monitoring[monitoredID] = make(map[string]MonitorType)
		}
		r.monitoring[monitoredID][monitorID] = linkType
	}
}

func (r *MonitorRegistry) Demonitor(monitorID, monitoredID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	linkType := MonitorType(0)
	if m, exists := r.monitors[monitoredID]; exists {
		if lt, ok := m[monitorID]; ok {
			linkType = lt
			delete(m, monitorID)
		}
	}

	if m, exists := r.monitoring[monitorID]; exists {
		delete(m, monitoredID)
	}

	if linkType == Bidirectional {
		if m, exists := r.monitors[monitorID]; exists {
			delete(m, monitoredID)
		}
		if m, exists := r.monitoring[monitoredID]; exists {
			delete(m, monitorID)
		}
	}
}

func (r *MonitorRegistry) GetMonitors(actorID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if monitors, exists := r.monitors[actorID]; exists {
		ids := make([]string, 0, len(monitors))
		for id := range monitors {
			ids = append(ids, id)
		}
		return ids
	}
	return nil
}

func (r *MonitorRegistry) GetMonitored(actorID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if monitored, exists := r.monitoring[actorID]; exists {
		ids := make([]string, 0, len(monitored))
		for id := range monitored {
			ids = append(ids, id)
		}
		return ids
	}
	return nil
}

func (r *MonitorRegistry) NotifyMonitors(ctx context.Context, actorID string, reason error, actorSystem ActorSystem) {
	r.mu.RLock()
	monitors := r.GetMonitors(actorID)
	r.mu.RUnlock()

	if len(monitors) == 0 {
		return
	}

	msg := &MonitorMessage{
		MonitoredID: actorID,
		Reason:      reason,
		Timestamp:   time.Now().UnixNano(),
	}

	for _, monitorID := range monitors {
		msg.MonitorID = monitorID
		actorRef, err := actorSystem.GetActor(monitorID)
		if err == nil {
			_ = actorRef.Send(ctx, msg)
		}
	}
}

func (r *MonitorRegistry) CleanupActor(actorID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var monitored []string
	if m, exists := r.monitoring[actorID]; exists {
		for id := range m {
			monitored = append(monitored, id)
		}
		delete(r.monitoring, actorID)
	}

	for _, id := range monitored {
		if m, exists := r.monitors[id]; exists {
			delete(m, actorID)
		}
	}

	var monitors []string
	if m, exists := r.monitors[actorID]; exists {
		for id := range m {
			monitors = append(monitors, id)
		}
		delete(r.monitors, actorID)
	}

	for _, id := range monitors {
		if m, exists := r.monitoring[id]; exists {
			delete(m, actorID)
		}
	}
}

var now = func() time.Time {
	return time.Now()
}

type ActorSystem interface {
	GetActor(id string) (ActorRef, error)
}

type int64Timestamp int64
