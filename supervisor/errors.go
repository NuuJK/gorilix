package supervisor

import "errors"

var (
	ErrTooManyRestarts = errors.New("too many restarts in the time interval")

	ErrSupervisorStopped = errors.New("supervisor is stopped")

	ErrInvalidStrategy = errors.New("invalid supervision strategy")

	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)
