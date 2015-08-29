package web

import (
	"encoding/json"
	"net/http"
)

// Errors

// Errors is a list of errors
type Errors struct {
	Errors []*Error `json:"errors"`
}

// Error holds the info about a web error
type Error struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// WriteError writes an error to the reply
func WriteError(w http.ResponseWriter, err *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(Errors{[]*Error{err}})
}

var (
	// ErrBadRequest is a generic bad request
	ErrBadRequest = &Error{"bad_request", 400, "Bad request", "Request body is not well-formed. It must be JSON."}
	// ErrBadCaptcha is a captcha error
	ErrBadCaptcha = &Error{"bad_captcha", 400, "Bad CAPTCHA", "The provided CAPTCHA response does not match."}
	// ErrMissingPartRequest returns 400 if the request is missing some parts
	ErrMissingPartRequest = &Error{"missing_request", 400, "Bad request", "Request body is missing mandatory parts."}
	// ErrBadContentRequest if the request content is wrong
	ErrBadContentRequest = &Error{"bad_content", 400, "Bad content", "Request contains bad content"}
	// ErrAuth if not authenticated
	ErrAuth = &Error{"unauthorized", 401, "Unauthorized", "The request requires authorization"}
	// ErrCredentials if there are missing / wrong credentials
	ErrCredentials = &Error{"invalid_credentials", 401, "Invalid credentials", "Invalid username or password"}
	// ErrNotFound if file is not found
	ErrNotFound = &Error{"not_found", 404, "Not found", "The page you requested is not found"}
	// ErrNotAcceptable wrong accept header
	ErrNotAcceptable = &Error{"not_acceptable", 406, "Not Acceptable", "Accept header must be set to 'application/json'."}
	// ErrUnsupportedMediaType wrong media type
	ErrUnsupportedMediaType = &Error{"unsupported_media_type", 415, "Unsupported Media Type", "Content-Type header must be set to: 'application/json'."}
	// ErrCSRF missing CSRF cookie or parameter
	ErrCSRF = &Error{"forbidden", 403, "Forbidden", "Issue with CSRF code"}
	// ErrForbidden if request is forbidden to the user
	ErrForbidden = &Error{"forbidden", 403, "Forbidden", "Forbidden"}
	// ErrInternalServer if things go wrong on our side
	ErrInternalServer = &Error{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
)
