package service

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	GatewayTraceCapturesKey = "gateway_trace_captures"

	// 0 means lossless capture. The Taoding export contract requires complete
	// model interaction payloads, including tool calls/results and reasoning.
	gatewayTraceBodyMaxBytes = 0
)

const (
	GatewayTraceStageClientRequest    = "client_request"
	GatewayTraceStageUpstreamRequest  = "upstream_request"
	GatewayTraceStageUpstreamResponse = "upstream_response"
)

type GatewayTraceCaptureEntry struct {
	Stage         string         `json:"stage"`
	Protocol      string         `json:"protocol,omitempty"`
	ContentType   string         `json:"content_type,omitempty"`
	StatusCode    int            `json:"status_code,omitempty"`
	Body          string         `json:"body,omitempty"`
	BodyTruncated bool           `json:"body_truncated,omitempty"`
	CapturedAt    string         `json:"captured_at"`
	Meta          map[string]any `json:"meta,omitempty"`
}

type gatewayTraceCaptureInput struct {
	stage       string
	protocol    string
	contentType string
	statusCode  int
	body        []byte
	meta        map[string]any
}

type gatewayTraceBodyBuffer struct {
	limit     int
	buf       bytes.Buffer
	truncated bool
}

func newGatewayTraceBodyBuffer() *gatewayTraceBodyBuffer {
	return &gatewayTraceBodyBuffer{limit: gatewayTraceBodyMaxBytes}
}

func (b *gatewayTraceBodyBuffer) WriteLine(line string) {
	if b == nil {
		return
	}
	b.WriteString(line)
	b.WriteString("\n")
}

func (b *gatewayTraceBodyBuffer) WriteString(s string) {
	if b == nil || s == "" {
		return
	}
	if b.limit <= 0 {
		_, _ = b.buf.WriteString(s)
		return
	}
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return
	}
	if len(s) > remaining {
		_, _ = b.buf.WriteString(s[:remaining])
		b.truncated = true
		return
	}
	_, _ = b.buf.WriteString(s)
}

func (b *gatewayTraceBodyBuffer) Bytes() []byte {
	if b == nil || b.buf.Len() == 0 {
		return nil
	}
	return b.buf.Bytes()
}

func (b *gatewayTraceBodyBuffer) Truncated() bool {
	if b == nil {
		return false
	}
	return b.truncated
}

func GetGatewayTraceCaptures(c *gin.Context) []GatewayTraceCaptureEntry {
	if c == nil {
		return nil
	}
	raw, ok := c.Get(GatewayTraceCapturesKey)
	if !ok {
		return nil
	}
	entries, ok := raw.([]GatewayTraceCaptureEntry)
	if !ok || len(entries) == 0 {
		return nil
	}
	out := make([]GatewayTraceCaptureEntry, len(entries))
	copy(out, entries)
	return out
}

func captureGatewayClientRequest(c *gin.Context, protocol string, body []byte, meta map[string]any) {
	captureGatewayTrace(c, gatewayTraceCaptureInput{
		stage:       GatewayTraceStageClientRequest,
		protocol:    protocol,
		contentType: "application/json",
		body:        body,
		meta:        meta,
	})
}

func inferGatewayTraceProtocol(c *gin.Context, prefix string) string {
	path := ""
	if c != nil && c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	return inferGatewayTraceProtocolFromPath(path, prefix)
}

func inferGatewayTraceProtocolFromPath(path string, prefix string) string {
	prefix = strings.TrimSpace(prefix)
	path = strings.ToLower(strings.TrimSpace(path))
	if prefix == "" {
		prefix = "gateway"
	}
	switch {
	case strings.Contains(path, "/chat/completions"):
		return prefix + ".chat_completions"
	case strings.Contains(path, "/responses"):
		return prefix + ".responses"
	case strings.Contains(path, "/messages"):
		return prefix + ".messages"
	case strings.Contains(path, ":streamgeneratecontent"):
		return prefix + ".stream_generate_content"
	case strings.Contains(path, ":generatecontent"):
		return prefix + ".generate_content"
	case strings.Contains(path, ":counttokens"):
		return prefix + ".count_tokens"
	case strings.Contains(path, "/images/generations"):
		return prefix + ".images_generations"
	case strings.Contains(path, "/images/edits"):
		return prefix + ".images_edits"
	case strings.Contains(path, "/models"):
		return prefix + ".models"
	default:
		return prefix
	}
}

func captureGatewayUpstreamRequest(c *gin.Context, protocol string, req *http.Request, body []byte, meta map[string]any) {
	mergedMeta := cloneGatewayTraceMeta(meta)
	if req != nil {
		if mergedMeta == nil {
			mergedMeta = make(map[string]any, 2)
		}
		if method := strings.TrimSpace(req.Method); method != "" {
			mergedMeta["method"] = method
		}
		if req.URL != nil {
			if url := safeUpstreamURL(req.URL.String()); url != "" {
				mergedMeta["url"] = url
			}
		}
	}
	contentType := ""
	if req != nil {
		contentType = strings.TrimSpace(req.Header.Get("Content-Type"))
	}
	captureGatewayTrace(c, gatewayTraceCaptureInput{
		stage:       GatewayTraceStageUpstreamRequest,
		protocol:    protocol,
		contentType: contentType,
		body:        body,
		meta:        mergedMeta,
	})
}

