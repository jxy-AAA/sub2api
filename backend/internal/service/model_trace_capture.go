package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// ModelTraceCapture stores one trace record in the model_trace_captures table.
// JSON blobs are stored losslessly; export views derive human-readable slices
// from the preserved prompt array.
type ModelTraceCapture struct {
	ID int64 `json:"id"`

	TaskID string `json:"task_id"`

	RequestID  *string `json:"request_id,omitempty"`
	ResponseID *string `json:"response_id,omitempty"`

	UserID        *int64 `json:"user_id,omitempty"`
	APIKeyID      *int64 `json:"api_key_id,omitempty"`
	GroupID       *int64 `json:"group_id,omitempty"`
	AccountID     *int64 `json:"account_id,omitempty"`
	CaptureRuleID *int64 `json:"capture_rule_id,omitempty"`

	Protocol string `json:"protocol"`
	Model    string `json:"model"`

	RequestedModel *string `json:"requested_model,omitempty"`
	UpstreamModel  *string `json:"upstream_model,omitempty"`

	RequestContentType  string `json:"request_content_type,omitempty"`
	ResponseContentType string `json:"response_content_type,omitempty"`

	InputTokens        *int64 `json:"input_tokens,omitempty"`
	OutputTokens       *int64 `json:"output_tokens,omitempty"`
	TotalTokens        *int64 `json:"total_tokens,omitempty"`
	UpstreamStatusCode *int   `json:"upstream_status_code,omitempty"`

	Scaffold        string `json:"scaffold"`
	ScaffoldVersion string `json:"scaffold_version"`

	Prompt     json.RawMessage `json:"prompt"`
	Candidates json.RawMessage `json:"candidates"`
	Tools      json.RawMessage `json:"tools"`
	Signature  json.RawMessage `json:"signature"`
	Meta       json.RawMessage `json:"meta"`

	RawRequest      json.RawMessage `json:"raw_request,omitempty"`
	RawResponse     json.RawMessage `json:"raw_response,omitempty"`
	RawRequestText  string          `json:"raw_request_text,omitempty"`
	RawResponseText string          `json:"raw_response_text,omitempty"`

	DedupeHash string    `json:"dedupe_hash"`
	PromptHash string    `json:"prompt_hash"`
	CreatedAt  time.Time `json:"created_at"`
}

// ModelTraceCaptureExportOptions controls optional export fields.
type ModelTraceCaptureExportOptions struct {
	IncludeRaw bool
}

// ModelTraceCaptureExport is the JSON export shape for root/admin download.
// Raw request/response payloads stay excluded unless explicitly requested.
type ModelTraceCaptureExport struct {
	ID int64 `json:"id"`

	TaskID     string  `json:"task_id"`
	RequestID  *string `json:"request_id,omitempty"`
	ResponseID *string `json:"response_id,omitempty"`

	UserID        *int64 `json:"user_id,omitempty"`
	APIKeyID      *int64 `json:"api_key_id,omitempty"`
	GroupID       *int64 `json:"group_id,omitempty"`
	AccountID     *int64 `json:"account_id,omitempty"`
	CaptureRuleID *int64 `json:"capture_rule_id,omitempty"`

	Protocol string `json:"protocol"`
	Model    string `json:"model"`

	RequestedModel *string `json:"requested_model,omitempty"`
	UpstreamModel  *string `json:"upstream_model,omitempty"`

	RequestContentType  string `json:"request_content_type,omitempty"`
	ResponseContentType string `json:"response_content_type,omitempty"`

	InputTokens        *int64 `json:"input_tokens,omitempty"`
	OutputTokens       *int64 `json:"output_tokens,omitempty"`
	TotalTokens        *int64 `json:"total_tokens,omitempty"`
	UpstreamStatusCode *int   `json:"upstream_status_code,omitempty"`

	Scaffold        string `json:"scaffold"`
	ScaffoldVersion string `json:"scaffold_version"`

	System    json.RawMessage `json:"system,omitempty"`
	User      json.RawMessage `json:"user,omitempty"`
	Tool      json.RawMessage `json:"tool,omitempty"`
	Assistant json.RawMessage `json:"assistant,omitempty"`

	Prompt     json.RawMessage `json:"prompt,omitempty"`
	Candidates json.RawMessage `json:"candidates,omitempty"`
	Tools      json.RawMessage `json:"tools,omitempty"`
	Signature  json.RawMessage `json:"signature,omitempty"`
	Meta       json.RawMessage `json:"meta,omitempty"`

	RawRequest             json.RawMessage `json:"raw_request,omitempty"`
	RawUpstreamRequest     json.RawMessage `json:"raw_upstream_request,omitempty"`
	RawResponse            json.RawMessage `json:"raw_response,omitempty"`
	RawRequestText         string          `json:"raw_request_text,omitempty"`
	RawUpstreamRequestText string          `json:"raw_upstream_request_text,omitempty"`
	RawResponseText        string          `json:"raw_response_text,omitempty"`

	DedupeHash string    `json:"dedupe_hash"`
	PromptHash string    `json:"prompt_hash"`
	CreatedAt  time.Time `json:"created_at"`
}

