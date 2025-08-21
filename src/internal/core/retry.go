package core

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/howmanysmall/relay/src/internal/config"
)

// RetryManager handles retry logic with exponential backoff.
type RetryManager struct {
	config *config.RetryConfig
}

// RetryableError wraps an error with retry information.
type RetryableError struct {
	Err       error
	Retryable bool
	Fatal     bool
}

func (re *RetryableError) Error() string {
	return re.Err.Error()
}

func (re *RetryableError) Unwrap() error {
	return re.Err
}

// NewRetryManager creates a new retry manager with the given configuration.
func NewRetryManager(cfg *config.RetryConfig) *RetryManager {
	if cfg == nil {
		cfg = &config.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
			Backoff:      string(config.BackoffExponential),
		}
	}

	return &RetryManager{
		config: cfg,
	}
}

// ExecuteWithRetry executes an operation with retry logic and exponential backoff.
func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= rm.config.MaxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		retryableErr := &RetryableError{}

		ok := errors.As(err, &retryableErr)
		if ok {
			if retryableErr.Fatal {
				return fmt.Errorf("fatal error on attempt %d: %w", attempt, err)
			}

			if !retryableErr.Retryable {
				return fmt.Errorf("non-retryable error on attempt %d: %w", attempt, err)
			}
		}

		if attempt == rm.config.MaxAttempts {
			break
		}

		delay := rm.calculateDelay(attempt)

		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled after %d attempts: %w", attempt, ctx.Err())
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("operation failed after %d attempts, last error: %w", rm.config.MaxAttempts, lastErr)
}

func (rm *RetryManager) calculateDelay(attempt int) time.Duration {
	switch config.BackoffStrategy(rm.config.Backoff) {
	case config.BackoffFixed:
		return rm.config.InitialDelay
	case config.BackoffLinear:
		delay := time.Duration(int64(rm.config.InitialDelay) * int64(attempt))
		if delay > rm.config.MaxDelay {
			return rm.config.MaxDelay
		}

		return delay
	case config.BackoffExponential:
		fallthrough
	default:
		delay := time.Duration(float64(rm.config.InitialDelay) * math.Pow(rm.config.Multiplier, float64(attempt-1)))
		if delay > rm.config.MaxDelay {
			return rm.config.MaxDelay
		}

		return delay
	}
}

// NewRetryableError wraps an error indicating whether it's retryable.
func NewRetryableError(err error, retryable bool) *RetryableError {
	return &RetryableError{
		Err:       err,
		Retryable: retryable,
		Fatal:     false,
	}
}

// NewFatalError wraps an error as fatal (non-retryable) and stops further retries.
func NewFatalError(err error) *RetryableError {
	return &RetryableError{
		Err:       err,
		Retryable: false,
		Fatal:     true,
	}
}

// ClassifyError analyzes an error and returns a RetryableError with appropriate classification.
func ClassifyError(err error) *RetryableError {
	if err == nil {
		return nil
	}

	// Network-related errors are typically retryable
	if isNetworkError(err) {
		return NewRetryableError(err, true)
	}

	// Permission errors are typically not retryable
	if isPermissionError(err) {
		return NewRetryableError(err, false)
	}

	// Disk full errors are typically not retryable immediately
	if isDiskFullError(err) {
		return NewRetryableError(err, false)
	}

	// Context cancellation is fatal
	if isContextError(err) {
		return NewFatalError(err)
	}

	// File not found errors are typically not retryable
	if isNotFoundError(err) {
		return NewRetryableError(err, false)
	}

	// IO errors might be temporarily retryable
	if isIOError(err) {
		return NewRetryableError(err, true)
	}

	// Default to retryable for unknown errors
	return NewRetryableError(err, true)
}

func isNetworkError(err error) bool {
	// Check for common network error patterns
	errStr := err.Error()
	networkPatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"network is unreachable",
		"temporary failure",
		"no route to host",
	}

	for _, pattern := range networkPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func isPermissionError(err error) bool {
	errStr := err.Error()
	permissionPatterns := []string{
		"permission denied",
		"access denied",
		"operation not permitted",
		"insufficient privileges",
	}

	for _, pattern := range permissionPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func isDiskFullError(err error) bool {
	errStr := err.Error()
	diskFullPatterns := []string{
		"no space left on device",
		"disk full",
		"insufficient space",
		"not enough space",
	}

	for _, pattern := range diskFullPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func isNotFoundError(err error) bool {
	errStr := err.Error()
	notFoundPatterns := []string{
		"no such file or directory",
		"file not found",
		"path not found",
		"not found",
	}

	for _, pattern := range notFoundPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func isIOError(err error) bool {
	errStr := err.Error()
	ioPatterns := []string{
		"i/o error",
		"input/output error",
		"read error",
		"write error",
		"broken pipe",
		"connection broken",
	}

	for _, pattern := range ioPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr ||
		(len(str) > len(substr) &&
			(str[:len(substr)] == substr ||
				str[len(str)-len(substr):] == substr ||
				indexContains(str, substr))))
}

func indexContains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
