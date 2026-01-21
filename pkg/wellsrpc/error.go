package wellsrpc

import (
	"errors"
)

// =======================
// RPC Error Code
// =======================
// Error code bersifat STABLE.
// Jangan diubah sembarangan (audit & backward compatibility).
const (
	// Generic
	ErrCodeInternal         = "INTERNAL_ERROR"
	ErrCodeUnavailable      = "SERVICE_UNAVAILABLE"
	ErrCodeDeadlineExceeded = "DEADLINE_EXCEEDED"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeBadRequest       = "BAD_REQUEST"

	// Retry related
	ErrCodeRetryLater = "RETRY_LATER"

	// Idempotency
	ErrCodeDuplicateRequest = "DUPLICATE_REQUEST"
	ErrCodeInProgress       = "REQUEST_IN_PROGRESS"

	// Business / banking example
	ErrCodeInsufficientFund = "INSUFFICIENT_FUND"
	ErrCodeAccountBlocked   = "ACCOUNT_BLOCKED"
)

// =======================
// RPCError (CORE TYPE)
// =======================
// Ini SATU-SATUNYA error resmi yang boleh
// keluar dari layer RPC.
type RPCError struct {
	Code      string // error code (stable)
	Message   string // human readable
	Retryable bool   // boleh di-retry atau tidak
}

// Pastikan RPCError implement error interface
func (e *RPCError) Error() string {
	return e.Code + ": " + e.Message
}

//
// =======================
// Helper Constructors
// =======================
// Supaya konsisten & tidak salah pakai
//

func NewInternalError(msg string) *RPCError {
	return &RPCError{
		Code:      ErrCodeInternal,
		Message:   msg,
		Retryable: false,
	}
}

func NewUnavailableError(msg string) *RPCError {
	return &RPCError{
		Code:      ErrCodeUnavailable,
		Message:   msg,
		Retryable: true,
	}
}

func NewDeadlineExceededError() *RPCError {
	return &RPCError{
		Code:      ErrCodeDeadlineExceeded,
		Message:   "request deadline exceeded",
		Retryable: true,
	}
}

func NewUnauthorizedError() *RPCError {
	return &RPCError{
		Code:      ErrCodeUnauthorized,
		Message:   "unauthorized",
		Retryable: false,
	}
}

func NewForbiddenError() *RPCError {
	return &RPCError{
		Code:      ErrCodeForbidden,
		Message:   "forbidden",
		Retryable: false,
	}
}

func NewBadRequestError(msg string) *RPCError {
	return &RPCError{
		Code:      ErrCodeBadRequest,
		Message:   msg,
		Retryable: false,
	}
}

func NewRetryLaterError(msg string) *RPCError {
	return &RPCError{
		Code:      ErrCodeRetryLater,
		Message:   msg,
		Retryable: true,
	}
}

func NewDuplicateRequestError() *RPCError {
	return &RPCError{
		Code:      ErrCodeDuplicateRequest,
		Message:   "duplicate request",
		Retryable: false,
	}
}

func NewRequestInProgressError() *RPCError {
	return &RPCError{
		Code:      ErrCodeInProgress,
		Message:   "request is still in progress",
		Retryable: true,
	}
}

//
// =======================
// Business / Banking Errors
// =======================
// Contoh error bisnis (TIDAK RETRYABLE)
//

func NewInsufficientFundError() *RPCError {
	return &RPCError{
		Code:      ErrCodeInsufficientFund,
		Message:   "insufficient balance",
		Retryable: false,
	}
}

func NewAccountBlockedError() *RPCError {
	return &RPCError{
		Code:      ErrCodeAccountBlocked,
		Message:   "account is blocked",
		Retryable: false,
	}
}

//
// =======================
// Utilities
// =======================

// IsRPCError mengecek apakah error adalah RPCError
func IsRPCError(err error) (*RPCError, bool) {
	var re *RPCError
	if errors.As(err, &re) {
		return re, true
	}
	return nil, false
}

// IsRetryable menentukan apakah error boleh di-retry
func IsRetryable(err error) bool {
	if re, ok := IsRPCError(err); ok {
		return re.Retryable
	}
	return false
}

// =======================
// Frame Conversion
// =======================
// Serialize error ke Frame (SERVER SIDE)
func ErrorToFrame(streamID uint32, method string, meta map[string]string, err error) *Frame {
	re, ok := IsRPCError(err)
	if !ok {
		re = NewInternalError(err.Error())
	}

	if meta == nil {
		meta = make(map[string]string)
	}

	meta["error-code"] = re.Code
	meta["retryable"] = boolToString(re.Retryable)

	return &Frame{
		Type:     FrameTypeError,
		StreamID: streamID,
		Method:   method,
		Metadata: meta,
		Payload:  []byte(re.Message),
	}
}

// Parse error dari Frame (CLIENT SIDE)
func FrameToError(f *Frame) error {
	if f == nil || f.Type != FrameTypeError {
		return nil
	}

	code := f.Metadata["error-code"]
	retryable := f.Metadata["retryable"] == "true"
	msg := string(f.Payload)

	if code == "" {
		code = ErrCodeInternal
	}

	return &RPCError{
		Code:      code,
		Message:   msg,
		Retryable: retryable,
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