type ModelTraceCaptureListFilter struct {
	Model           string
	UserID          *int64
	APIKeyID        *int64
	CaptureRuleID   *int64
	StartTime       *time.Time
	EndTime         *time.Time
	Keyword         string
	MinInputTokens  *int64
	MaxInputTokens  *int64
	MinOutputTokens *int64
	MaxOutputTokens *int64
	MinTotalTokens  *int64
	MaxTotalTokens  *int64
}

type ModelTraceCaptureRepository interface {
	Create(ctx context.Context, capture *ModelTraceCapture) (bool, error)
	GetByID(ctx context.Context, id int64) (*ModelTraceCapture, error)
	GetByTaskID(ctx context.Context, taskID string) (*ModelTraceCapture, error)
	GetByDedupeHash(ctx context.Context, dedupeHash string) (*ModelTraceCapture, error)
	List(ctx context.Context, filter ModelTraceCaptureListFilter, params pagination.PaginationParams) ([]*ModelTraceCapture, *pagination.PaginationResult, error)
	ListByTimeRange(ctx context.Context, startTime, endTime time.Time, params pagination.PaginationParams) ([]*ModelTraceCapture, *pagination.PaginationResult, error)
	DeleteByID(ctx context.Context, id int64) (bool, error)
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error)
}

