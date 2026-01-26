package lsp

import (
	"sync"
	"time"
)

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half-open"
)

type CircuitConfig struct {
	FailureThreshold int
	SuccessThreshold int
	OpenTimeout      time.Duration
	HalfOpenMaxCalls int
}

func DefaultCircuitConfig() CircuitConfig {
	return CircuitConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		OpenTimeout:      30 * time.Second,
		HalfOpenMaxCalls: 1,
	}
}

type CircuitBreaker struct {
	config        CircuitConfig
	state         CircuitState
	failures      int
	successes     int
	lastFailure   time.Time
	halfOpenCalls int
	mu            sync.RWMutex
}

func NewCircuitBreaker(config CircuitConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		if time.Since(cb.lastFailure) >= cb.config.OpenTimeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenCalls = 0
			cb.successes = 0
			return cb.allowHalfOpen()
		}
		return false

	case CircuitHalfOpen:
		return cb.allowHalfOpen()
	}

	return false
}

func (cb *CircuitBreaker) allowHalfOpen() bool {
	if cb.halfOpenCalls < cb.config.HalfOpenMaxCalls {
		cb.halfOpenCalls++
		return true
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failures = 0

	case CircuitHalfOpen:
		cb.successes++
		cb.halfOpenCalls--
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successes = 0
		}
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = CircuitOpen
		}

	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.halfOpenCalls = 0
		cb.successes = 0
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCalls = 0
}

func (cb *CircuitBreaker) Stats() CircuitStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitStats{
		State:         cb.state,
		Failures:      cb.failures,
		Successes:     cb.successes,
		LastFailure:   cb.lastFailure,
		HalfOpenCalls: cb.halfOpenCalls,
	}
}

type CircuitStats struct {
	State         CircuitState `json:"state"`
	Failures      int          `json:"failures"`
	Successes     int          `json:"successes"`
	LastFailure   time.Time    `json:"last_failure,omitempty"`
	HalfOpenCalls int          `json:"half_open_calls"`
}
