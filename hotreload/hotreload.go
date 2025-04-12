package hotreload

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kleeedolinux/gorilix/actor"
)

var (
	ErrModuleNotFound     = errors.New("module not found")
	ErrVersionNotFound    = errors.New("version not found")
	ErrActorNotReloadable = errors.New("actor is not reloadable")
	ErrCodeReloadFailed   = errors.New("code reload failed")
	ErrActorNotFound      = errors.New("actor not found")
)

type VersionInfo struct {
	Version     string
	Timestamp   time.Time
	Description string
	IsActive    bool
}

type ModuleInfo struct {
	Name          string
	Versions      map[string]*VersionInfo
	ActiveVersion string
	mutex         sync.RWMutex
}

type CodeVersion interface {
	Version() string
	Initialize(ctx context.Context, args interface{}) error
	TransferState(oldState interface{}) (interface{}, error)
}

type ReloadableActor interface {
	actor.Actor
	GetCodeVersion() string
	Upgrade(ctx context.Context, newVersion CodeVersion) error
	GetState() interface{}
}

type HotReloader struct {
	modules      map[string]*ModuleInfo
	actors       map[string]ReloadableActor
	factories    map[string]map[string]func() CodeVersion
	mutex        sync.RWMutex
	reloadEvents chan ReloadEvent
}

type ReloadEventType int

const (
	ModuleRegistered ReloadEventType = iota
	VersionRegistered
	VersionActivated
	ActorUpgraded
	ReloadFailed
)

type ReloadEvent struct {
	Type      ReloadEventType
	Module    string
	Version   string
	ActorID   string
	Timestamp time.Time
	Error     error
}

func NewHotReloader() *HotReloader {
	return &HotReloader{
		modules:      make(map[string]*ModuleInfo),
		actors:       make(map[string]ReloadableActor),
		factories:    make(map[string]map[string]func() CodeVersion),
		reloadEvents: make(chan ReloadEvent, 100),
	}
}

func (h *HotReloader) RegisterModule(moduleName string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, exists := h.modules[moduleName]; exists {
		return nil
	}

	h.modules[moduleName] = &ModuleInfo{
		Name:     moduleName,
		Versions: make(map[string]*VersionInfo),
	}

	h.factories[moduleName] = make(map[string]func() CodeVersion)

	h.reloadEvents <- ReloadEvent{
		Type:      ModuleRegistered,
		Module:    moduleName,
		Timestamp: time.Now(),
	}

	return nil
}

func (h *HotReloader) RegisterVersion(moduleName, version string, factory func() CodeVersion, description string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	module, exists := h.modules[moduleName]
	if !exists {
		return ErrModuleNotFound
	}

	module.mutex.Lock()
	defer module.mutex.Unlock()

	if _, exists := module.Versions[version]; exists {
		return nil
	}

	versionInfo := &VersionInfo{
		Version:     version,
		Timestamp:   time.Now(),
		Description: description,
		IsActive:    false,
	}

	module.Versions[version] = versionInfo
	h.factories[moduleName][version] = factory

	if module.ActiveVersion == "" {
		module.ActiveVersion = version
		versionInfo.IsActive = true
	}

	h.reloadEvents <- ReloadEvent{
		Type:      VersionRegistered,
		Module:    moduleName,
		Version:   version,
		Timestamp: time.Now(),
	}

	return nil
}

func (h *HotReloader) ActivateVersion(moduleName, version string) error {
	h.mutex.RLock()
	module, exists := h.modules[moduleName]
	if !exists {
		h.mutex.RUnlock()
		return ErrModuleNotFound
	}
	h.mutex.RUnlock()

	module.mutex.Lock()
	defer module.mutex.Unlock()

	if _, exists := module.Versions[version]; !exists {
		return ErrVersionNotFound
	}

	if module.ActiveVersion != "" {
		if oldVersion, exists := module.Versions[module.ActiveVersion]; exists {
			oldVersion.IsActive = false
		}
	}

	module.ActiveVersion = version
	module.Versions[version].IsActive = true

	h.reloadEvents <- ReloadEvent{
		Type:      VersionActivated,
		Module:    moduleName,
		Version:   version,
		Timestamp: time.Now(),
	}

	return nil
}

