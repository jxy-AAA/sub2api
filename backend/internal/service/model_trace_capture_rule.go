package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ModelTraceCaptureRule struct {
	ID int64 `json:"id"`

	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`

	ModelPatterns []string `json:"model_patterns"`
	UserIDs       []int64  `json:"user_ids"`
	APIKeyIDs     []int64  `json:"api_key_ids"`
	Keywords      []string `json:"keywords"`

	MinTokens *int64 `json:"min_tokens,omitempty"`
	MaxTokens *int64 `json:"max_tokens,omitempty"`

	SamplingRatio float64 `json:"sampling_ratio"`

	ActiveFrom *time.Time `json:"active_from,omitempty"`
	ActiveTo   *time.Time `json:"active_to,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ModelTraceCaptureRuleRepository interface {
	Create(ctx context.Context, rule *ModelTraceCaptureRule) (*ModelTraceCaptureRule, error)
	Update(ctx context.Context, rule *ModelTraceCaptureRule) (*ModelTraceCaptureRule, error)
	GetByID(ctx context.Context, id int64) (*ModelTraceCaptureRule, error)
	List(ctx context.Context) ([]*ModelTraceCaptureRule, error)
	DeleteByID(ctx context.Context, id int64) (bool, error)
}

func (r *ModelTraceCaptureRule) Validate() error {
	if r == nil {
		return fmt.Errorf("model trace capture rule is nil")
	}

	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	r.ModelPatterns = normalizeTraceCaptureRuleStrings(r.ModelPatterns)
	r.Keywords = normalizeTraceCaptureRuleStrings(r.Keywords)

	userIDs, err := normalizeTraceCaptureRuleInt64s("user_ids", r.UserIDs)
	if err != nil {
		return err
	}
	r.UserIDs = userIDs

	apiKeyIDs, err := normalizeTraceCaptureRuleInt64s("api_key_ids", r.APIKeyIDs)
	if err != nil {
		return err
	}
	r.APIKeyIDs = apiKeyIDs

	if err := validateTraceCaptureOptionalInt64("min_tokens", r.MinTokens); err != nil {
		return err
	}
	if err := validateTraceCaptureOptionalInt64("max_tokens", r.MaxTokens); err != nil {
		return err
	}
	if r.MinTokens != nil && r.MaxTokens != nil && *r.MinTokens > *r.MaxTokens {
		return fmt.Errorf("min_tokens must be <= max_tokens")
	}

	if r.SamplingRatio == 0 {
		r.SamplingRatio = 1
	}
	if r.SamplingRatio <= 0 || r.SamplingRatio > 1 {
		return fmt.Errorf("sampling_ratio must be > 0 and <= 1")
	}

	if r.ActiveFrom != nil && r.ActiveTo != nil && r.ActiveFrom.After(*r.ActiveTo) {
		return fmt.Errorf("active_from must be <= active_to")
	}

	return nil
}

func normalizeTraceCaptureRuleStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func normalizeTraceCaptureRuleInt64s(name string, values []int64) ([]int64, error) {
	if len(values) == 0 {
		return []int64{}, nil
	}
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			return nil, fmt.Errorf("%s must contain positive ids", name)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}
