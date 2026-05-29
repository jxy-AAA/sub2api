package service

import (
	"bytes"
	"encoding/json"
	"strings"
)

const traceCaptureScaffoldJSONMetaKey = "scaffold_json"

func modelTraceCaptureFromGatewayTrace(trace *CodexTraceExport, entries []GatewayTraceCaptureEntry, input GatewayTraceRecordInput) (*ModelTraceCapture, error) {
	if trace == nil {
		return nil, nil
	}
	if err := trace.Validate(); err != nil {
		return nil, err
	}

	client := firstGatewayTraceEntry(entries, GatewayTraceStageClientRequest)
	upstreamRequest := firstGatewayTraceEntry(entries, GatewayTraceStageUpstreamRequest)
	response := lastGatewayTraceEntry(entries, GatewayTraceStageUpstreamResponse)

	requestRaw := gatewayTraceEntryBodyBytes(client)
	responseRaw := gatewayTraceEntryBodyBytes(response)
	requestBodyText := gatewayTraceEntryBodyText(client)
	responseBodyText := gatewayTraceEntryBodyText(response)
	requestMap := traceJSONMap(requestRaw)
	responseMap := traceJSONMap(responseRaw)
	responseInfo := extractTraceResponseInfo(responseRaw)

	requestedModel := traceCaptureFirstNonEmptyString(
		rawJSONObjectStringFromMap(requestMap, "model"),
		rawJSONObjectString(trace.Meta, "model"),
	)
	upstreamModel := traceCaptureFirstNonEmptyString(
		responseInfo.Model,
		rawJSONObjectStringFromMap(responseMap, "model"),
		traceNestedJSONString(responseMap["response"], "model"),
	)
	model := traceCaptureFirstNonEmptyString(upstreamModel, requestedModel, rawJSONObjectString(trace.Meta, "model"), "unknown")
	accountID := firstNonZeroInt64(
		traceMetaInt64(client, "account_id"),
		traceMetaInt64(upstreamRequest, "account_id"),
		traceMetaInt64(response, "account_id"),
	)
	if input.AccountID != nil && *input.AccountID > 0 {
		accountID = *input.AccountID
	}

	capture := &ModelTraceCapture{
		TaskID:              strings.TrimSpace(trace.TaskID),
		RequestID:           traceStringPtr(traceCaptureFirstNonEmptyString(input.RequestID, rawJSONObjectString(trace.Meta, "request_id"))),
		ResponseID:          traceStringPtr(traceCaptureFirstNonEmptyString(responseInfo.ResponseID, rawJSONObjectStringFromMap(responseMap, "id"), traceNestedJSONString(responseMap["response"], "id"))),
		UserID:              cloneTraceExportTaskInt64Ptr(input.UserID),
		APIKeyID:            cloneTraceExportTaskInt64Ptr(input.APIKeyID),
		GroupID:             cloneTraceExportTaskInt64Ptr(input.GroupID),
		AccountID:           traceInt64Ptr(accountID),
		Protocol:            traceCaptureFirstNonEmptyString(traceEntryProtocol(response), traceEntryProtocol(upstreamRequest), traceEntryProtocol(client), "gateway"),
		Model:               model,
		RequestedModel:      traceStringPtr(requestedModel),
		UpstreamModel:       traceStringPtr(upstreamModel),
		RequestContentType:  traceCaptureFirstNonEmptyString(traceEntryContentType(client), traceEntryContentType(upstreamRequest)),
		ResponseContentType: traceEntryContentType(response),
		Scaffold:            traceCaptureFirstNonEmptyString(rawJSONObjectString(trace.Meta, "scaffold"), TaodingTraceScaffoldName),
		ScaffoldVersion:     strings.TrimSpace(trace.ScaffoldVersion),
		Prompt:              cloneRawJSON(trace.Prompt),
		Candidates:          cloneRawJSON(trace.Candidates),
		Tools:               cloneRawJSON(trace.Tools),
		Signature:           cloneRawJSON(trace.Signature),
		Meta:                enrichTraceCaptureMeta(trace.Meta, trace.Scaffold),
	}

	if response != nil && response.StatusCode > 0 {
		status := response.StatusCode
		capture.UpstreamStatusCode = &status
	}
	capture.InputTokens = cloneTraceExportTaskInt64Ptr(responseInfo.InputTokens)
	capture.OutputTokens = cloneTraceExportTaskInt64Ptr(responseInfo.OutputTokens)
	capture.TotalTokens = cloneTraceExportTaskInt64Ptr(responseInfo.TotalTokens)

	if strings.TrimSpace(requestBodyText) != "" {
		capture.RawRequestText = requestBodyText
	}
	if len(requestRaw) > 0 {
		if json.Valid(requestRaw) {
			capture.RawRequest = cloneRawJSON(requestRaw)
		}
	}
	if strings.TrimSpace(responseBodyText) != "" {
		capture.RawResponseText = responseBodyText
	}
	if len(responseRaw) > 0 {
		if json.Valid(responseRaw) {
			capture.RawResponse = cloneRawJSON(responseRaw)
		}
	}

	if err := capture.Validate(); err != nil {
		return nil, err
	}
	return capture, nil
}

