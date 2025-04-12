package actor

import (
	"context"
	"sync"
	"time"
)

type Actor interface {
	Receive(ctx context.Context, message interface{}) error

	Stop() error

	ID() string

	IsRunning() bool
}

type ActorRef interface {
	Send(ctx context.Context, message interface{}) error

	ID() string

	IsRunning() bool
}

type DefaultActor struct {
	id          string
	mailbox     chan interface{}
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	receiver    func(context.Context, interface{}) error
	stopped     bool
	mu          sync.RWMutex
	lastError   error
	stateData   map[string]interface{}
	stateDataMu sync.RWMutex
}

func NewActor(id string, receiver func(context.Context, interface{}) error, bufferSize int) *DefaultActor {
	ctx, cancel := context.WithCancel(context.Background())

	actor := &DefaultActor{
		id:        id,
		mailbox:   make(chan interface{}, bufferSize),
		ctx:       ctx,
		cancel:    cancel,
		receiver:  receiver,
		stateData: make(map[string]interface{}),
	}

	actor.wg.Add(1)
	go actor.processMessages()

	return actor
}

func (a *DefaultActor) processMessages() {
	defer a.wg.Done()

	for {
		select {
		case msg := <-a.mailbox:
			err := a.receiver(a.ctx, msg)
			if err != nil {
				a.setLastError(err)

			}
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *DefaultActor) Receive(ctx context.Context, message interface{}) error {
	a.mu.RLock()
	if a.stopped {
		a.mu.RUnlock()
		return ErrActorStopped
	}
	a.mu.RUnlock()

	select {
	case a.mailbox <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-a.ctx.Done():
		return ErrActorStopped
	default:

		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case a.mailbox <- message:
			timer.Stop()
			return nil
		case <-timer.C:
			return ErrMailboxFull
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-a.ctx.Done():
			timer.Stop()
			return ErrActorStopped
		}
	}
}

func (a *DefaultActor) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.stopped {
		return nil
	}

	a.stopped = true
	a.cancel()
	a.wg.Wait()
	close(a.mailbox)
	return nil
}

func (a *DefaultActor) ID() string {
	return a.id
}

func (a *DefaultActor) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return !a.stopped
}

func (a *DefaultActor) SetState(key string, value interface{}) {
	a.stateDataMu.Lock()
	defer a.stateDataMu.Unlock()
	a.stateData[key] = value
}

func (a *DefaultActor) GetState(key string) (interface{}, bool) {
	a.stateDataMu.RLock()
	defer a.stateDataMu.RUnlock()
	val, ok := a.stateData[key]
	return val, ok
}

func (a *DefaultActor) setLastError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastError = err
}

func (a *DefaultActor) GetLastError() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastError
}

type ActorRefImpl struct {
	actor Actor
}

func NewActorRef(actor Actor) ActorRef {
	return &ActorRefImpl{actor: actor}
}

func (r *ActorRefImpl) Send(ctx context.Context, message interface{}) error {
	return r.actor.Receive(ctx, message)
}

func (r *ActorRefImpl) ID() string {
	return r.actor.ID()
}

func (r *ActorRefImpl) IsRunning() bool {
	return r.actor.IsRunning()
}