func (h *HotReloader) RegisterActor(actor ReloadableActor) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.actors[actor.ID()] = actor
}

func (h *HotReloader) UnregisterActor(actorID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.actors, actorID)
}

// .
func (h *HotReloader) UpgradeActor(ctx context.Context, actorID, moduleName, version string) error {
	h.mutex.RLock()
	actor, exists := h.actors[actorID]
	if !exists {
		h.mutex.RUnlock()
		return ErrActorNotFound
	}

	module, exists := h.modules[moduleName]
	if !exists {
		h.mutex.RUnlock()
		return ErrModuleNotFound
	}

	module.mutex.RLock()
	versionFactory, exists := h.factories[moduleName][version]
	if !exists {
		module.mutex.RUnlock()
		h.mutex.RUnlock()
		return ErrVersionNotFound
	}
	module.mutex.RUnlock()
	h.mutex.RUnlock()

	newCodeVersion := versionFactory()

	err := actor.Upgrade(ctx, newCodeVersion)
	if err != nil {
		h.reloadEvents <- ReloadEvent{
			Type:      ReloadFailed,
			Module:    moduleName,
			Version:   version,
			ActorID:   actorID,
			Timestamp: time.Now(),
			Error:     err,
		}
		return fmt.Errorf("%w: %s", ErrCodeReloadFailed, err)
	}

	h.reloadEvents <- ReloadEvent{
		Type:      ActorUpgraded,
		Module:    moduleName,
		Version:   version,
		ActorID:   actorID,
		Timestamp: time.Now(),
	}

	return nil
}

func (h *HotReloader) UpgradeAllActors(ctx context.Context, moduleName, version string) (int, error) {
	h.mutex.RLock()
	module, exists := h.modules[moduleName]
	if !exists {
		h.mutex.RUnlock()
		return 0, ErrModuleNotFound
	}

	module.mutex.RLock()
	_, exists = module.Versions[version]
	if !exists {
		module.mutex.RUnlock()
		h.mutex.RUnlock()
		return 0, ErrVersionNotFound
	}
	module.mutex.RUnlock()

	actorIDs := make([]string, 0, len(h.actors))
	for id := range h.actors {
		actorIDs = append(actorIDs, id)
	}
	h.mutex.RUnlock()

	successCount := 0
	failureCount := 0

	for _, id := range actorIDs {
		err := h.UpgradeActor(ctx, id, moduleName, version)
		if err != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		_ = h.ActivateVersion(moduleName, version)
	}

	if failureCount > 0 {
		return successCount, fmt.Errorf("failed to upgrade %d actors", failureCount)
	}

	return successCount, nil
}

func (h *HotReloader) GetReloadEvents() <-chan ReloadEvent {
	return h.reloadEvents
}

func (h *HotReloader) ListModules() map[string]string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	result := make(map[string]string)
	for name, module := range h.modules {
		module.mutex.RLock()
		result[name] = module.ActiveVersion
		module.mutex.RUnlock()
	}

	return result
}

func (h *HotReloader) GetModuleVersions(moduleName string) ([]VersionInfo, error) {
	h.mutex.RLock()
	module, exists := h.modules[moduleName]
	if !exists {
		h.mutex.RUnlock()
		return nil, ErrModuleNotFound
	}
	h.mutex.RUnlock()

	module.mutex.RLock()
	defer module.mutex.RUnlock()

	result := make([]VersionInfo, 0, len(module.Versions))
	for _, v := range module.Versions {
		result = append(result, *v)
	}

	return result, nil
}

func (h *HotReloader) Close() {
	close(h.reloadEvents)
}
