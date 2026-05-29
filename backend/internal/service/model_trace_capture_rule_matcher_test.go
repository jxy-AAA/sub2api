package service

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestModelTraceCaptureRuleMatchesAllConfiguredCriteria(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	activeFrom := now.Add(-time.Hour)
	activeTo := now.Add(time.Hour)
	minTokens := int64(150)
	maxTokens := int64(250)
	rule := &ModelTraceCaptureRule{
		ID:            7,
		Name:          "incident window",
		Enabled:       true,
		ModelPatterns: []string{"gpt-*"},
		UserIDs:       []int64{42},
		APIKeyIDs:     []int64{9},
		Keywords:      []string{"incident", "vip"},
		MinTokens:     &minTokens,
		MaxTokens:     &maxTokens,
		SamplingRatio: 1,
		ActiveFrom:    &activeFrom,
		ActiveTo:      &activeTo,
	}
	require.NoError(t, rule.Validate())

	capture := newTraceRuleTestCapture()
	require.True(t, rule.Matches(capture, now))

	missingTokens := *capture
	missingTokens.TotalTokens = nil
	missingTokens.InputTokens = nil
	missingTokens.OutputTokens = nil
	require.False(t, rule.Matches(&missingTokens, now))

	outsideWindow := now.Add(2 * time.Hour)
	require.False(t, rule.Matches(capture, outsideWindow))
}

func TestSelectModelTraceCaptureRuleRequiresEnabledMatchAndFallsBackAfterSamplingMiss(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	capture := newTraceRuleTestCapture()

	rule, ok := SelectModelTraceCaptureRule(capture, nil, now)
	require.True(t, ok)
	require.Nil(t, rule)

	highPriority := &ModelTraceCaptureRule{
		ID:            11,
		Name:          "sampled",
		Enabled:       true,
		Priority:      20,
		ModelPatterns: []string{"gpt-*"},
		SamplingRatio: 1,
	}
	lowPriority := &ModelTraceCaptureRule{
		ID:            12,
		Name:          "fallback",
		Enabled:       true,
		Priority:      10,
		ModelPatterns: []string{"gpt-*"},
		SamplingRatio: 1,
	}

	fraction := traceRuleSamplingFraction(highPriority, capture)
	require.GreaterOrEqual(t, fraction, 0.0)
	require.LessOrEqual(t, fraction, 1.0)
	highPriority.SamplingRatio = math.Max(math.SmallestNonzeroFloat64, fraction/2)

	matched, ok := SelectModelTraceCaptureRule(capture, []*ModelTraceCaptureRule{
		highPriority,
		lowPriority,
	}, now)
	require.True(t, ok)
	require.NotNil(t, matched)
	require.Equal(t, lowPriority.ID, matched.ID)
}

func newTraceRuleTestCapture() *ModelTraceCapture {
	userID := int64(42)
	apiKeyID := int64(9)
	totalTokens := int64(200)
	capture := &ModelTraceCapture{
		TaskID:          "task-match-001",
		UserID:          &userID,
		APIKeyID:        &apiKeyID,
		Protocol:        "openai.responses",
		Model:           "gpt-4.1",
		Scaffold:        "sub2api",
		ScaffoldVersion: TaodingTraceScaffoldVersion,
		Prompt:          json.RawMessage(`[{"role":"user","content":"critical incident report"}]`),
		Candidates:      json.RawMessage(`[{"message":{"role":"assistant","content":"vip escalation"}}]`),
		Tools:           json.RawMessage(`[]`),
		Signature:       json.RawMessage(`{"available":false}`),
		Meta:            json.RawMessage(`{"model":"gpt-4.1"}`),
		TotalTokens:     &totalTokens,
	}
	_ = capture.Validate()
	return capture
}

func traceRuleSamplingFraction(rule *ModelTraceCaptureRule, capture *ModelTraceCapture) float64 {
	h := sha256.New()
	var idBuf [8]byte
	binary.BigEndian.PutUint64(idBuf[:], uint64(rule.ID))
	_, _ = h.Write(idBuf[:])
	_, _ = h.Write([]byte(strings.TrimSpace(capture.DedupeHash)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.TrimSpace(capture.TaskID)))
	sum := h.Sum(nil)
	bucket := binary.BigEndian.Uint64(sum[:8])
	return float64(bucket) / float64(^uint64(0))
}
