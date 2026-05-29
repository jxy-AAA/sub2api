package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	TaodingTraceExportType      = "sub2api-taoding-traces"
	TaodingTraceExportVersion   = 1
	TaodingTraceSchemaVersion   = "taoding.requirement.v1"
	TaodingTraceScaffoldName    = "sub2api"
	TaodingTraceScaffoldVersion = "sub2api-taoding-trace-v1"
)

type ModelInteractionTraceService struct {
	repo        ModelInteractionTraceRepository
	captureRepo ModelTraceCaptureRepository
	ruleRepo    ModelTraceCaptureRuleRepository
	now         func() time.Time
}

func NewModelInteractionTraceService(
	repo ModelInteractionTraceRepository,
	captureRepo ModelTraceCaptureRepository,
	ruleRepo ModelTraceCaptureRuleRepository,
) *ModelInteractionTraceService {
	return &ModelInteractionTraceService{
		repo:        repo,
		captureRepo: captureRepo,
		ruleRepo:    ruleRepo,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

type GatewayTraceRecordInput struct {
	UserID    *int64
	APIKeyID  *int64
	GroupID   *int64
	AccountID *int64
	RequestID string
	Method    string
	Path      string
}

type TaodingTraceExportBundle struct {
	Type          string             `json:"type"`
	Version       int                `json:"version"`
	SchemaVersion string             `json:"schema_version"`
	ExportedAt    string             `json:"exported_at"`
	Count         int                `json:"count"`
	Records       []CodexTraceExport `json:"records"`
}

func (s *ModelInteractionTraceService) RecordGatewayTraceCaptures(ctx context.Context, entries []GatewayTraceCaptureEntry, input GatewayTraceRecordInput) (bool, error) {
	if s == nil || (s.repo == nil && s.captureRepo == nil) || len(entries) == 0 {
		return false, nil
	}
	trace, err := BuildCodexTraceExportFromGatewayCaptures(entries, input)
	if err != nil {
		return false, err
	}
	if s.captureRepo == nil {
		return s.RecordTraceExport(ctx, trace, input)
	}

	capture, err := modelTraceCaptureFromGatewayTrace(trace, entries, input)
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}

	var rules []*ModelTraceCaptureRule
	if s.ruleRepo != nil {
		rules, err = s.ruleRepo.List(ctx)
		if err != nil {
			return false, err
		}
	}
	matchedRule, shouldCapture := SelectModelTraceCaptureRule(capture, rules, now)
	if !shouldCapture {
		return false, nil
	}
	if matchedRule != nil {
		ruleID := matchedRule.ID
		capture.CaptureRuleID = &ruleID
	}

	stored, err := s.captureRepo.Create(ctx, capture)
	if err != nil {
		return false, err
	}
	return stored, nil
}

func (s *ModelInteractionTraceService) RecordTraceExport(ctx context.Context, trace *CodexTraceExport, input GatewayTraceRecordInput) (bool, error) {
	if s == nil || s.repo == nil || trace == nil {
		return false, nil
	}
	if err := trace.Validate(); err != nil {
		return false, err
	}
	item := modelInteractionTraceFromExport(trace, input)
	return s.repo.Create(ctx, item)
}

func (s *ModelInteractionTraceService) Export(ctx context.Context, now time.Time) (*TaodingTraceExportBundle, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	bundle := &TaodingTraceExportBundle{
		Type:          TaodingTraceExportType,
		Version:       TaodingTraceExportVersion,
		SchemaVersion: TaodingTraceSchemaVersion,
		ExportedAt:    now.UTC().Format(time.RFC3339Nano),
		Records:       []CodexTraceExport{},
	}
	if s == nil || (s.repo == nil && s.captureRepo == nil) {
		return bundle, nil
	}

	bundle.Records = make([]CodexTraceExport, 0)
	if s.captureRepo != nil {
		page := 1
		for {
			items, result, err := s.captureRepo.List(ctx, ModelTraceCaptureListFilter{}, pagination.PaginationParams{Page: page, PageSize: 1000})
			if err != nil {
				return nil, err
			}
			for _, item := range items {
				record := codexTraceExportFromModelTraceCapture(item)
				if err := record.Validate(); err != nil {
					return nil, fmt.Errorf("invalid stored trace %q: %w", item.TaskID, err)
				}
				bundle.Records = append(bundle.Records, record)
			}
			if result == nil || int64(page*1000) >= result.Total || len(items) == 0 {
				break
			}
			page++
		}
	} else if s.repo != nil {
		items, err := s.repo.ListAll(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			record := codexTraceExportFromModelInteractionTrace(item)
			if err := record.Validate(); err != nil {
				return nil, fmt.Errorf("invalid stored trace %q: %w", item.TaskID, err)
			}
			bundle.Records = append(bundle.Records, record)
		}
	}
	bundle.Count = len(bundle.Records)
	return bundle, nil
}

func BuildCodexTraceExportFromGatewayCaptures(entries []GatewayTraceCaptureEntry, input GatewayTraceRecordInput) (*CodexTraceExport, error) {
	for _, entry := range entries {
		if entry.BodyTruncated {
			return nil, fmt.Errorf("gateway trace capture %s was truncated", entry.Stage)
		}
	}

	client := firstGatewayTraceEntry(entries, GatewayTraceStageClientRequest)
	if client == nil || strings.TrimSpace(client.Body) == "" {
		return nil, fmt.Errorf("gateway trace client request is missing")
	}

	if trace, err := ParseCodexTraceExport([]byte(client.Body)); err == nil {
		return trace, nil
	}

	requestRaw := []byte(strings.TrimSpace(client.Body))
	if !json.Valid(requestRaw) {
		return nil, fmt.Errorf("gateway trace client request is not valid JSON")
	}
	var request map[string]json.RawMessage
	if err := json.Unmarshal(requestRaw, &request); err != nil {
		return nil, fmt.Errorf("parse gateway trace request: %w", err)
	}

	response := lastGatewayTraceEntry(entries, GatewayTraceStageUpstreamResponse)
	responseRaw := []byte(nil)
	if response != nil {
		responseRaw = []byte(strings.TrimSpace(response.Body))
	}

	prompt := deriveTracePrompt(request, requestRaw)
	tools := deriveTraceTools(request)
	candidates := deriveTraceCandidates(responseRaw)
	signature := deriveTraceSignature(request, responseRaw)
	meta := buildGatewayTraceMeta(entries, input, request, client, response)
	scaffold := buildGatewayTraceScaffold(entries, requestRaw, responseRaw)

	trace := &CodexTraceExport{
		TaskID:          deriveTraceTaskID(request, responseRaw, input),
		Prompt:          prompt,
		Candidates:      candidates,
		Tools:           tools,
		Signature:       signature,
		Meta:            meta,
		Scaffold:        scaffold,
		ScaffoldVersion: TaodingTraceScaffoldVersion,
	}
	if err := trace.Validate(); err != nil {
		return nil, err
	}
	return trace, nil
}

func modelInteractionTraceFromExport(trace *CodexTraceExport, input GatewayTraceRecordInput) *ModelInteractionTrace {
	item := &ModelInteractionTrace{
		TaskID:          strings.TrimSpace(trace.TaskID),
		Prompt:          cloneRawJSON(trace.Prompt),
		Candidates:      cloneRawJSON(trace.Candidates),
		Tools:           cloneRawJSON(trace.Tools),
		Signature:       cloneRawJSON(trace.Signature),
		Meta:            cloneRawJSON(trace.Meta),
		Scaffold:        cloneRawJSON(trace.Scaffold),
		ScaffoldVersion: strings.TrimSpace(trace.ScaffoldVersion),
		UserID:          input.UserID,
		APIKeyID:        input.APIKeyID,
		DedupeHash:      trace.DedupeHash(),
	}
	if requestID := strings.TrimSpace(input.RequestID); requestID != "" {
		item.RequestID = &requestID
	}
	if model := rawJSONObjectString(trace.Meta, "model"); model != "" {
		item.Model = &model
	}
	return item
}

func codexTraceExportFromModelInteractionTrace(item *ModelInteractionTrace) CodexTraceExport {
	if item == nil {
		return CodexTraceExport{}
	}
	return CodexTraceExport{
		TaskID:          item.TaskID,
		Prompt:          cloneRawJSON(item.Prompt),
		Candidates:      cloneRawJSON(item.Candidates),
		Tools:           cloneRawJSON(item.Tools),
		Signature:       cloneRawJSON(item.Signature),
		Meta:            cloneRawJSON(item.Meta),
		Scaffold:        cloneRawJSON(item.Scaffold),
		ScaffoldVersion: item.ScaffoldVersion,
	}
}

func firstGatewayTraceEntry(entries []GatewayTraceCaptureEntry, stage string) *GatewayTraceCaptureEntry {
	for i := range entries {
		if entries[i].Stage == stage {
			return &entries[i]
		}
	}
	return nil
}

func lastGatewayTraceEntry(entries []GatewayTraceCaptureEntry, stage string) *GatewayTraceCaptureEntry {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Stage == stage && strings.TrimSpace(entries[i].Body) != "" {
			return &entries[i]
		}
	}
	return nil
}

