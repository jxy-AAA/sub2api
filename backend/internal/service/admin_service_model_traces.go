package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const adminTraceExportPageSize = 1000

func (s *adminServiceImpl) ListModelInteractionTraces(ctx context.Context, startTime, endTime *time.Time) ([]*ModelInteractionTrace, error) {
	if s == nil || s.traceRepo == nil {
		return nil, fmt.Errorf("model interaction trace repository is not configured")
	}

	var start time.Time
	if startTime != nil {
		start = *startTime
	}

	var end time.Time
	if endTime != nil {
		end = *endTime
	}

	page := 1
	out := make([]*ModelInteractionTrace, 0, adminTraceExportPageSize)
	for {
		items, result, err := s.traceRepo.ListByTimeRange(ctx, start, end, pagination.PaginationParams{
			Page:     page,
			PageSize: adminTraceExportPageSize,
		})
		if err != nil {
			return nil, err
		}

		out = append(out, items...)
		if result == nil || len(items) == 0 || len(out) >= int(result.Total) {
			break
		}
		page++
	}

	return out, nil
}
