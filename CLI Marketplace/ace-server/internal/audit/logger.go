package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/nicroldan/ans/shared/ace"
	"github.com/nicroldan/ans/ace-server/internal/store"
)

type contextKey string

const correlationIDKey contextKey = "correlation_id"

// Logger wraps the audit store with convenience methods.
type Logger struct {
	store *store.MemoryStore
}

// NewLogger creates a new audit logger.
func NewLogger(s *store.MemoryStore) *Logger {
	return &Logger{store: s}
}

// Log records an audit entry.
func (l *Logger) Log(ctx context.Context, storeID, action, actor, actorType, resource, resourceID string, details map[string]any) {
	corrID := CorrelationIDFromContext(ctx)
	if corrID == "" {
		corrID = generateCorrelationID()
	}
	entry := &ace.AuditEntry{
		StoreID:       storeID,
		Action:        action,
		Actor:         actor,
		ActorType:     actorType,
		Resource:      resource,
		ResourceID:    resourceID,
		Details:       details,
		CorrelationID: corrID,
	}
	l.store.AppendAuditEntry(entry)
}

// WithCorrelationID returns a context with the given correlation ID.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationIDFromContext extracts the correlation ID from context.
func CorrelationIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(correlationIDKey).(string); ok {
		return v
	}
	return ""
}

func generateCorrelationID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
