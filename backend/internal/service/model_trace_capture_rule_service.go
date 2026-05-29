package service

import (
	"context"
	"fmt"
)

type ModelTraceCaptureRuleService struct {
	repo ModelTraceCaptureRuleRepository
}

func NewModelTraceCaptureRuleService(repo ModelTraceCaptureRuleRepository) *ModelTraceCaptureRuleService {
	return &ModelTraceCaptureRuleService{repo: repo}
}

func (s *ModelTraceCaptureRuleService) Create(ctx context.Context, rule *ModelTraceCaptureRule) (*ModelTraceCaptureRule, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model trace capture rule repository is not configured")
	}
	if rule == nil {
		return nil, fmt.Errorf("model trace capture rule is nil")
	}
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, rule)
}

func (s *ModelTraceCaptureRuleService) Update(ctx context.Context, rule *ModelTraceCaptureRule) (*ModelTraceCaptureRule, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model trace capture rule repository is not configured")
	}
	if rule == nil {
		return nil, fmt.Errorf("model trace capture rule is nil")
	}
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, rule)
}

func (s *ModelTraceCaptureRuleService) GetByID(ctx context.Context, id int64) (*ModelTraceCaptureRule, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model trace capture rule repository is not configured")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ModelTraceCaptureRuleService) List(ctx context.Context) ([]*ModelTraceCaptureRule, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model trace capture rule repository is not configured")
	}
	return s.repo.List(ctx)
}

func (s *ModelTraceCaptureRuleService) DeleteByID(ctx context.Context, id int64) (bool, error) {
	if s == nil || s.repo == nil {
		return false, fmt.Errorf("model trace capture rule repository is not configured")
	}
	return s.repo.DeleteByID(ctx, id)
}
