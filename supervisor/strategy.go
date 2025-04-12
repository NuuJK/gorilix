package supervisor

import (
	"context"
	"math/rand"
	"time"
)

type RestartStrategy int

const (
	OneForOne RestartStrategy = iota

	OneForAll

	RestForOne
)

type BackoffType int

const (
	NoBackoff BackoffType = iota

	LinearBackoff

	ExponentialBackoff

	JitteredExponentialBackoff
)

type CircuitBreakerState int

const (
	Closed CircuitBreakerState = iota
	Open
	HalfOpen
)

type Strategy interface {
	HandleFailure(ctx context.Context, childID string, err error) ([]string, error)

	MaxRestarts() int

	TimeInterval() int

	Type() RestartStrategy

	BackoffStrategy() BackoffType

	CalculateBackoff(restarts int) time.Duration

	ShouldTerminateOnFailure() bool

	CircuitBreaker() CircuitBreaker
}

type CircuitBreaker interface {
	GetState() CircuitBreakerState
	RecordFailure() bool
	RecordSuccess()
	Reset()
	ShouldAllow() bool
	TripThreshold() int
	FailureWindow() time.Duration
	ResetTimeout() time.Duration
}

type DefaultCircuitBreaker struct {
	state              CircuitBreakerState
	failures           int
	tripThreshold      int
	failureWindow      time.Duration
	resetTimeout       time.Duration
	lastFailure        time.Time
	lastStateChange    time.Time
	consecutiveSuccess int
	successThreshold   int
}

func NewCircuitBreaker(tripThreshold int, failureWindow, resetTimeout time.Duration, successThreshold int) CircuitBreaker {
	return &DefaultCircuitBreaker{
		state:              Closed,
		failures:           0,
		tripThreshold:      tripThreshold,
		failureWindow:      failureWindow,
		resetTimeout:       resetTimeout,
		lastFailure:        time.Time{},
		lastStateChange:    time.Now(),
		consecutiveSuccess: 0,
		successThreshold:   successThreshold,
	}
}

func (cb *DefaultCircuitBreaker) GetState() CircuitBreakerState {
	now := time.Now()

	if cb.state == Open && now.Sub(cb.lastStateChange) > cb.resetTimeout {
		cb.state = HalfOpen
		cb.lastStateChange = now
	}

	return cb.state
}

func (cb *DefaultCircuitBreaker) RecordFailure() bool {
	now := time.Now()

	if !cb.lastFailure.IsZero() && now.Sub(cb.lastFailure) > cb.failureWindow {
		cb.failures = 0
	}

	cb.failures++
	cb.lastFailure = now
	cb.consecutiveSuccess = 0

	if cb.state == Closed && cb.failures >= cb.tripThreshold {
		cb.state = Open
		cb.lastStateChange = now
		return true
	}

	if cb.state == HalfOpen {
		cb.state = Open
		cb.lastStateChange = now
	}

	return false
}

func (cb *DefaultCircuitBreaker) RecordSuccess() {
	if cb.state == HalfOpen {
		cb.consecutiveSuccess++
		if cb.consecutiveSuccess >= cb.successThreshold {
			cb.Reset()
		}
	}
}

func (cb *DefaultCircuitBreaker) Reset() {
	cb.state = Closed
	cb.failures = 0
	cb.lastStateChange = time.Now()
	cb.consecutiveSuccess = 0
}

func (cb *DefaultCircuitBreaker) ShouldAllow() bool {
	state := cb.GetState()

	switch state {
	case Closed:
		return true
	case Open:
		return false
	case HalfOpen:
		return true
	default:
		return false
	}
}

func (cb *DefaultCircuitBreaker) TripThreshold() int {
	return cb.tripThreshold
}

func (cb *DefaultCircuitBreaker) FailureWindow() time.Duration {
	return cb.failureWindow
}

func (cb *DefaultCircuitBreaker) ResetTimeout() time.Duration {
	return cb.resetTimeout
}

type DefaultStrategy struct {
	strategyType           RestartStrategy
	maxRestarts            int
	timeInterval           int
	backoffStrategy        BackoffType
	baseBackoff            time.Duration
	maxBackoff             time.Duration
	terminateOnMaxRestarts bool
	circuitBreaker         CircuitBreaker
	jitterFactor           float64
}

