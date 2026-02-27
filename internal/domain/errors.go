package domain

import "fmt"

// ---------- Not Found ----------

// NotFoundError indicates that the requested entity does not exist.
type NotFoundError struct {
	Entity string
	Key    string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Entity, e.Key)
}

// NewNotFoundError constructs a NotFoundError for the given entity and lookup key.
func NewNotFoundError(entity, key string) *NotFoundError {
	return &NotFoundError{Entity: entity, Key: key}
}

// ---------- Validation ----------

// ValidationError signals that input data failed business-rule checks.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s — %s", e.Field, e.Message)
}

// NewValidationError constructs a ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// ---------- Conflict ----------

// ConflictError indicates a uniqueness or invariant violation
// (e.g. duplicate email).
type ConflictError struct {
	Entity  string
	Message string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict: %s — %s", e.Entity, e.Message)
}

// NewConflictError constructs a ConflictError.
func NewConflictError(entity, message string) *ConflictError {
	return &ConflictError{Entity: entity, Message: message}
}