func deriveTracePrompt(request map[string]json.RawMessage, requestRaw []byte) json.RawMessage {
	if raw := rawArrayField(request, "prompt"); len(raw) > 0 {
		return raw
	}
	if raw := rawArrayField(request, "messages"); len(raw) > 0 {
		return prependSystemMessageIfPresent(request["system"], raw)
	}
	if raw := rawArrayField(request, "input"); len(raw) > 0 {
		return prependSystemMessageIfPresent(request["instructions"], raw)
	}
	if raw := rawStringField(request, "input"); len(raw) > 0 {
		return mustTraceRawJSON([]map[string]any{{
			"role":    "user",
			"content": string(raw),
		}})
	}
	return mustTraceRawJSON([]map[string]any{{
		"role": "user",
		"content": []map[string]any{{
			"type": "raw_request",
			"json": json.RawMessage(requestRaw),
		}},
	}})
}

func prependSystemMessageIfPresent(system json.RawMessage, messages json.RawMessage) json.RawMessage {
	system = bytes.TrimSpace(system)
	messages = bytes.TrimSpace(messages)
	if len(system) == 0 || bytes.Equal(system, []byte("null")) {
		return cloneRawJSON(messages)
	}
	if len(messages) < 2 || messages[0] != '[' || messages[len(messages)-1] != ']' {
		return cloneRawJSON(messages)
	}

	systemMessage := append([]byte(`{"role":"system","content":`), system...)
	systemMessage = append(systemMessage, '}')
	inner := bytes.TrimSpace(messages[1 : len(messages)-1])
	out := make([]byte, 0, len(systemMessage)+len(inner)+3)
	out = append(out, '[')
	out = append(out, systemMessage...)
	if len(inner) > 0 {
		out = append(out, ',')
		out = append(out, inner...)
	}
	out = append(out, ']')
	return out
}