func captureGatewayUpstreamResponse(c *gin.Context, protocol string, resp *http.Response, body []byte, meta map[string]any) {
	mergedMeta := cloneGatewayTraceMeta(meta)
	contentType := ""
	statusCode := 0
	if resp != nil {
		contentType = strings.TrimSpace(resp.Header.Get("Content-Type"))
		statusCode = resp.StatusCode
		if mergedMeta == nil {
			mergedMeta = make(map[string]any, 1)
		}
		if requestID := strings.TrimSpace(resp.Header.Get("x-request-id")); requestID != "" {
			mergedMeta["upstream_request_id"] = requestID
		}
	}
	captureGatewayTrace(c, gatewayTraceCaptureInput{
		stage:       GatewayTraceStageUpstreamResponse,
		protocol:    protocol,
		contentType: contentType,
		statusCode:  statusCode,
		body:        body,
		meta:        mergedMeta,
	})
}

func captureGatewayUpstreamResponseBuffer(c *gin.Context, protocol string, resp *http.Response, buf *gatewayTraceBodyBuffer, meta map[string]any) {
	if buf == nil {
		return
	}
	input := gatewayTraceCaptureInput{
		stage:      GatewayTraceStageUpstreamResponse,
		protocol:   protocol,
		statusCode: 0,
		body:       buf.Bytes(),
		meta:       cloneGatewayTraceMeta(meta),
	}
	if resp != nil {
		input.contentType = strings.TrimSpace(resp.Header.Get("Content-Type"))
		input.statusCode = resp.StatusCode
		if input.meta == nil {
			input.meta = make(map[string]any, 1)
		}
		if requestID := strings.TrimSpace(resp.Header.Get("x-request-id")); requestID != "" {
			input.meta["upstream_request_id"] = requestID
		}
	}
	captureGatewayTraceWithTruncation(c, input, buf.Truncated())
}

func captureGatewayTrace(c *gin.Context, input gatewayTraceCaptureInput) {
	captureGatewayTraceWithTruncation(c, input, false)
}

func captureGatewayTraceWithTruncation(c *gin.Context, input gatewayTraceCaptureInput, truncated bool) {
	if c == nil {
		return
	}
	defer func() {
		_ = recover()
	}()

	body, bodyTruncated := limitGatewayTraceBody(input.body)
	meta := buildGatewayTraceEntryMeta(input.contentType, []byte(body), input.meta)
	allowEmpty := shouldPreserveGatewayTraceEmptyBody(input.contentType, []byte(body))
	entry := GatewayTraceCaptureEntry{
		Stage:         strings.TrimSpace(input.stage),
		Protocol:      strings.TrimSpace(input.protocol),
		ContentType:   strings.TrimSpace(input.contentType),
		StatusCode:    input.statusCode,
		Body:          body,
		BodyTruncated: truncated || bodyTruncated,
		CapturedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Meta:          meta,
	}
	if entry.Stage == "" || (entry.Body == "" && !allowEmpty) {
		return
	}

	var existing []GatewayTraceCaptureEntry
	if raw, ok := c.Get(GatewayTraceCapturesKey); ok {
		if entries, ok := raw.([]GatewayTraceCaptureEntry); ok {
			existing = entries
		}
	}
	existing = append(existing, entry)
	c.Set(GatewayTraceCapturesKey, existing)
}

func cloneGatewayTraceMeta(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		dst[key] = value
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func limitGatewayTraceBody(body []byte) (string, bool) {
	if len(body) == 0 {
		return "", false
	}
	if gatewayTraceBodyMaxBytes <= 0 {
		return string(body), false
	}
	if len(body) <= gatewayTraceBodyMaxBytes {
		return string(body), false
	}
	return string(body[:gatewayTraceBodyMaxBytes]), true
}

func buildGatewayTraceEntryMeta(contentType string, body []byte, meta map[string]any) map[string]any {
	cloned := cloneGatewayTraceMeta(meta)
	if !isGatewayTraceEventStream(contentType, body) {
		return cloned
	}
	if cloned == nil {
		cloned = make(map[string]any, 3)
	}
	eventCount, terminated := summarizeGatewayTraceEventStream(body)
	cloned["raw_transport"] = "sse"
	cloned["raw_event_count"] = eventCount
	cloned["raw_stream_terminated"] = terminated
	return cloned
}

func shouldPreserveGatewayTraceEmptyBody(contentType string, body []byte) bool {
	return isGatewayTraceEventStream(contentType, body)
}

func isGatewayTraceEventStream(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(strings.TrimSpace(contentType)), "text/event-stream") {
		return true
	}
	trimmed := strings.TrimSpace(string(body))
	return strings.HasPrefix(trimmed, "data:") || strings.HasPrefix(trimmed, "event:")
}

func summarizeGatewayTraceEventStream(body []byte) (int, bool) {
	if len(body) == 0 {
		return 0, true
	}

	eventCount := 0
	inEvent := false
	terminated := true

	reader := bufio.NewReader(bytes.NewReader(body))
	for {
		rawLine, err := reader.ReadBytes('\n')
		if len(rawLine) > 0 {
			line := bytes.TrimSuffix(rawLine, []byte("\n"))
			line = bytes.TrimSuffix(line, []byte("\r"))
			switch {
			case len(line) == 0:
				if inEvent {
					eventCount++
					inEvent = false
				}
			case bytes.HasPrefix(line, []byte("event:")) || bytes.HasPrefix(line, []byte("data:")):
				inEvent = true
			}
		}
		if err == nil {
			continue
		}
		if err == io.EOF {
			break
		}
		terminated = false
		break
	}

	if inEvent {
		eventCount++
		terminated = false
	}

	return eventCount, terminated
}
