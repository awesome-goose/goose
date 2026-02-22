package errors

import (
	"errors"
	"fmt"
)

type Error struct {
	Code    string
	Message string
	Meta    any
	Err     error
}

func New(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func NewWithMeta(code, message string, meta any) *Error {
	return &Error{Code: code, Message: message, Meta: meta}
}

func Wrap(err error, code, message string) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
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

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

func (e *Error) As(target any) bool {
	return errors.As(e, target)
}

func (e *Error) WithMeta(meta any) *Error {
	e.Meta = meta
	return e
}

func (e *Error) WithError(err error) *Error {
	e.Err = err
	return e
}

func (e *Error) String() string {
	return e.Error()
}

func (e *Error) GetCode() string {
	if e == nil {
		return ""
	}
	return e.Code
}

func (e *Error) GetMessage() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) GetMeta() any {
	if e == nil {
		return nil
	}
	return e.Meta
}
