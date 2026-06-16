package domain

import "errors"

var (
	ErrNotFound            = errors.New("resource not found")
	ErrAlreadyExists       = errors.New("resource already exists")
	ErrInvalidInput        = errors.New("invalid input")
	ErrVersionConflict     = errors.New("version conflict")
	ErrFlagArchived        = errors.New("flag is archived")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrSegmentNotFound     = errors.New("segment not found")
	ErrFlagNotFound        = errors.New("flag not found")
	ErrRuleNotFound        = errors.New("rule not found")
	ErrApplicationNotFound = errors.New("application not found")
	ErrEnvironmentNotFound = errors.New("environment not found")
)
