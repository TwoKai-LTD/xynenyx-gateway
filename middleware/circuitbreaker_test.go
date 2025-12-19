package middleware

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	// Test closed state - should allow calls
	err := cb.Call(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error in closed state, got %v", err)
	}
	if cb.GetState() != StateClosed {
		t.Errorf("Expected closed state, got %v", cb.GetState())
	}

	// Test failures leading to open state
	for i := 0; i < 3; i++ {
		err := cb.Call(func() error {
			return errors.New("test error")
		})
		if err == nil {
			t.Errorf("Expected error on failure %d", i+1)
		}
	}

	// Circuit should be open now
	if cb.GetState() != StateOpen {
		t.Errorf("Expected open state after failures, got %v", cb.GetState())
	}

	// Test open state - should reject calls immediately
	err = cb.Call(func() error {
		return nil
	})
	if err == nil {
		t.Error("Expected error when circuit is open")
	}

	// Wait for timeout
	time.Sleep(2 * time.Second)

	// Test half-open state - should allow one call
	err = cb.Call(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error in half-open state, got %v", err)
	}

	// Circuit should be closed after successful call
	if cb.GetState() != StateClosed {
		t.Errorf("Expected closed state after recovery, got %v", cb.GetState())
	}
}

func TestCircuitBreakerManager(t *testing.T) {
	cbm := NewCircuitBreakerManager(2, 1*time.Second)

	// Test different services
	service1 := "agent"
	service2 := "rag"

	// Service 1 failures
	breaker1 := cbm.GetBreaker(service1)
	for i := 0; i < 2; i++ {
		breaker1.Call(func() error {
			return errors.New("error")
		})
	}

	// Service 1 should be open
	if cbm.GetState(service1) != StateOpen {
		t.Errorf("Expected service1 to be open")
	}

	// Service 2 should still be closed
	if cbm.GetState(service2) != StateClosed {
		t.Errorf("Expected service2 to be closed")
	}
}