// Validate normalizes and validates the record before persistence.
func (c *ModelTraceCapture) Validate() error {
	if c == nil {
		return fmt.Errorf("model trace capture is nil")
	}
	c.TaskID = strings.TrimSpace(c.TaskID)
	c.Protocol = strings.TrimSpace(c.Protocol)
	c.Model = strings.TrimSpace(c.Model)
	c.Scaffold = strings.TrimSpace(c.Scaffold)
	c.ScaffoldVersion = strings.TrimSpace(c.ScaffoldVersion)
	if c.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if c.Protocol == "" {
		return fmt.Errorf("protocol is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	c.RequestContentType = strings.TrimSpace(c.RequestContentType)
	c.ResponseContentType = strings.TrimSpace(c.ResponseContentType)
	if c.Scaffold == "" {
		return fmt.Errorf("scaffold is required")
	}
	if c.ScaffoldVersion == "" {
		return fmt.Errorf("scaffold_version is required")
	}

	c.Candidates = defaultTraceCaptureJSON(c.Candidates, `[]`)
	c.Tools = defaultTraceCaptureJSON(c.Tools, `[]`)
	c.Signature = defaultTraceCaptureJSON(c.Signature, `[]`)
	c.Meta = defaultTraceCaptureJSON(c.Meta, `{}`)

	if err := requireTraceCaptureJSONArray("prompt", c.Prompt); err != nil {
		return err
	}
	if err := requireTraceCaptureJSONArray("candidates", c.Candidates); err != nil {
		return err
	}
	if err := requireTraceCaptureJSONArray("tools", c.Tools); err != nil {
		return err
	}
	if err := requireTraceCaptureJSON("signature", c.Signature); err != nil {
		return err
	}
	if err := requireTraceCaptureJSONObject("meta", c.Meta); err != nil {
		return err
	}
	if len(bytes.TrimSpace(c.RawRequest)) > 0 && !json.Valid(bytes.TrimSpace(c.RawRequest)) {
		return fmt.Errorf("raw_request must be valid JSON")
	}
	if len(bytes.TrimSpace(c.RawResponse)) > 0 && !json.Valid(bytes.TrimSpace(c.RawResponse)) {
		return fmt.Errorf("raw_response must be valid JSON")
	}
	if err := validateTraceCaptureOptionalInt64("input_tokens", c.InputTokens); err != nil {
		return err
	}
	if err := validateTraceCaptureOptionalInt64("output_tokens", c.OutputTokens); err != nil {
		return err
	}
	if err := validateTraceCaptureOptionalInt64("total_tokens", c.TotalTokens); err != nil {
		return err
	}
	if c.UpstreamStatusCode != nil {
		if *c.UpstreamStatusCode < 100 || *c.UpstreamStatusCode > 999 {
			return fmt.Errorf("upstream_status_code must be between 100 and 999")
		}
	}

	if strings.TrimSpace(c.PromptHash) == "" {
		c.PromptHash = c.ComputePromptHash()
	}
	if strings.TrimSpace(c.DedupeHash) == "" {
		c.DedupeHash = c.ComputeDedupeHash()
	}
	if len(strings.TrimSpace(c.PromptHash)) != 64 {
		return fmt.Errorf("prompt_hash must be a 64-character hex digest")
	}
	if len(strings.TrimSpace(c.DedupeHash)) != 64 {
		return fmt.Errorf("dedupe_hash must be a 64-character hex digest")
	}
	return nil
}

// ComputePromptHash returns a stable hash for the raw prompt JSON.
func (c *ModelTraceCapture) ComputePromptHash() string {
	if c == nil {
		return ""
	}
	return traceCaptureHash(traceHashPart{label: "prompt", value: normalizeTraceCaptureJSON(c.Prompt)})
}

// ComputeDedupeHash returns a stable hash for prompt + candidates + tools.
func (c *ModelTraceCapture) ComputeDedupeHash() string {
	if c == nil {
		return ""
	}
	return traceCaptureHash(
		traceHashPart{label: "prompt", value: normalizeTraceCaptureJSON(c.Prompt)},
		traceHashPart{label: "candidates", value: normalizeTraceCaptureJSON(c.Candidates)},
		traceHashPart{label: "tools", value: normalizeTraceCaptureJSON(c.Tools)},
	)
}

// Export converts the capture into the root/admin JSON export shape.
func (c *ModelTraceCapture) Export(opts ModelTraceCaptureExportOptions) *ModelTraceCaptureExport {
	if c == nil {
		return nil
	}

	export := &ModelTraceCaptureExport{
		ID:                  c.ID,
		TaskID:              c.TaskID,
		RequestID:           traceCaptureCloneStringPtr(c.RequestID),
		ResponseID:          traceCaptureCloneStringPtr(c.ResponseID),
		UserID:              traceCaptureCloneInt64Ptr(c.UserID),
		APIKeyID:            traceCaptureCloneInt64Ptr(c.APIKeyID),
		GroupID:             traceCaptureCloneInt64Ptr(c.GroupID),
		AccountID:           traceCaptureCloneInt64Ptr(c.AccountID),
		CaptureRuleID:       traceCaptureCloneInt64Ptr(c.CaptureRuleID),
		Protocol:            c.Protocol,
		Model:               c.Model,
		RequestedModel:      traceCaptureCloneStringPtr(c.RequestedModel),
		UpstreamModel:       traceCaptureCloneStringPtr(c.UpstreamModel),
		RequestContentType:  c.RequestContentType,
		ResponseContentType: c.ResponseContentType,
		InputTokens:         traceCaptureCloneInt64Ptr(c.InputTokens),
		OutputTokens:        traceCaptureCloneInt64Ptr(c.OutputTokens),
		TotalTokens:         traceCaptureCloneInt64Ptr(c.TotalTokens),
		UpstreamStatusCode:  traceCaptureCloneIntPtr(c.UpstreamStatusCode),
		Scaffold:            c.Scaffold,
		ScaffoldVersion:     c.ScaffoldVersion,
		Prompt:              traceCaptureCloneRawMessage(c.Prompt),
		Candidates:          traceCaptureCloneRawMessage(c.Candidates),
		Tools:               traceCaptureCloneRawMessage(c.Tools),
		Signature:           traceCaptureCloneRawMessage(c.Signature),
		Meta:                traceCaptureExportMeta(c.Meta, opts.IncludeRaw),
		DedupeHash:          c.DedupeHash,
		PromptHash:          c.PromptHash,
		CreatedAt:           c.CreatedAt,
	}

	export.System = firstTraceMessageByRole(c.Prompt, "system")
	export.User = traceMessagesByRole(c.Prompt, "user")
	export.Tool = traceMessagesByRole(c.Prompt, "tool")
	export.Assistant = traceMessagesByRole(c.Prompt, "assistant")

	if opts.IncludeRaw {
		export.RawRequest = traceCaptureCloneRawMessage(c.RawRequest)
		export.RawUpstreamRequest, export.RawUpstreamRequestText = traceCaptureRawStageFromMeta(c.Meta, GatewayTraceStageUpstreamRequest)
		export.RawResponse = traceCaptureCloneRawMessage(c.RawResponse)
		export.RawRequestText = c.RawRequestText
		export.RawResponseText = c.RawResponseText
	}

	return export
}

func traceCaptureExportMeta(meta json.RawMessage, includeRaw bool) json.RawMessage {
	if includeRaw {
		return traceCaptureCloneRawMessage(meta)
	}
	trimmed := bytes.TrimSpace(meta)
	if len(trimmed) == 0 || !json.Valid(trimmed) || trimmed[0] != '{' {
		return traceCaptureCloneRawMessage(meta)
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &values); err != nil {
		return traceCaptureCloneRawMessage(meta)
	}
	delete(values, traceCaptureScaffoldJSONMetaKey)
	out, err := json.Marshal(values)
	if err != nil {
		return traceCaptureCloneRawMessage(meta)
	}
	return out
}

func traceCaptureRawStageFromMeta(meta json.RawMessage, stage string) (json.RawMessage, string) {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		return nil, ""
	}
	scaffold := traceRawField(meta, traceCaptureScaffoldJSONMetaKey)
	if len(bytes.TrimSpace(scaffold)) == 0 {
		return nil, ""
	}

	var payload struct {
		Captures []struct {
			Stage string `json:"stage"`
			Body  string `json:"body"`
		} `json:"captures"`
	}
	if err := json.Unmarshal(scaffold, &payload); err != nil {
		return nil, ""
	}
	for i := range payload.Captures {
		if strings.TrimSpace(payload.Captures[i].Stage) != stage {
			continue
		}
		body := payload.Captures[i].Body
		trimmed := bytes.TrimSpace([]byte(body))
		if len(trimmed) == 0 {
			return nil, ""
		}
		if json.Valid(trimmed) {
			return traceCaptureCloneRawMessage(trimmed), body
		}
		return nil, body
	}
	return nil, ""
}

func validateTraceCaptureOptionalInt64(name string, value *int64) error {
	if value == nil {
		return nil
	}
	if *value < 0 {
		return fmt.Errorf("%s must be >= 0", name)
	}
	return nil
}

func defaultTraceCaptureJSON(raw json.RawMessage, defaultValue string) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage(defaultValue)
	}
	return append(json.RawMessage(nil), trimmed...)
}

