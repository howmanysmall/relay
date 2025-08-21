package core

import (
	"fmt"
	"time"
)

// ErrorCategory represents different types of errors
type ErrorCategory int

// Error categories for classification
const (
	ErrorCategoryUnknown ErrorCategory = iota
	ErrorCategoryNetwork
	ErrorCategoryPermission
	ErrorCategoryDisk
	ErrorCategoryCorruption
	ErrorCategoryConfiguration
	ErrorCategoryCancellation
)

func (ec ErrorCategory) String() string {
	switch ec {
	case ErrorCategoryNetwork:
		return "Network"
	case ErrorCategoryPermission:
		return "Permission"
	case ErrorCategoryDisk:
		return "Disk"
	case ErrorCategoryCorruption:
		return "Corruption"
	case ErrorCategoryConfiguration:
		return "Configuration"
	case ErrorCategoryCancellation:
		return "Cancellation"
	default:
		return "Unknown"
	}
}

// SyncError represents a detailed synchronization error
type SyncError struct {
	Category    ErrorCategory `json:"category"`
	Operation   string        `json:"operation"`
	Path        string        `json:"path"`
	Message     string        `json:"message"`
	Underlying  error         `json:"-"`
	Timestamp   time.Time     `json:"timestamp"`
	Recoverable bool          `json:"recoverable"`
	Suggestion  string        `json:"suggestion"`
}

func (se *SyncError) Error() string {
	return fmt.Sprintf("[%s] %s: %s (path: %s)", se.Category, se.Operation, se.Message, se.Path)
}

func (se *SyncError) Unwrap() error {
	return se.Underlying
}

// NewNetworkError creates a new network-related error.
func NewNetworkError(operation, path string, err error) *SyncError {
	return &SyncError{
		Category:    ErrorCategoryNetwork,
		Operation:   operation,
		Path:        path,
		Message:     err.Error(),
		Underlying:  err,
		Timestamp:   time.Now(),
		Recoverable: true,
		Suggestion:  "Check network connectivity and try again",
	}
}

// NewPermissionError creates a new permission-related error.
func NewPermissionError(operation, path string, err error) *SyncError {
	return &SyncError{
		Category:    ErrorCategoryPermission,
		Operation:   operation,
		Path:        path,
		Message:     err.Error(),
		Underlying:  err,
		Timestamp:   time.Now(),
		Recoverable: false,
		Suggestion:  "Check file permissions or run with elevated privileges",
	}
}

// NewDiskError creates a new disk space-related error.
func NewDiskError(operation, path string, err error) *SyncError {
	return &SyncError{
		Category:    ErrorCategoryDisk,
		Operation:   operation,
		Path:        path,
		Message:     err.Error(),
		Underlying:  err,
		Timestamp:   time.Now(),
		Recoverable: false,
		Suggestion:  "Free up disk space and try again",
	}
}

// NewCorruptionError creates a new data corruption error.
func NewCorruptionError(operation, path string, err error) *SyncError {
	return &SyncError{
		Category:    ErrorCategoryCorruption,
		Operation:   operation,
		Path:        path,
		Message:     err.Error(),
		Underlying:  err,
		Timestamp:   time.Now(),
		Recoverable: false,
		Suggestion:  "Verify file integrity and restore from backup if necessary",
	}
}

// NewConfigurationError creates a new configuration-related error.
func NewConfigurationError(operation, path string, err error) *SyncError {
	return &SyncError{
		Category:    ErrorCategoryConfiguration,
		Operation:   operation,
		Path:        path,
		Message:     err.Error(),
		Underlying:  err,
		Timestamp:   time.Now(),
		Recoverable: false,
		Suggestion:  "Check configuration file syntax and settings",
	}
}

// NewCancellationError creates a new cancellation error.
func NewCancellationError(operation, path string, err error) *SyncError {
	return &SyncError{
		Category:    ErrorCategoryCancellation,
		Operation:   operation,
		Path:        path,
		Message:     err.Error(),
		Underlying:  err,
		Timestamp:   time.Now(),
		Recoverable: false,
		Suggestion:  "Operation was cancelled by user or timeout",
	}
}

