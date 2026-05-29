package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// ModelInteractionTrace keeps the raw, lossless payload for one model interaction.
// JSON fields are stored as-is so downstream exporters can emit the original structure.
type ModelInteractionTrace struct {
	ID int64

	TaskID string

	Prompt     json.RawMessage
	Candidates json.RawMessage
	Tools      json.RawMessage
	Signature  json.RawMessage
	Meta       json.RawMessage
	Scaffold   json.RawMessage

	ScaffoldVersion string
	Model           *string
	UserID          *int64
	APIKeyID        *int64
	RequestID       *string

	DedupeHash string
	CreatedAt  time.Time
}

type ModelInteractionTraceRepository interface {
	Create(ctx context.Context, trace *ModelInteractionTrace) (bool, error)
	ListAll(ctx context.Context) ([]*ModelInteractionTrace, error)
	ListByTimeRange(ctx context.Context, startTime, endTime time.Time, params pagination.PaginationParams) ([]*ModelInteractionTrace, *pagination.PaginationResult, error)
}
