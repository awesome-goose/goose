package output

import (
	"net/http"

	"github.com/awesome-goose/goose/types"
)

// =============================================================================
// JSON Response Helpers
// =============================================================================

// Success creates a successful JSON response with data
func Success(message string, data any) types.Output {
	return NewJSONOutput(data, http.StatusOK, true, message, nil)
}

// SuccessWithCode creates a successful JSON response with a custom status code
func SuccessWithCode(message string, data any, code int) types.Output {
	return NewJSONOutput(data, code, true, message, nil)
}

// SuccessWithMeta creates a successful JSON response with metadata
func SuccessWithMeta(message string, data any, meta any) types.Output {
	return NewJSONOutput(data, http.StatusOK, true, message, meta)
}

// SuccessWithCodeAndMeta creates a successful JSON response with custom code and metadata
func SuccessWithCodeAndMeta(message string, data any, code int, meta any) types.Output {
	return NewJSONOutput(data, code, true, message, meta)
}

// Error creates an error JSON response
func Error(message string) types.Output {
	return NewJSONOutput(nil, http.StatusBadRequest, false, message, nil)
}

// ErrorWithCode creates an error JSON response with a custom status code
func ErrorWithCode(message string, code int) types.Output {
	return NewJSONOutput(nil, code, false, message, nil)
}

// ErrorWithData creates an error JSON response with additional data
func ErrorWithData(message string, data any) types.Output {
	return NewJSONOutput(data, http.StatusBadRequest, false, message, nil)
}

// ErrorWithCodeAndData creates an error JSON response with custom code and data
func ErrorWithCodeAndData(message string, data any, code int) types.Output {
	return NewJSONOutput(data, code, false, message, nil)
}

// =============================================================================
// Common HTTP Status Helpers
// =============================================================================

// OK returns a 200 OK response with data
func OK(data any) types.Output {
	return JSON(data)
}

// Created returns a 201 Created response
func Created(data any, message ...string) types.Output {
	msg := "Resource created successfully"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(data, http.StatusCreated, true, msg, nil)
}

// Accepted returns a 202 Accepted response
func Accepted(data any, message ...string) types.Output {
	msg := "Request accepted for processing"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(data, http.StatusAccepted, true, msg, nil)
}

// NoContent returns a 204 No Content response
func NoContent() types.Output {
	return NewJSONOutput(nil, http.StatusNoContent, true, "", nil)
}

// BadRequest returns a 400 Bad Request response
func BadRequest(message string, data ...any) types.Output {
	var d any
	if len(data) > 0 {
		d = data[0]
	}
	return NewJSONOutput(d, http.StatusBadRequest, false, message, nil)
}

// Unauthorized returns a 401 Unauthorized response
func Unauthorized(message ...string) types.Output {
	msg := "Unauthorized access"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(nil, http.StatusUnauthorized, false, msg, nil)
}

// Forbidden returns a 403 Forbidden response
func Forbidden(message ...string) types.Output {
	msg := "Access forbidden"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(nil, http.StatusForbidden, false, msg, nil)
}

// NotFound returns a 404 Not Found response
func NotFound(message ...string) types.Output {
	msg := "Resource not found"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(nil, http.StatusNotFound, false, msg, nil)
}

// MethodNotAllowed returns a 405 Method Not Allowed response
func MethodNotAllowed(message ...string) types.Output {
	msg := "Method not allowed"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(nil, http.StatusMethodNotAllowed, false, msg, nil)
}

// Conflict returns a 409 Conflict response
func Conflict(message string, data ...any) types.Output {
	var d any
	if len(data) > 0 {
		d = data[0]
	}
	return NewJSONOutput(d, http.StatusConflict, false, message, nil)
}

// Gone returns a 410 Gone response
func Gone(message ...string) types.Output {
	msg := "Resource is no longer available"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(nil, http.StatusGone, false, msg, nil)
}

// UnprocessableEntity returns a 422 response (commonly used for validation errors)
func UnprocessableEntity(message string, errors any) types.Output {
	return NewJSONOutput(errors, http.StatusUnprocessableEntity, false, message, nil)
}

// ValidationErrors returns a 422 response with validation errors
func ValidationErrors(errors map[string][]string) types.Output {
	return NewJSONOutput(errors, http.StatusUnprocessableEntity, false, "Validation failed", nil)
}

// TooManyRequests returns a 429 Too Many Requests response
func TooManyRequests(message ...string) types.Output {
	msg := "Too many requests"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(nil, http.StatusTooManyRequests, false, msg, nil)
}

// InternalServerError returns a 500 Internal Server Error response
func InternalServerError(message ...string) types.Output {
	msg := "Internal server error"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(nil, http.StatusInternalServerError, false, msg, nil)
}

// ServiceUnavailable returns a 503 Service Unavailable response
func ServiceUnavailable(message ...string) types.Output {
	msg := "Service temporarily unavailable"
	if len(message) > 0 {
		msg = message[0]
	}
	return NewJSONOutput(nil, http.StatusServiceUnavailable, false, msg, nil)
}

// =============================================================================
// Pagination Helper
// =============================================================================

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	CurrentPage  int   `json:"current_page"`
	PerPage      int   `json:"per_page"`
	Total        int64 `json:"total"`
	TotalPages   int   `json:"total_pages"`
	HasMore      bool  `json:"has_more"`
	HasPrevious  bool  `json:"has_previous"`
	NextPage     *int  `json:"next_page,omitempty"`
	PreviousPage *int  `json:"previous_page,omitempty"`
}

// NewPaginationMeta creates pagination metadata
func NewPaginationMeta(page, perPage int, total int64) *PaginationMeta {
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	hasMore := page < totalPages
	hasPrevious := page > 1

	meta := &PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
		HasMore:     hasMore,
		HasPrevious: hasPrevious,
	}

	if hasMore {
		next := page + 1
		meta.NextPage = &next
	}
	if hasPrevious {
		prev := page - 1
		meta.PreviousPage = &prev
	}

	return meta
}

// Paginated creates a paginated response
func Paginated(data any, page, perPage int, total int64) types.Output {
	meta := NewPaginationMeta(page, perPage, total)
	return NewJSONOutput(data, http.StatusOK, true, "", meta)
}
