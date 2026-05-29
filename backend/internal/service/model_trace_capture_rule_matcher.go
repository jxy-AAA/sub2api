package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"path"
	"strings"
	"time"
)

// SelectModelTraceCaptureRule evaluates enabled capture rules in repository
// order and returns the first rule that fully matches the capture. When no
// enabled rules exist, capture is enabled by default so a missing rule set does
// not silently disable trace collection.
func SelectModelTraceCaptureRule(capture *ModelTraceCapture, rules []*ModelTraceCaptureRule, now time.Time) (*ModelTraceCaptureRule, bool) {
	if capture == nil {
		return nil, false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	hasEnabledRule := false
	for _, rule := range rules {
		if rule != nil && rule.Enabled {
			hasEnabledRule = true
			break
		}
	}
	if !hasEnabledRule {
		return nil, true
	}

	for _, rule := range rules {
		if rule != nil && rule.Matches(capture, now) {
			return rule, true
		}
	}
	return nil, false
}

func (rule *ModelTraceCaptureRule) Matches(capture *ModelTraceCapture, now time.Time) bool {
	if rule == nil || capture == nil || !rule.Enabled {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if rule.ActiveFrom != nil && now.Before(rule.ActiveFrom.UTC()) {
		return false
	}
	if rule.ActiveTo != nil && now.After(rule.ActiveTo.UTC()) {
		return false
	}
	if len(rule.ModelPatterns) > 0 && !traceRuleModelMatches(rule.ModelPatterns, capture.Model, capture.RequestedModel, capture.UpstreamModel) {
		return false
	}
	if len(rule.UserIDs) > 0 && !traceRuleIDMatches(rule.UserIDs, capture.UserID) {
		return false
	}
	if len(rule.APIKeyIDs) > 0 && !traceRuleIDMatches(rule.APIKeyIDs, capture.APIKeyID) {
		return false
	}
	if len(rule.Keywords) > 0 && !traceRuleKeywordMatches(rule.Keywords, capture) {
		return false
	}
	totalTokens, hasTokens := traceCaptureComparableTokens(capture)
	if (rule.MinTokens != nil || rule.MaxTokens != nil) && !hasTokens {
		return false
	}
	if rule.MinTokens != nil && totalTokens < *rule.MinTokens {
		return false
	}
	if rule.MaxTokens != nil && totalTokens > *rule.MaxTokens {
		return false
	}
	return traceRuleSamplingAllows(rule, capture)
}

func traceRuleModelMatches(patterns []string, values ...any) bool {
	for _, value := range values {
		var model string
		switch v := value.(type) {
		case string:
			model = v
		case *string:
			if v != nil {
				model = *v
			}
		}
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		for _, pattern := range patterns {
			if traceRulePatternMatches(pattern, model) {
				return true
			}
		}
	}
	return false
}

func traceRulePatternMatches(pattern, value string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	value = strings.ToLower(strings.TrimSpace(value))
	if pattern == "" || value == "" {
		return false
	}
	if !strings.ContainsAny(pattern, "*?[]") {
		return pattern == value
	}
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}

func traceRuleIDMatches(ids []int64, value *int64) bool {
	if value == nil || *value <= 0 {
		return false
	}
	for _, id := range ids {
		if id == *value {
			return true
		}
	}
	return false
}

func traceRuleKeywordMatches(keywords []string, capture *ModelTraceCapture) bool {
	haystack := strings.ToLower(strings.Join([]string{
		capture.Model,
		traceRuleOptionalString(capture.RequestedModel),
		traceRuleOptionalString(capture.UpstreamModel),
		string(bytes.TrimSpace(capture.Prompt)),
		string(bytes.TrimSpace(capture.Candidates)),
		string(bytes.TrimSpace(capture.Tools)),
		string(bytes.TrimSpace(capture.Signature)),
		string(bytes.TrimSpace(capture.Meta)),
		string(bytes.TrimSpace(capture.RawRequest)),
		string(bytes.TrimSpace(capture.RawResponse)),
		capture.RawRequestText,
		capture.RawResponseText,
	}, "\n"))
	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(haystack, keyword) {
			return true
		}
	}
	return false
}

func traceCaptureComparableTokens(capture *ModelTraceCapture) (int64, bool) {
	if capture == nil {
		return 0, false
	}
	if capture.TotalTokens != nil {
		return *capture.TotalTokens, true
	}
	var total int64
	hasValue := false
	if capture.InputTokens != nil {
		total += *capture.InputTokens
		hasValue = true
	}
	if capture.OutputTokens != nil {
		total += *capture.OutputTokens
		hasValue = true
	}
	return total, hasValue
}

func traceRuleSamplingAllows(rule *ModelTraceCaptureRule, capture *ModelTraceCapture) bool {
	if rule == nil {
		return false
	}
	ratio := rule.SamplingRatio
	if ratio <= 0 {
		ratio = 1
	}
	if ratio >= 1 {
		return true
	}

	h := sha256.New()
	var idBuf [8]byte
	binary.BigEndian.PutUint64(idBuf[:], uint64(rule.ID))
	_, _ = h.Write(idBuf[:])
	_, _ = h.Write([]byte(strings.TrimSpace(capture.DedupeHash)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.TrimSpace(capture.TaskID)))
	sum := h.Sum(nil)
	bucket := binary.BigEndian.Uint64(sum[:8])
	fraction := float64(bucket) / float64(^uint64(0))
	return fraction < ratio
}

func traceRuleOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