func deriveTraceTools(request map[string]json.RawMessage) json.RawMessage {
	if raw := rawArrayField(request, "tools"); len(raw) > 0 {
		return normalizeAnthropicTools(raw)
	}
	if raw := rawArrayField(request, "functions"); len(raw) > 0 {
		return normalizeLegacyOpenAIFunctions(raw)
	}
	return json.RawMessage(`[]`)
}

func normalizeAnthropicTools(raw json.RawMessage) json.RawMessage {
	var tools []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tools); err != nil || len(tools) == 0 {
		return cloneRawJSON(raw)
	}
	first := tools[0]
	if _, ok := first["input_schema"]; !ok {
		return cloneRawJSON(raw)
	}

	normalized := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		name := rawJSONToString(tool["name"])
		if name == "" {
			continue
		}
		fn := map[string]any{
			"name":       name,
			"parameters": json.RawMessage(bytes.TrimSpace(tool["input_schema"])),
		}
		if description := rawJSONToString(tool["description"]); description != "" {
			fn["description"] = description
		}
		normalized = append(normalized, map[string]any{
			"type":     "function",
			"function": fn,
		})
	}
	if len(normalized) == 0 {
		return cloneRawJSON(raw)
	}
	return mustTraceRawJSON(normalized)
}

func normalizeLegacyOpenAIFunctions(raw json.RawMessage) json.RawMessage {
	var functions []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &functions); err != nil {
		return cloneRawJSON(raw)
	}
	tools := make([]map[string]any, 0, len(functions))
	for _, fn := range functions {
		tools = append(tools, map[string]any{
			"type":     "function",
			"function": fn,
		})
	}
	return mustTraceRawJSON(tools)
}

