package domain

import (
	"context"
	"time"
)

// AuditRecord describes an append-only audit event.
type AuditRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Remote    string    `json:"remote"`
	Status    int       `json:"status"`
}

// AuditLogger persists audit records.
type AuditLogger interface {
	Append(ctx context.Context, record AuditRecord) error
}
