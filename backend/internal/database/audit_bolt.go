package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"go.etcd.io/bbolt"

	"github.com/rxritet/Specto/internal/domain"
)

var bucketAuditLog = []byte("audit_log")

// BoltAuditLogger stores append-only audit records in BoltDB.
type BoltAuditLogger struct {
	db     *bbolt.DB
	logger *slog.Logger
}

func NewBoltAuditLogger(db *bbolt.DB, logger *slog.Logger) *BoltAuditLogger {
	return &BoltAuditLogger{db: db, logger: logger}
}

func (l *BoltAuditLogger) Append(_ context.Context, record domain.AuditRecord) error {
	return l.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketAuditLog)
		if b == nil {
			return fmt.Errorf("audit bucket not found")
		}

		id, err := b.NextSequence()
		if err != nil {
			return fmt.Errorf("next audit sequence: %w", err)
		}

		payload, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("marshal audit record: %w", err)
		}

		if err := b.Put(itob(int64(id)), payload); err != nil {
			return fmt.Errorf("store audit record: %w", err)
		}
		return nil
	})
}