func deriveTraceCandidates(responseRaw []byte) json.RawMessage {
	responseRaw = bytes.TrimSpace(responseRaw)
	if len(responseRaw) == 0 {
		return json.RawMessage(`[]`)
	}
	if json.Valid(responseRaw) {
		var response map[string]json.RawMessage
		if err := json.Unmarshal(responseRaw, &response); err == nil {
			if raw := rawArrayField(response, "candidates"); len(raw) > 0 {
				return raw
			}
			if raw := rawArrayField(response, "choices"); len(raw) > 0 {
				return raw
			}
			if output := bytes.TrimSpace(response["output"]); len(output) > 0 && !bytes.Equal(output, []byte("null")) {
				return mustTraceRawJSON([]map[string]any{{
					"output":       json.RawMessage(output),
					"raw_response": json.RawMessage(responseRaw),
				}})
			}
			if content := bytes.TrimSpace(response["content"]); len(content) > 0 && !bytes.Equal(content, []byte("null")) {
				return mustTraceRawJSON([]map[string]any{{
					"content":      json.RawMessage(content),
					"raw_response": json.RawMessage(responseRaw),
				}})
			}
		}
		return mustTraceRawJSON([]map[string]any{{
			"raw_response": json.RawMessage(responseRaw),
		}})
	}

	events := parseSSEJSONEvents(string(responseRaw))
	if len(events) > 0 {
		return mustTraceRawJSON([]map[string]any{{
			"events": events,
		}})
	}
	return mustTraceRawJSON([]map[string]any{{
		"raw_response_text": string(responseRaw),
	}})
}

func parseSSEJSONEvents(raw string) []json.RawMessage {
	lines := strings.Split(raw, "\n")
	events := make([]json.RawMessage, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		if json.Valid([]byte(data)) {
			events = append(events, json.RawMessage(data))
		}
	}
	return events
}

func deriveTraceSignature(request map[string]json.RawMessage, responseRaw []byte) json.RawMessage {
	for _, key := range []string{"signature", "thinking_signature"} {
		if raw := bytes.TrimSpace(request[key]); len(raw) > 0 && !bytes.Equal(raw, []byte("null")) {
			return cloneRawJSON(raw)
		}
	}
	if json.Valid(responseRaw) {
		var response map[string]json.RawMessage
		if err := json.Unmarshal(responseRaw, &response); err == nil {
			for _, key := range []string{"signature", "thinking_signature"} {
				if raw := bytes.TrimSpace(response[key]); len(raw) > 0 && !bytes.Equal(raw, []byte("null")) {
					return cloneRawJSON(raw)
				}
			}
		}
	}
	return mustTraceRawJSON(map[string]any{
		"available": false,
		"source":    "sub2api_gateway_capture",
		"reason":    "upstream payload did not expose a thinking signature",
	})
}

