package errors

import (
	"errors"
	"fmt"
)

type Error struct {
	Code            string
	Message         string
	Detail          string
	SuggestedAction string
	Meta            any
	Err             error
}

func New(code, message, detail, suggestedAction string) *Error {
	return &Error{Code: code, Message: message, Detail: detail, SuggestedAction: suggestedAction}
}

func NewWithMeta(code, message, detail, suggestedAction string, meta any) *Error {
	return &Error{Code: code, Message: message, Detail: detail, SuggestedAction: suggestedAction, Meta: meta}
}

func Wrap(err error, code, message, detail, suggestedAction string) *Error {
	return &Error{Code: code, Message: message, Detail: detail, SuggestedAction: suggestedAction, Err: err}
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err != nil {
		if e.Detail != "" {
			return fmt.Sprintf("%s: %s (%s): %v", e.Code, e.Message, e.Detail, e.Err)
		}
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Detail)
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

func (e *Error) GetDetail() string {
	if e == nil {
		return ""
	}
	return e.Detail
}

func (e *Error) GetSuggestedAction() string {
	if e == nil {
		return ""
	}
	return e.SuggestedAction
}

func (e *Error) WithDetail(detail string) *Error {
	e.Detail = detail
	return e
}

func (e *Error) WithSuggestedAction(action string) *Error {
	e.SuggestedAction = action
	return e
}
