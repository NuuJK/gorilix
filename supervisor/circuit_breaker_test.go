package supervisor

import (
	"testing"
	"time"
)

func TestCircuitBreaker(t *testing.T) {

	cb := NewCircuitBreaker(3, 100*time.Millisecond, 50*time.Millisecond, 2)

	if state := cb.GetState(); state != Closed {
		t.Errorf("Expected initial state Closed, got %v", state)
	}

	if !cb.ShouldAllow() {
		t.Error("Expected circuit breaker to allow operations initially")
	}

	cb.RecordFailure()
	cb.RecordFailure()

	if state := cb.GetState(); state != Closed {
		t.Errorf("Expected state Closed after 2 failures, got %v", state)
	}

	cb.RecordFailure()

	if state := cb.GetState(); state != Open {
		t.Errorf("Expected state Open after 3 failures, got %v", state)
	}

	if cb.ShouldAllow() {
		t.Error("Expected circuit breaker to deny operations when open")
	}

	time.Sleep(60 * time.Millisecond)

	if state := cb.GetState(); state != HalfOpen {
		t.Errorf("Expected state HalfOpen after timeout, got %v", state)
	}

	if !cb.ShouldAllow() {
		t.Error("Expected circuit breaker to allow operations in half-open state")
	}

	cb.RecordFailure()

	if state := cb.GetState(); state != Open {
		t.Errorf("Expected state Open after failure in half-open, got %v", state)
	}

	time.Sleep(60 * time.Millisecond)

	cb.RecordSuccess()

	if state := cb.GetState(); state != HalfOpen {
		t.Errorf("Expected state HalfOpen after 1 success, got %v", state)
	}

	cb.RecordSuccess()

	if state := cb.GetState(); state != Closed {
		t.Errorf("Expected state Closed after 2 successes, got %v", state)
	}

	cb.RecordFailure()
	cb.Reset()

	if state := cb.GetState(); state != Closed {
		t.Errorf("Expected state Closed after reset, got %v", state)
	}
}

func TestCircuitBreakerFailureWindow(t *testing.T) {

	cb := NewCircuitBreaker(3, 50*time.Millisecond, 100*time.Millisecond, 1)

	cb.RecordFailure()
	cb.RecordFailure()

	if state := cb.GetState(); state != Closed {
		t.Errorf("Expected state Closed after 2 failures, got %v", state)
	}

	time.Sleep(60 * time.Millisecond)

	cb.RecordFailure()

	if state := cb.GetState(); state != Closed {
		t.Errorf("Expected state Closed after window expiry, got %v", state)
	}

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if state := cb.GetState(); state != Open {
		t.Errorf("Expected state Open after 3 quick failures, got %v", state)
	}
}