func codexTraceExportFromModelTraceCapture(item *ModelTraceCapture) CodexTraceExport {
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
		Scaffold:        traceCaptureScaffoldForExport(item),
		ScaffoldVersion: item.ScaffoldVersion,
	}
}

func traceCaptureScaffoldForExport(item *ModelTraceCapture) json.RawMessage {
	if item == nil {
		return json.RawMessage(`{}`)
	}
	if raw := traceRawField(item.Meta, traceCaptureScaffoldJSONMetaKey); len(raw) > 0 && json.Valid(raw) && bytes.TrimSpace(raw)[0] == '{' {
		return cloneRawJSON(raw)
	}

	scaffold := map[string]any{
		"source":           "model_trace_captures",
		"capture_id":       item.ID,
		"scaffold":         item.Scaffold,
		"scaffold_version": item.ScaffoldVersion,
	}
	if len(bytes.TrimSpace(item.RawRequest)) > 0 {
		scaffold["client_request_json"] = json.RawMessage(bytes.TrimSpace(item.RawRequest))
	} else if strings.TrimSpace(item.RawRequestText) != "" {
		scaffold["client_request_text"] = item.RawRequestText
	}
	if len(bytes.TrimSpace(item.RawResponse)) > 0 {
		scaffold["upstream_response_json"] = json.RawMessage(bytes.TrimSpace(item.RawResponse))
	} else if strings.TrimSpace(item.RawResponseText) != "" {
		scaffold["upstream_response_text"] = item.RawResponseText
	}
	return mustTraceRawJSON(scaffold)
}

func enrichTraceCaptureMeta(meta json.RawMessage, scaffold json.RawMessage) json.RawMessage {
	trimmedMeta := bytes.TrimSpace(meta)
	if len(trimmedMeta) == 0 || !json.Valid(trimmedMeta) || trimmedMeta[0] != '{' {
		trimmedMeta = json.RawMessage(`{}`)
	}

	var values map[string]json.RawMessage
	if err := json.Unmarshal(trimmedMeta, &values); err != nil {
		return cloneRawJSON(trimmedMeta)
	}
	if len(bytes.TrimSpace(scaffold)) > 0 && json.Valid(scaffold) {
		values[traceCaptureScaffoldJSONMetaKey] = cloneRawJSON(scaffold)
	}
	out, err := json.Marshal(values)
	if err != nil {
		return cloneRawJSON(trimmedMeta)
	}
	return out
}

func gatewayTraceEntryBodyBytes(entry *GatewayTraceCaptureEntry) []byte {
	if entry == nil {
		return nil
	}
	return bytes.TrimSpace([]byte(entry.Body))
}

