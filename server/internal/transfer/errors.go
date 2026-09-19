package transfer

import (
	"context"
	"errors"
	"fmt"
)

// TransferError is a structured, classified transport error. It is the primary
// error type returned by backends and the engine so upper layers (UI) can
// decide how to react (retry, wait for device, show a message...).
type TransferError struct {
	Code       string
	Message    string
	Retryable  bool
	DeviceLost bool

	// Cause is the underlying error (may be nil).
	Cause error
}

func (e *TransferError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("transfer error: %s", e.Code)
}

func (e *TransferError) Unwrap() error {
	return e.Cause
}

// Transfer error codes.
const (
	CodeAFCIOError     = "AFC_IO_ERROR"
	CodeSourceNotFound = "SOURCE_NOT_FOUND"
	CodeVerifyFailed   = "VERIFY_FAILED"
	CodeUserCancelled  = "USER_CANCELLED"
)

// NewTransferError builds a structured TransferError.
func NewTransferError(code, message string, retryable, deviceLost bool) *TransferError {
	return &TransferError{
		Code:       code,
		Message:    message,
		Retryable:  retryable,
		DeviceLost: deviceLost,
	}
}

// WrapTransferError attaches a cause to a TransferError.
func WrapTransferError(err *TransferError, cause error) error {
	if err == nil {
		return cause
	}
	err.Cause = cause
	return err
}

// Error helpers for the most common cases.
func ErrSourceNotFound(msg string) *TransferError {
	return NewTransferError(CodeSourceNotFound, msg, false, false)
}

func ErrVerifyFailed(msg string) *TransferError {
	return NewTransferError(CodeVerifyFailed, msg, true, false)
}

// AsTransferError extracts a *TransferError from err, returning nil if err is
// not (and does not wrap) a TransferError.
func AsTransferError(err error) *TransferError {
	var te *TransferError
	if errors.As(err, &te) {
		return te
	}
	return nil
}

// IsCancellation reports whether err represents a user/context cancellation
// (and not a genuine failure).
func IsCancellation(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	if te := AsTransferError(err); te != nil && te.Code == CodeUserCancelled {
		return true
	}
	return false
}

// IsRetryable reports whether err is worth retrying (transient conditions).
func IsRetryable(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if te := AsTransferError(err); te != nil {
		return te.Retryable
	}
	return false
}
