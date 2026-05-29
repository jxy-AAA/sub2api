package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestModelTraceCaptureRuleValidateNormalizesAndDefaults(t *testing.T) {
	t.Parallel()

	minTokens := int64(10)
	maxTokens := int64(20)
	activeFrom := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	activeTo := activeFrom.Add(time.Hour)
	rule := &ModelTraceCaptureRule{
		Name:          "  Incident rule  ",
		Enabled:       true,
		Priority:      5,
		ModelPatterns: []string{" gpt-4.1 ", "", "gpt-4.1", " claude-* "},
		UserIDs:       []int64{42, 42, 84},
		APIKeyIDs:     []int64{7, 8, 7},
		Keywords:      []string{" incident ", "", "vip", "incident"},
		MinTokens:     &minTokens,
		MaxTokens:     &maxTokens,
		ActiveFrom:    &activeFrom,
		ActiveTo:      &activeTo,
	}

	require.NoError(t, rule.Validate())
	require.Equal(t, "Incident rule", rule.Name)
	require.Equal(t, []string{"gpt-4.1", "claude-*"}, rule.ModelPatterns)
	require.Equal(t, []int64{42, 84}, rule.UserIDs)
	require.Equal(t, []int64{7, 8}, rule.APIKeyIDs)
	require.Equal(t, []string{"incident", "vip"}, rule.Keywords)
	require.Equal(t, 1.0, rule.SamplingRatio)
}

func TestModelTraceCaptureRuleValidateRejectsInvalidRanges(t *testing.T) {
	t.Parallel()

	minTokens := int64(30)
	maxTokens := int64(20)
	activeFrom := time.Date(2026, 5, 27, 2, 0, 0, 0, time.UTC)
	activeTo := activeFrom.Add(-time.Hour)
	rule := &ModelTraceCaptureRule{
		Name:          "invalid",
		UserIDs:       []int64{1},
		MinTokens:     &minTokens,
		MaxTokens:     &maxTokens,
		SamplingRatio: 1.5,
		ActiveFrom:    &activeFrom,
		ActiveTo:      &activeTo,
	}

	err := rule.Validate()
	require.EqualError(t, err, "min_tokens must be <= max_tokens")

	minTokens = 1
	maxTokens = 2
	err = rule.Validate()
	require.EqualError(t, err, "sampling_ratio must be > 0 and <= 1")

	rule.SamplingRatio = 0.5
	rule.UserIDs = []int64{-1}
	err = rule.Validate()
	require.EqualError(t, err, "user_ids must contain positive ids")
}
