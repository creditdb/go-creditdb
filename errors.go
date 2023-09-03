package creditdb

import "net/http"

type Error struct {
	Message  string
	Category string
}

const (
	CategoryNotFound   = "NotFound"
	CategoryBadRequest = "BadRequest"
	CategoryTimeout = "Timeout"
	CategoryServiceUnavailable = "ServiceUnavailable"
	CategoryInternalError = "InternalError"
)

func (e *Error) Error() string {
	return e.Message
}

func NewError(message string, category string) *Error {
	return &Error{Message: message, Category: category}
}

func (e *Error) StatusCode() int {
	switch e.Category {
	case CategoryNotFound:
		return http.StatusNotFound
	case CategoryBadRequest:
		return http.StatusBadRequest
	case CategoryTimeout:
		return http.StatusRequestTimeout
	case CategoryServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}

}
