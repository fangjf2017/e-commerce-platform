// Package api wires together handlers, middleware and the chi router.
// The shared HTTP helper types and functions live in the apiutil sub-package
// so that the handlers sub-package can import them without creating a cycle.
package api

import (
	"github.com/ecommerce/feature-management/internal/api/apiutil"
)

// Re-export the shared types so callers that only import "api" still work.

// Response is the standard success envelope.
type Response = apiutil.Response

// Meta carries pagination and request metadata.
type Meta = apiutil.Meta

// ErrorResponse is the standard error envelope.
type ErrorResponse = apiutil.ErrorResponse

// ErrorDetail contains the machine-readable error code and human message.
type ErrorDetail = apiutil.ErrorDetail

// Re-export functions from apiutil so handlers importing "api" can use them directly.

// JSON writes a JSON response.
var JSON = apiutil.JSON

// JSONError writes a structured JSON error response.
var JSONError = apiutil.JSONError

// HandleError maps domain errors to HTTP status codes.
var HandleError = apiutil.HandleError

// ParseUUID parses a UUID string.
var ParseUUID = apiutil.ParseUUID

// ParseIntQuery reads an integer query parameter.
var ParseIntQuery = apiutil.ParseIntQuery