func gatewayTraceEntryBodyText(entry *GatewayTraceCaptureEntry) string {
	if entry == nil {
		return ""
	}
	return entry.Body
}

func traceEntryProtocol(entry *GatewayTraceCaptureEntry) string {
	if entry == nil {
		return ""
	}
	return strings.TrimSpace(entry.Protocol)
}

func traceEntryContentType(entry *GatewayTraceCaptureEntry) string {
	if entry == nil {
		return ""
	}
	return strings.TrimSpace(entry.ContentType)
}

func traceJSONMap(raw []byte) map[string]json.RawMessage {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || !json.Valid(raw) || raw[0] != '{' {
		return nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return values
}

func traceRawField(raw json.RawMessage, key string) json.RawMessage {
	values := traceJSONMap(raw)
	if len(values) == 0 {
		return nil
	}
	field := bytes.TrimSpace(values[key])
	if len(field) == 0 || bytes.Equal(field, []byte("null")) {
		return nil
	}
	return cloneRawJSON(field)
}

func traceNestedJSONString(raw json.RawMessage, key string) string {
	return rawJSONObjectString(raw, key)
}

func traceMetaInt64(entry *GatewayTraceCaptureEntry, key string) int64 {
	if entry == nil || len(entry.Meta) == 0 {
		return 0
	}
	value, ok := entry.Meta[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case int64:
		if v > 0 {
			return v
		}
	case int:
		if v > 0 {
			return int64(v)
		}
	case float64:
		if v > 0 {
			return int64(v)
		}
	case json.Number:
		if parsed, err := v.Int64(); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func extractTraceUsage(responseRaw []byte) (*int64, *int64, *int64) {
	info := extractTraceResponseInfo(responseRaw)
	return cloneTraceExportTaskInt64Ptr(info.InputTokens), cloneTraceExportTaskInt64Ptr(info.OutputTokens), cloneTraceExportTaskInt64Ptr(info.TotalTokens)
}

type traceResponseInfo struct {
	ResponseID   string
	Model        string
	InputTokens  *int64
	OutputTokens *int64
	TotalTokens  *int64
}

func extractTraceResponseInfo(responseRaw []byte) traceResponseInfo {
	responseRaw = bytes.TrimSpace(responseRaw)
	if len(responseRaw) == 0 {
		return traceResponseInfo{}
	}

	if json.Valid(responseRaw) {
		return extractTraceResponseInfoFromJSON(responseRaw)
	}

	info := traceResponseInfo{}
	for _, event := range parseSSEJSONEvents(string(responseRaw)) {
		info = mergeTraceResponseInfo(info, extractTraceResponseInfoFromJSON(event))
	}
	return finalizeTraceResponseInfo(info)
}

func extractTraceResponseInfoFromJSON(raw json.RawMessage) traceResponseInfo {
	values := traceJSONMap(raw)
	if len(values) == 0 {
		return traceResponseInfo{}
	}

	info := traceResponseInfo{
		ResponseID: traceCaptureFirstNonEmptyString(
			rawJSONObjectStringFromMap(values, "id"),
			traceNestedJSONString(values["response"], "id"),
			traceNestedJSONString(values["message"], "id"),
		),
		Model: traceCaptureFirstNonEmptyString(
			rawJSONObjectStringFromMap(values, "model"),
			traceNestedJSONString(values["response"], "model"),
			traceNestedJSONString(values["message"], "model"),
		),
	}

	info.InputTokens, info.OutputTokens, info.TotalTokens = extractTraceUsageFromUsageJSON(raw)
	if hasTraceUsageInfo(info) {
		return finalizeTraceResponseInfo(info)
	}

	for _, key := range []string{"response", "message"} {
		if nested := traceRawField(raw, key); len(nested) > 0 {
			info = mergeTraceResponseInfo(info, extractTraceResponseInfoFromJSON(nested))
		}
	}
	return finalizeTraceResponseInfo(info)
}

func extractTraceUsageFromUsageJSON(raw json.RawMessage) (*int64, *int64, *int64) {
	usage := traceRawField(raw, "usage")
	if len(usage) == 0 {
		usage = traceRawField(raw, "usageMetadata")
	}
	if len(usage) == 0 {
		return nil, nil, nil
	}

	usageValues := traceJSONMap(usage)
	if len(usageValues) == 0 {
		return nil, nil, nil
	}
	baseInput := firstJSONInt64(usageValues, "input_tokens", "prompt_tokens", "promptTokenCount", "inputTokenCount")
	cacheInput := sumJSONInt64s(usageValues, "cache_creation_input_tokens", "cache_read_input_tokens")
	input := mergeInputTokenCounts(baseInput, cacheInput)
	output := firstJSONInt64(usageValues, "output_tokens", "completion_tokens", "candidatesTokenCount", "outputTokenCount")
	total := firstJSONInt64(usageValues, "total_tokens", "totalTokenCount")
	if total == nil && input != nil && output != nil {
		value := *input + *output
		total = &value
	}
	return input, output, total
}

func firstJSONInt64(values map[string]json.RawMessage, keys ...string) *int64 {
	for _, key := range keys {
		raw := bytes.TrimSpace(values[key])
		if len(raw) == 0 || !json.Valid(raw) {
			continue
		}
		var value int64
		if err := json.Unmarshal(raw, &value); err == nil && value >= 0 {
			return &value
		}
	}
	return nil
}

func sumJSONInt64s(values map[string]json.RawMessage, keys ...string) *int64 {
	var total int64
	found := false
	for _, key := range keys {
		raw := bytes.TrimSpace(values[key])
		if len(raw) == 0 || !json.Valid(raw) {
			continue
		}
		var value int64
		if err := json.Unmarshal(raw, &value); err == nil && value >= 0 {
			total += value
			found = true
		}
	}
	if !found {
		return nil
	}
	return &total
}

func mergeInputTokenCounts(base *int64, extra *int64) *int64 {
	switch {
	case base == nil && extra == nil:
		return nil
	case base == nil:
		return cloneTraceExportTaskInt64Ptr(extra)
	case extra == nil:
		return cloneTraceExportTaskInt64Ptr(base)
	default:
		total := *base + *extra
		return &total
	}
}

func mergeTraceResponseInfo(current traceResponseInfo, incoming traceResponseInfo) traceResponseInfo {
	if current.ResponseID == "" {
		current.ResponseID = strings.TrimSpace(incoming.ResponseID)
	}
	if current.Model == "" {
		current.Model = strings.TrimSpace(incoming.Model)
	}
	if current.InputTokens == nil && incoming.InputTokens != nil {
		current.InputTokens = cloneTraceExportTaskInt64Ptr(incoming.InputTokens)
	}
	if current.OutputTokens == nil && incoming.OutputTokens != nil {
		current.OutputTokens = cloneTraceExportTaskInt64Ptr(incoming.OutputTokens)
	}
	if current.TotalTokens == nil && incoming.TotalTokens != nil {
		current.TotalTokens = cloneTraceExportTaskInt64Ptr(incoming.TotalTokens)
	}
	return current
}

func finalizeTraceResponseInfo(info traceResponseInfo) traceResponseInfo {
	if info.TotalTokens == nil && info.InputTokens != nil && info.OutputTokens != nil {
		total := *info.InputTokens + *info.OutputTokens
		info.TotalTokens = &total
	}
	return info
}

func hasTraceUsageInfo(info traceResponseInfo) bool {
	return info.InputTokens != nil || info.OutputTokens != nil || info.TotalTokens != nil
}

func traceCaptureFirstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func traceStringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func traceInt64Ptr(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}