func buildGatewayTraceMeta(entries []GatewayTraceCaptureEntry, input GatewayTraceRecordInput, request map[string]json.RawMessage, client *GatewayTraceCaptureEntry, response *GatewayTraceCaptureEntry) json.RawMessage {
	bodyTruncated := false
	for _, entry := range entries {
		if entry.BodyTruncated {
			bodyTruncated = true
			break
		}
	}

	meta := map[string]any{
		"source":             "sub2api_gateway_capture",
		"schema_version":     TaodingTraceSchemaVersion,
		"scaffold":           TaodingTraceScaffoldName,
		"scaffold_version":   TaodingTraceScaffoldVersion,
		"capture_count":      len(entries),
		"export_contract":    "taoding_pdf_requirement",
		"dedupe_fields":      []string{"prompt", "candidates", "tools"},
		"body_truncated":     bodyTruncated,
		"capture_incomplete": client == nil || response == nil,
	}
	if input.RequestID != "" {
		meta["request_id"] = input.RequestID
	}
	if input.Method != "" {
		meta["method"] = input.Method
	}
	if input.Path != "" {
		meta["path"] = input.Path
	}
	if input.UserID != nil {
		meta["user_id"] = *input.UserID
	}
	if input.APIKeyID != nil {
		meta["api_key_id"] = *input.APIKeyID
	}
	if input.GroupID != nil {
		meta["group_id"] = *input.GroupID
	}
	if input.AccountID != nil {
		meta["account_id"] = *input.AccountID
	}
	if model := rawJSONObjectStringFromMap(request, "model"); model != "" {
		meta["model"] = model
	}
	if client != nil {
		meta["protocol"] = client.Protocol
		meta["client_captured_at"] = client.CapturedAt
		mergeGatewayEntryMeta(meta, client.Meta)
	}
	if response != nil {
		meta["upstream_status_code"] = response.StatusCode
		meta["upstream_captured_at"] = response.CapturedAt
		mergeGatewayEntryMeta(meta, response.Meta)
	}
	return mustTraceRawJSON(meta)
}

func mergeGatewayEntryMeta(dst map[string]any, src map[string]any) {
	for key, value := range src {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := dst[key]; exists {
			dst["gateway_"+key] = value
			continue
		}
		dst[key] = value
	}
}

func buildGatewayTraceScaffold(entries []GatewayTraceCaptureEntry, requestRaw []byte, responseRaw []byte) json.RawMessage {
	scaffold := map[string]any{
		"source":   "sub2api_gateway_capture",
		"captures": entries,
	}
	if len(requestRaw) > 0 && json.Valid(requestRaw) {
		scaffold["client_request_json"] = json.RawMessage(requestRaw)
	}
	if len(responseRaw) > 0 && json.Valid(responseRaw) {
		scaffold["upstream_response_json"] = json.RawMessage(responseRaw)
	} else if len(responseRaw) > 0 {
		scaffold["upstream_response_text"] = string(responseRaw)
	}
	return mustTraceRawJSON(scaffold)
}

func deriveTraceTaskID(request map[string]json.RawMessage, responseRaw []byte, input GatewayTraceRecordInput) string {
	for _, key := range []string{"task_id", "id"} {
		if value := rawJSONObjectStringFromMap(request, key); value != "" {
			return value
		}
	}
	if value := rawNestedString(request["metadata"], "task_id"); value != "" {
		return value
	}
	if json.Valid(responseRaw) {
		var response map[string]json.RawMessage
		if err := json.Unmarshal(responseRaw, &response); err == nil {
			if value := rawJSONObjectStringFromMap(response, "id"); value != "" {
				return value
			}
		}
	}
	if requestID := strings.TrimSpace(input.RequestID); requestID != "" {
		return requestID
	}

	h := sha256.New()
	for _, key := range []string{"model", "messages", "input", "prompt"} {
		h.Write(bytes.TrimSpace(request[key]))
		h.Write([]byte{0})
	}
	return "sub2api-trace-" + hex.EncodeToString(h.Sum(nil))[:24]
}

func rawArrayField(values map[string]json.RawMessage, key string) json.RawMessage {
	raw := bytes.TrimSpace(values[key])
	if len(raw) > 0 && raw[0] == '[' && json.Valid(raw) {
		return cloneRawJSON(raw)
	}
	return nil
}

func rawStringField(values map[string]json.RawMessage, key string) string {
	return rawJSONToString(values[key])
}

func rawJSONObjectString(raw json.RawMessage, key string) string {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return ""
	}
	return rawJSONObjectStringFromMap(values, key)
}

func rawJSONObjectStringFromMap(values map[string]json.RawMessage, key string) string {
	return rawJSONToString(values[key])
}

func rawNestedString(raw json.RawMessage, key string) string {
	if len(bytes.TrimSpace(raw)) == 0 {
		return ""
	}
	return rawJSONObjectString(raw, key)
}

func rawJSONToString(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || !json.Valid(raw) {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return json.RawMessage(out)
}

func mustTraceRawJSON(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}