type StrategyOptions struct {
	BackoffType            BackoffType
	BaseBackoff            time.Duration
	MaxBackoff             time.Duration
	TerminateOnMaxRestarts bool
	CircuitBreakerOptions  *CircuitBreakerOptions
	JitterFactor           float64
}

type CircuitBreakerOptions struct {
	Enabled          bool
	TripThreshold    int
	FailureWindow    time.Duration
	ResetTimeout     time.Duration
	SuccessThreshold int
}

func DefaultStrategyOptions() StrategyOptions {
	return StrategyOptions{
		BackoffType:            NoBackoff,
		BaseBackoff:            100 * time.Millisecond,
		MaxBackoff:             1 * time.Minute,
		TerminateOnMaxRestarts: true,
		CircuitBreakerOptions: &CircuitBreakerOptions{
			Enabled:          false,
			TripThreshold:    5,
			FailureWindow:    1 * time.Minute,
			ResetTimeout:     5 * time.Second,
			SuccessThreshold: 2,
		},
		JitterFactor: 0.2,
	}
}

func NewStrategy(strategyType RestartStrategy, maxRestarts, timeInterval int) Strategy {
	options := DefaultStrategyOptions()
	return NewStrategyWithOptions(strategyType, maxRestarts, timeInterval, options)
}

func NewStrategyWithOptions(strategyType RestartStrategy, maxRestarts, timeInterval int, options StrategyOptions) Strategy {
	var cb CircuitBreaker
	if options.CircuitBreakerOptions != nil && options.CircuitBreakerOptions.Enabled {
		cbOpts := options.CircuitBreakerOptions
		cb = NewCircuitBreaker(
			cbOpts.TripThreshold,
			cbOpts.FailureWindow,
			cbOpts.ResetTimeout,
			cbOpts.SuccessThreshold,
		)
	} else {

		cb = NewCircuitBreaker(9999, 24*time.Hour, 1*time.Millisecond, 1)
	}

	return &DefaultStrategy{
		strategyType:           strategyType,
		maxRestarts:            maxRestarts,
		timeInterval:           timeInterval,
		backoffStrategy:        options.BackoffType,
		baseBackoff:            options.BaseBackoff,
		maxBackoff:             options.MaxBackoff,
		terminateOnMaxRestarts: options.TerminateOnMaxRestarts,
		circuitBreaker:         cb,
		jitterFactor:           options.JitterFactor,
	}
}

func (s *DefaultStrategy) HandleFailure(ctx context.Context, childID string, err error) ([]string, error) {

	s.circuitBreaker.RecordFailure()

	switch s.strategyType {
	case OneForOne:
		return []string{childID}, nil
	case OneForAll:
		return nil, nil
	case RestForOne:
		return nil, nil
	default:
		return []string{childID}, nil
	}
}

func (s *DefaultStrategy) MaxRestarts() int {
	return s.maxRestarts
}

func (s *DefaultStrategy) TimeInterval() int {
	return s.timeInterval
}

func (s *DefaultStrategy) Type() RestartStrategy {
	return s.strategyType
}

func (s *DefaultStrategy) BackoffStrategy() BackoffType {
	return s.backoffStrategy
}

func (s *DefaultStrategy) CalculateBackoff(restarts int) time.Duration {
	if restarts <= 0 || s.backoffStrategy == NoBackoff {
		return 0
	}

	var backoff time.Duration

	switch s.backoffStrategy {
	case LinearBackoff:
		backoff = s.baseBackoff * time.Duration(restarts)
	case ExponentialBackoff:
		backoff = s.baseBackoff
		for i := 0; i < restarts-1; i++ {
			backoff *= 2
		}
	case JitteredExponentialBackoff:

		backoff = s.baseBackoff
		for i := 0; i < restarts-1; i++ {
			backoff *= 2
		}

		jitter := float64(backoff) * s.jitterFactor * (rand.Float64()*2 - 1)
		backoffWithJitter := backoff + time.Duration(jitter)

		if backoffWithJitter < 0 {
			backoff = 0
		} else {
			backoff = backoffWithJitter
		}
	default:
		return 0
	}

	if backoff > s.maxBackoff {
		backoff = s.maxBackoff
	}

	return backoff
}

func (s *DefaultStrategy) ShouldTerminateOnFailure() bool {
	return s.terminateOnMaxRestarts
}

func (s *DefaultStrategy) CircuitBreaker() CircuitBreaker {
	return s.circuitBreaker
}