func requireTraceCaptureJSON(name string, raw json.RawMessage) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return fmt.Errorf("%s is required", name)
	}
	if !json.Valid(trimmed) {
		return fmt.Errorf("%s must be valid JSON", name)
	}
	return nil
}

func requireTraceCaptureJSONArray(name string, raw json.RawMessage) error {
	if err := requireTraceCaptureJSON(name, raw); err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(raw)
	if trimmed[0] != '[' {
		return fmt.Errorf("%s must be a JSON array", name)
	}
	return nil
}

func requireTraceCaptureJSONObject(name string, raw json.RawMessage) error {
	if err := requireTraceCaptureJSON(name, raw); err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(raw)
	if trimmed[0] != '{' {
		return fmt.Errorf("%s must be a JSON object", name)
	}
	return nil
}

func normalizeTraceCaptureJSON(raw json.RawMessage) []byte {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	var v any
	if err := json.Unmarshal(trimmed, &v); err != nil {
		return append([]byte(nil), trimmed...)
	}

	out, err := json.Marshal(v)
	if err != nil {
		return append([]byte(nil), trimmed...)
	}
	return out
}

type traceHashPart struct {
	label string
	value []byte
}

func traceCaptureHash(parts ...traceHashPart) string {
	h := sha256.New()
	var sizeBuf [8]byte
	for _, part := range parts {
		_, _ = h.Write([]byte(part.label))
		_, _ = h.Write([]byte{0})
		binary.BigEndian.PutUint64(sizeBuf[:], uint64(len(part.value)))
		_, _ = h.Write(sizeBuf[:])
		_, _ = h.Write(part.value)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func traceCaptureCloneRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func traceCaptureCloneStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func traceCaptureCloneInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func traceCaptureCloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func firstTraceMessageByRole(raw json.RawMessage, role string) json.RawMessage {
	messages, err := splitTraceMessages(raw)
	if err != nil || len(messages) == 0 {
		return nil
	}
	for _, message := range messages {
		if traceMessageRole(message) == role {
			return traceCaptureCloneRawMessage(message)
		}
	}
	return nil
}

func traceMessagesByRole(raw json.RawMessage, role string) json.RawMessage {
	messages, err := splitTraceMessages(raw)
	if err != nil || len(messages) == 0 {
		return nil
	}

	filtered := make([]json.RawMessage, 0, len(messages))
	for _, msg := range messages {
		if traceMessageRole(msg) == role {
			filtered = append(filtered, msg)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	rawOut, err := json.Marshal(filtered)
	if err != nil {
		return nil
	}
	return rawOut
}

func splitTraceMessages(raw json.RawMessage) ([]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("prompt must be a JSON array")
	}

	var messages []json.RawMessage
	if err := json.Unmarshal(trimmed, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func traceMessageRole(raw json.RawMessage) string {
	var payload struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(payload.Role))
}
