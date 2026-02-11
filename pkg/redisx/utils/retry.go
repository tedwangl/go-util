package utils

import (
	"context"
	"fmt"
	"time"

	redisxerrors "github.com/tedwangl/go-util/pkg/redisx/errors"
)

type RetryConfig struct {
	MaxAttempts      int
	InitialDelay     time.Duration
	MaxDelay         time.Duration
	BackoffFactor    float64
	RetryableChecker func(error) bool
}

func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:      3,
		InitialDelay:     100 * time.Millisecond,
		MaxDelay:         5 * time.Second,
		BackoffFactor:    2.0,
		RetryableChecker: redisxerrors.IsRetryableError,
	}
}

type RetryFunc func(attempt int) error

func Retry(ctx context.Context, config *RetryConfig, fn RetryFunc) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := fn(attempt)
		if err == nil {
			return nil
		}

		lastErr = err

		if config.RetryableChecker != nil && !config.RetryableChecker(err) {
			return fmt.Errorf("non-retryable error on attempt %d: %w", attempt, err)
		}

		if attempt < config.MaxAttempts {
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry canceled: %w", ctx.Err())
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * config.BackoffFactor)
				if delay > config.MaxDelay {
					delay = config.MaxDelay
				}
			}
		}
	}

	return redisxerrors.NewRetryError(config.MaxAttempts, "all retry attempts failed", lastErr)
}

func RetryWithBackoff(ctx context.Context, maxAttempts int, initialDelay, maxDelay time.Duration, fn RetryFunc) error {
	config := &RetryConfig{
		MaxAttempts:      maxAttempts,
		InitialDelay:     initialDelay,
		MaxDelay:         maxDelay,
		BackoffFactor:    2.0,
		RetryableChecker: redisxerrors.IsRetryableError,
	}
	return Retry(ctx, config, fn)
}

func RetryFixedDelay(ctx context.Context, maxAttempts int, delay time.Duration, fn RetryFunc) error {
	config := &RetryConfig{
		MaxAttempts:      maxAttempts,
		InitialDelay:     delay,
		MaxDelay:         delay,
		BackoffFactor:    1.0,
		RetryableChecker: redisxerrors.IsRetryableError,
	}
	return Retry(ctx, config, fn)
}

func RetryUntilSuccess(ctx context.Context, interval time.Duration, fn RetryFunc) error {
	attempt := 0
	for {
		attempt++
		err := fn(attempt)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("retry until success canceled after %d attempts: %w", attempt, ctx.Err())
		case <-time.After(interval):
		}
	}
}

type RetryableOperation[T any] func(attempt int) (T, error)

func RetryOperation[T any](ctx context.Context, config *RetryConfig, fn RetryableOperation[T]) (T, error) {
	var result T
	err := Retry(ctx, config, func(attempt int) error {
		var err error
		result, err = fn(attempt)
		return err
	})
	return result, err
}

func RetryOperationWithBackoff[T any](ctx context.Context, maxAttempts int, initialDelay, maxDelay time.Duration, fn RetryableOperation[T]) (T, error) {
	config := &RetryConfig{
		MaxAttempts:      maxAttempts,
		InitialDelay:     initialDelay,
		MaxDelay:         maxDelay,
		BackoffFactor:    2.0,
		RetryableChecker: redisxerrors.IsRetryableError,
	}
	return RetryOperation(ctx, config, fn)
}

type CircuitBreakerConfig struct {
	MaxFailures     int
	ResetTimeout    time.Duration
	HalfOpenTimeout time.Duration
}

type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	config      *CircuitBreakerConfig
	state       CircuitBreakerState
	failures    int
	lastFailure time.Time
	lastAttempt time.Time
}

func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = &CircuitBreakerConfig{
			MaxFailures:     5,
			ResetTimeout:    30 * time.Second,
			HalfOpenTimeout: 5 * time.Second,
		}
	}
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.config.ResetTimeout {
			cb.state = StateHalfOpen
			cb.lastAttempt = time.Now()
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()

		if cb.failures >= cb.config.MaxFailures {
			cb.state = StateOpen
		}

		return err
	}

	cb.failures = 0
	cb.state = StateClosed
	return nil
}

func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.state = StateClosed
	cb.failures = 0
	cb.lastFailure = time.Time{}
}
