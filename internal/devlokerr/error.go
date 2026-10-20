package devlokerr

import (
	"errors"
	"fmt"
)

type Code string

const (
	CodeInternal          Code = "INTERNAL"
	CodeInvalidArgument   Code = "INVALID_ARGUMENT"
	CodeInvalidManifest   Code = "INVALID_MANIFEST"
	CodeManifestNotFound  Code = "MANIFEST_NOT_FOUND"
	CodePortInUse         Code = "PORT_IN_USE"
	CodeElevationRequired Code = "ELEVATION_REQUIRED"
	CodeDependencyMissing Code = "DEPENDENCY_MISSING"
	CodeTimeout           Code = "TIMEOUT"
	CodeCanceled          Code = "CANCELED"
	CodeStateInconsistent Code = "STATE_INCONSISTENT"
	CodeUnsupported       Code = "UNSUPPORTED"
	CodePermissionDenied  Code = "PERMISSION_DENIED"
	CodeRuntimeNotFound   Code = "RUNTIME_NOT_FOUND"
	CodeChecksumMismatch  Code = "CHECKSUM_MISMATCH"
)

const (
	ExitOK                = 0
	ExitGeneral           = 1
	ExitInvalid           = 2
	ExitPortInUse         = 3
	ExitElevation         = 4
	ExitDependency        = 5
	ExitTimeout           = 6
	ExitCanceled          = 7
	ExitStateInconsistent = 8
)

type Error struct {
	Code    Code
	Message string
	Hint    string
	Details map[string]any
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(code Code, message, hint string) *Error {
	return &Error{Code: code, Message: message, Hint: hint}
}

func Wrap(code Code, err error, message, hint string) *Error {
	return &Error{Code: code, Message: message, Hint: hint, Err: err}
}

func WithDetails(err *Error, details map[string]any) *Error {
	if err == nil {
		return nil
	}
	err.Details = details
	return err
}

func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	e, ok := As(err)
	if !ok {
		return ExitGeneral
	}
	switch e.Code {
	case CodeInvalidManifest, CodeInvalidArgument, CodeManifestNotFound:
		return ExitInvalid
	case CodePortInUse:
		return ExitPortInUse
	case CodeElevationRequired, CodePermissionDenied:
		return ExitElevation
	case CodeDependencyMissing, CodeRuntimeNotFound, CodeChecksumMismatch, CodeUnsupported:
		return ExitDependency
	case CodeTimeout:
		return ExitTimeout
	case CodeCanceled:
		return ExitCanceled
	case CodeStateInconsistent:
		return ExitStateInconsistent
	default:
		return ExitGeneral
	}
}
