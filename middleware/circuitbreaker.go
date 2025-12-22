package middleware

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	failures     int           // Consecutive failures
	maxFailures  int           // Failures before opening
	timeout      time.Duration // Timeout before attempting half-open
	lastFailTime time.Time     // Time of last failure
	state        CircuitState  // Current state
	mu           sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       StateClosed,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	state := cb.state
	cb.mu.Unlock()

	// Check if circuit is open
	if state == StateOpen {
		// Check if timeout has passed to try half-open
		cb.mu.Lock()
		timeSinceFail := time.Since(cb.lastFailTime)
		// Reduce timeout check - if lastFailTime is zero or timeout passed, allow retry
		if cb.lastFailTime.IsZero() || timeSinceFail >= cb.timeout {
			cb.state = StateHalfOpen
			cb.failures = 0 // Reset failures when transitioning to half-open
			cb.mu.Unlock()
		} else {
			cb.mu.Unlock()
			// Return error - circuit breaker is still open
			return errors.New("circuit breaker is open")
		}
	}

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// Failure
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.state == StateHalfOpen {
			// Half-open failed, go back to open
			cb.state = StateOpen
		} else if cb.failures >= cb.maxFailures {
			// Too many failures, open circuit
			cb.state = StateOpen
		}
	} else {
		// Success
		if cb.state == StateHalfOpen {
			// Half-open succeeded, close circuit
			cb.state = StateClosed
			cb.failures = 0
		} else {
			// Reset failure count on success
			cb.failures = 0
		}
	}

	return err
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.lastFailTime = time.Time{}
}

// ForceHalfOpen forces the circuit breaker to half-open state (for testing recovery)
func (cb *CircuitBreaker) ForceHalfOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateHalfOpen
	cb.lastFailTime = time.Time{} // Reset timeout so it can try immediately
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// CircuitBreakerManager manages circuit breakers for multiple services
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	maxFailures int
	timeout     time.Duration
	mu          sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(maxFailures int, timeout time.Duration) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers:    make(map[string]*CircuitBreaker),
		maxFailures: maxFailures,
		timeout:     timeout,
	}
}

// GetBreaker gets or creates a circuit breaker for a service
func (cbm *CircuitBreakerManager) GetBreaker(service string) *CircuitBreaker {
	cbm.mu.RLock()
	breaker, exists := cbm.breakers[service]
	cbm.mu.RUnlock()

	if !exists {
		cbm.mu.Lock()
		// Double-check after acquiring write lock
		breaker, exists = cbm.breakers[service]
		if !exists {
			breaker = NewCircuitBreaker(cbm.maxFailures, cbm.timeout)
			cbm.breakers[service] = breaker
		}
		cbm.mu.Unlock()
	}

	return breaker
}

// GetState returns the state of a service's circuit breaker
func (cbm *CircuitBreakerManager) GetState(service string) CircuitState {
	breaker := cbm.GetBreaker(service)
	return breaker.GetState()
}

// Reset resets a service's circuit breaker
func (cbm *CircuitBreakerManager) Reset(service string) {
	breaker := cbm.GetBreaker(service)
	state := breaker.GetState()
	// If open or half-open, force to closed state to allow immediate requests
	if state == StateOpen || state == StateHalfOpen {
		// Force to closed instead of half-open for immediate availability
		breaker.Reset() // This sets to closed
	} else {
		// If already closed, just ensure it stays closed
		breaker.Reset()
	}
}

// GetAllStates returns the state of all circuit breakers
func (cbm *CircuitBreakerManager) GetAllStates() map[string]CircuitState {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	states := make(map[string]CircuitState)
	for service, breaker := range cbm.breakers {
		states[service] = breaker.GetState()
	}
	return states
}

