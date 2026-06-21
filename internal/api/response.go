package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/ecommerce/feature-management/internal/domain"
)

// Response is the standard success envelope.
type Response struct {
	Data interface{} `json:"data"`
	Meta *Meta       `json:"meta,omitempty"`
}

// Meta carries pagination and request metadata.
type Meta struct {
	Total     int    `json:"total,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the machine-readable error code and human message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// JSONError writes a JSON error response.
func JSONError(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// HandleError maps domain errors to HTTP responses.
func HandleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound),
		errors.Is(err, domain.ErrFlagNotFound),
		errors.Is(err, domain.ErrRuleNotFound),
		errors.Is(err, domain.ErrSegmentNotFound),
		errors.Is(err, domain.ErrApplicationNotFound),
		errors.Is(err, domain.ErrEnvironmentNotFound):
		JSONError(w, http.StatusNotFound, "not_found", err.Error())

	case errors.Is(err, domain.ErrAlreadyExists):
		JSONError(w, http.StatusConflict, "already_exists", err.Error())

	case errors.Is(err, domain.ErrVersionConflict):
		JSONError(w, http.StatusConflict, "version_conflict", err.Error())

	case errors.Is(err, domain.ErrInvalidInput):
		JSONError(w, http.StatusBadRequest, "invalid_input", err.Error())

	case errors.Is(err, domain.ErrFlagArchived):
		JSONError(w, http.StatusUnprocessableEntity, "flag_archived", err.Error())

	default:
		JSONError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	}
}

// ParseUUID parses a UUID string and returns an error if invalid.
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// ParseIntQuery reads an integer query parameter, returning def if missing or unparseable.
func ParseIntQuery(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