// ErrorHandler manages error collection and reporting
type ErrorHandler struct {
	errors    []*SyncError
	maxErrors int
}

// NewErrorHandler creates a new error handler with the specified maximum error count.
func NewErrorHandler(maxErrors int) *ErrorHandler {
	if maxErrors <= 0 {
		maxErrors = 1000 // Default limit
	}

	return &ErrorHandler{
		errors:    make([]*SyncError, 0),
		maxErrors: maxErrors,
	}
}

// AddError adds an error to the handler.
func (eh *ErrorHandler) AddError(err *SyncError) {
	if len(eh.errors) >= eh.maxErrors {
		// Remove oldest error to make room
		eh.errors = eh.errors[1:]
	}

	eh.errors = append(eh.errors, err)
}

// GetErrors returns all collected errors.
func (eh *ErrorHandler) GetErrors() []*SyncError {
	result := make([]*SyncError, len(eh.errors))
	copy(result, eh.errors)

	return result
}

// GetErrorsByCategory returns errors of a specific category.
func (eh *ErrorHandler) GetErrorsByCategory(category ErrorCategory) []*SyncError {
	var result []*SyncError

	for _, err := range eh.errors {
		if err.Category == category {
			result = append(result, err)
		}
	}

	return result
}

// GetRecoverableErrors returns errors that can be retried.
func (eh *ErrorHandler) GetRecoverableErrors() []*SyncError {
	var result []*SyncError

	for _, err := range eh.errors {
		if err.Recoverable {
			result = append(result, err)
		}
	}

	return result
}

// Clear removes all errors from the handler.
func (eh *ErrorHandler) Clear() {
	eh.errors = eh.errors[:0]
}

// HasErrors returns true if any errors have been collected.
func (eh *ErrorHandler) HasErrors() bool {
	return len(eh.errors) > 0
}

// ErrorCount returns the total number of errors.
func (eh *ErrorHandler) ErrorCount() int {
	return len(eh.errors)
}

// GetSummary returns a summary of errors by category.
func (eh *ErrorHandler) GetSummary() map[ErrorCategory]int {
	summary := make(map[ErrorCategory]int)
	for _, err := range eh.errors {
		summary[err.Category]++
	}

	return summary
}

// ClassifySyncError automatically classifies an error into the appropriate category.
func ClassifySyncError(operation, path string, err error) *SyncError {
	if err == nil {
		return nil
	}

	retryableErr := ClassifyError(err)

	if isNetworkError(err) {
		return NewNetworkError(operation, path, err)
	}

	if isPermissionError(err) {
		return NewPermissionError(operation, path, err)
	}

	if isDiskFullError(err) {
		return NewDiskError(operation, path, err)
	}

	if isContextError(err) {
		return NewCancellationError(operation, path, err)
	}

	// Default to unknown error with retryable status
	return &SyncError{
		Category:    ErrorCategoryUnknown,
		Operation:   operation,
		Path:        path,
		Message:     err.Error(),
		Underlying:  err,
		Timestamp:   time.Now(),
		Recoverable: retryableErr != nil && retryableErr.Retryable,
		Suggestion:  "Check logs for more details and try again",
	}
}

// GetRecoverySuggestion returns a suggestion for how to recover from an error.
func GetRecoverySuggestion(err *SyncError) string {
	if err.Suggestion != "" {
		return err.Suggestion
	}

	switch err.Category {
	case ErrorCategoryNetwork:
		return "Check network connectivity, firewall settings, and DNS resolution"
	case ErrorCategoryPermission:
		return "Verify file permissions, user privileges, and access rights"
	case ErrorCategoryDisk:
		return "Free up disk space, check disk health, and verify mount points"
	case ErrorCategoryCorruption:
		return "Verify file integrity, check for hardware issues, and restore from backup"
	case ErrorCategoryConfiguration:
		return "Validate configuration syntax, check file paths, and verify settings"
	case ErrorCategoryCancellation:
		return "Increase timeout values or avoid interrupting operations"
	default:
		return "Check system logs, verify prerequisites, and contact support if needed"
	}
}
