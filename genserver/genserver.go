package genserver

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kleeedolinux/gorilix/actor"
)

type InitFunc func(ctx context.Context, args interface{}) (interface{}, error)

type TerminateFunc func(ctx context.Context, reason error, state interface{})

type CallHandler func(ctx context.Context, message interface{}, state interface{}) (interface{}, interface{}, error)

type CastHandler func(ctx context.Context, message interface{}, state interface{}) (interface{}, error)

type InfoHandler func(ctx context.Context, message interface{}, state interface{}) (interface{}, error)

type CallMessage struct {
	Payload   interface{}
	ReplyTo   chan<- interface{}
	From      string
	Timeout   time.Duration
	Timestamp time.Time
	ID        string
}

type CastMessage struct {
	Payload   interface{}
	From      string
	Timestamp time.Time
	ID        string
}

type Options struct {
	InitFunc      InitFunc
	TerminateFunc TerminateFunc
	CallHandler   CallHandler
	CastHandler   CastHandler
	InfoHandler   InfoHandler
	BufferSize    int
	InitArgs      interface{}
	Name          string
}

type GenServer struct {
	*actor.DefaultActor
	options     Options
	state       interface{}
	initCalled  bool
	terminateCh chan struct{}
	mu          sync.RWMutex
}

func New(id string, options Options) *GenServer {
	if options.BufferSize <= 0 {
		options.BufferSize = 100
	}

	gs := &GenServer{
		options:     options,
		terminateCh: make(chan struct{}),
	}

	gs.DefaultActor = actor.NewActor(id, gs.processMessage, options.BufferSize)
	return gs
}

func Start(id string, options Options) (*GenServer, actor.ActorRef, error) {
	gs := New(id, options)
	ref := actor.NewActorRef(gs)

	if options.InitFunc != nil {
		ctx := context.Background()
		var err error
		gs.state, err = options.InitFunc(ctx, options.InitArgs)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize GenServer: %w", err)
		}
		gs.initCalled = true
	}

	return gs, ref, nil
}

func (g *GenServer) processMessage(ctx context.Context, msg interface{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var newState interface{}
	var err error

	switch m := msg.(type) {
	case *CallMessage:
		var reply interface{}
		reply, newState, err = g.handleCall(ctx, m)
		if m.ReplyTo != nil {
			select {
			case m.ReplyTo <- reply:
			default:

			}
		}
	case *CastMessage:
		newState, err = g.handleCast(ctx, m)
	case *actor.MonitorMessage:

		newState, err = g.handleInfo(ctx, m)
	default:

		newState, err = g.handleInfo(ctx, msg)
	}

	if err != nil {
		return err
	}

	if newState != nil {
		g.state = newState
	}

	return nil
}

func (g *GenServer) handleCall(ctx context.Context, msg *CallMessage) (interface{}, interface{}, error) {
	if g.options.CallHandler != nil {
		return g.options.CallHandler(ctx, msg.Payload, g.state)
	}
	return nil, g.state, nil
}

func (g *GenServer) handleCast(ctx context.Context, msg *CastMessage) (interface{}, error) {
	if g.options.CastHandler != nil {
		return g.options.CastHandler(ctx, msg.Payload, g.state)
	}
	return g.state, nil
}

func (g *GenServer) handleInfo(ctx context.Context, msg interface{}) (interface{}, error) {
	if g.options.InfoHandler != nil {
		return g.options.InfoHandler(ctx, msg, g.state)
	}
	return g.state, nil
}

func (g *GenServer) Stop() error {
	g.mu.Lock()
	terminateFunc := g.options.TerminateFunc
	state := g.state
	g.mu.Unlock()

	if terminateFunc != nil {
		terminateFunc(context.Background(), nil, state)
	}

	close(g.terminateCh)
	return g.DefaultActor.Stop()
}

func MakeCallSync(ctx context.Context, to actor.ActorRef, payload interface{}, timeout time.Duration) (interface{}, error) {
	replyCh := make(chan interface{}, 1)

	callMsg := &CallMessage{
		Payload:   payload,
		ReplyTo:   replyCh,
		Timestamp: time.Now(),
		Timeout:   timeout,
		ID:        fmt.Sprintf("call-%d", time.Now().UnixNano()),
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := to.Send(ctx, callMsg)
	if err != nil {
		return nil, err
	}

	select {
	case reply := <-replyCh:
		return reply, nil
	case <-callCtx.Done():
		return nil, callCtx.Err()
	}
}

func MakeCast(ctx context.Context, to actor.ActorRef, payload interface{}) error {
	castMsg := &CastMessage{
		Payload:   payload,
		Timestamp: time.Now(),
		ID:        fmt.Sprintf("cast-%d", time.Now().UnixNano()),
	}

	return to.Send(ctx, castMsg)
}
