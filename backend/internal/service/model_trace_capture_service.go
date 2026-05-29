package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type ModelTraceCaptureService struct {
	repo ModelTraceCaptureRepository
}

func NewModelTraceCaptureService(repo ModelTraceCaptureRepository) *ModelTraceCaptureService {
	return &ModelTraceCaptureService{repo: repo}
}

func (s *ModelTraceCaptureService) Create(ctx context.Context, capture *ModelTraceCapture) (bool, error) {
	if s == nil || s.repo == nil || capture == nil {
		return false, nil
	}
	return s.repo.Create(ctx, capture)
}

func (s *ModelTraceCaptureService) GetByID(ctx context.Context, id int64) (*ModelTraceCapture, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model trace capture repository is not configured")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ModelTraceCaptureService) GetByTaskID(ctx context.Context, taskID string) (*ModelTraceCapture, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model trace capture repository is not configured")
	}
	return s.repo.GetByTaskID(ctx, taskID)
}

func (s *ModelTraceCaptureService) GetByMainSessionKey(ctx context.Context, mainSessionKey string) (*ModelTraceCapture, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model trace capture repository is not configured")
	}
	return s.repo.GetByMainSessionKey(ctx, mainSessionKey)
}

func (s *ModelTraceCaptureService) GetByDedupeHash(ctx context.Context, dedupeHash string) (*ModelTraceCapture, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model trace capture repository is not configured")
	}
	return s.repo.GetByDedupeHash(ctx, dedupeHash)
}

func (s *ModelTraceCaptureService) List(ctx context.Context, filter ModelTraceCaptureListFilter, params pagination.PaginationParams) ([]*ModelTraceCapture, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("model trace capture repository is not configured")
	}
	return s.repo.List(ctx, filter, params)
}

func (s *ModelTraceCaptureService) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, params pagination.PaginationParams) ([]*ModelTraceCapture, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("model trace capture repository is not configured")
	}
	return s.repo.ListByTimeRange(ctx, startTime, endTime, params)
}

func (s *ModelTraceCaptureService) DeleteByID(ctx context.Context, id int64) (bool, error) {
	if s == nil || s.repo == nil {
		return false, fmt.Errorf("model trace capture repository is not configured")
	}
	return s.repo.DeleteByID(ctx, id)
}

func (s *ModelTraceCaptureService) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, fmt.Errorf("model trace capture repository is not configured")
	}
	return s.repo.DeleteByIDs(ctx, ids)
}
