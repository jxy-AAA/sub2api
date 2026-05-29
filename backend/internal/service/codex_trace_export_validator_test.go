package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCodexTraceExport_PreservesRawNestedJSON(t *testing.T) {
	t.Parallel()

	rawPrompt := strings.TrimSpace(`[
  {"role":"system","content":[{"type":"input_text","text":"follow the original payload"}]},
  {"role":"assistant","content":[{"type":"output_text","text":"thinking"},{"type":"tool_call","tool_calls":[{"id":"call_2","function":{"name":"edit","arguments":"{\"path\":\"a.go\",\"diff\":\"@@ -1 +1 @@\"}"}}]}]},
  {"role":"tool","content":[{"type":"tool_result","tool_results":[{"tool_use_id":"call_2","content":{"raw":true,"items":[3,2,1]}}]}]}
]`)
	rawCandidates := strings.TrimSpace(`[
  {"id":"candidate_1","content":[{"type":"output_text","text":"done"},{"type":"tool_call","tool_calls":[{"id":"call_2","function":{"name":"edit","arguments":"{\"path\":\"a.go\",\"diff\":\"@@ -1 +1 @@\"}"}}]}],"raw":{"content":["keep","exact"]}}
]`)
	rawTools := strings.TrimSpace(`[
  {"type":"function","name":"edit","description":"apply patch","parameters":{"type":"object","properties":{"path":{"type":"string"},"diff":{"type":"string"}},"required":["path","diff"]}}
]`)
	rawSignature := strings.TrimSpace(`{"alg":"sha256","value":"sig_abc123"}`)
	rawMeta := strings.TrimSpace(`{"source":"codex","admin_only":true,"root_only":true}`)
	rawScaffold := strings.TrimSpace(`{"root":{"messages":["keep","raw"],"tool_calls":[{"id":"call_2"}]}}`)

	exportJSON := fmt.Sprintf(`{"task_id":"task_123","prompt":%s,"candidates":%s,"tools":%s,"signature":%s,"meta":%s,"scaffold":%s,"scaffold_version":"2026-05-27"}`,
		rawPrompt,
		rawCandidates,
		rawTools,
		rawSignature,
		rawMeta,
		rawScaffold,
	)

	trace, err := ParseCodexTraceExport([]byte(exportJSON))
	require.NoError(t, err)
	require.Equal(t, "task_123", trace.TaskID)
	require.Equal(t, "2026-05-27", trace.ScaffoldVersion)
	require.Equal(t, rawPrompt, string(trace.Prompt))
	require.Equal(t, rawCandidates, string(trace.Candidates))
	require.Equal(t, rawTools, string(trace.Tools))
	require.Equal(t, rawSignature, string(trace.Signature))
	require.Equal(t, rawMeta, string(trace.Meta))
	require.Equal(t, rawScaffold, string(trace.Scaffold))

	hash, err := CodexTraceExportDedupeHash([]byte(exportJSON))
	require.NoError(t, err)
	require.Equal(t, trace.DedupeHash(), hash)
}

func TestParseCodexTraceExport_RequiresAllTopLevelFields(t *testing.T) {
	t.Parallel()

	requiredFields := []string{
		"task_id",
		"prompt",
		"candidates",
		"tools",
		"signature",
		"meta",
		"scaffold",
		"scaffold_version",
	}

	for _, field := range requiredFields {
		field := field
		t.Run(field, func(t *testing.T) {
			payload := validCodexTraceExportFixture()
			delete(payload, field)

			raw, err := json.Marshal(payload)
			require.NoError(t, err)

			_, err = ParseCodexTraceExport(raw)
			require.Error(t, err)
			require.Contains(t, err.Error(), field)
		})
	}
}

func TestParseCodexTraceExport_ValidatesContainerTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		field      string
		value      any
		wantErrSub string
	}{
		{name: "prompt must be array", field: "prompt", value: "not-an-array", wantErrSub: "prompt must be a JSON array"},
		{name: "candidates must be array", field: "candidates", value: map[string]any{"id": "candidate_1"}, wantErrSub: "candidates must be a JSON array"},
		{name: "tools must be array", field: "tools", value: map[string]any{"type": "function"}, wantErrSub: "tools must be a JSON array"},
		{name: "meta must be object", field: "meta", value: []any{"not", "an", "object"}, wantErrSub: "meta must be a JSON object"},
		{name: "scaffold must be object", field: "scaffold", value: []any{"not", "an", "object"}, wantErrSub: "scaffold must be a JSON object"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			payload := validCodexTraceExportFixture()
			payload[tc.field] = tc.value

			raw, err := json.Marshal(payload)
			require.NoError(t, err)

			_, err = ParseCodexTraceExport(raw)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErrSub)
		})
	}
}

func TestCodexTraceExportDedupeHash_StableAcrossTopLevelFieldOrder(t *testing.T) {
	t.Parallel()

	rawPrompt := `[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]`
	rawCandidates := `[{"id":"candidate_1","content":[{"type":"tool_call","tool_calls":[{"id":"call_1","function":{"name":"shell","arguments":"{\"cmd\":\"pwd\"}"}}]}]}]`
	rawTools := `[{"type":"function","name":"shell","parameters":{"type":"object"}}]`
	rawSignature := `{"alg":"sha256","value":"sig"}`
	rawMeta := `{"source":"codex","admin_only":true}`
	rawScaffold := `{"root":{"tool_results":[{"tool_use_id":"call_1","content":"ok"}]}}`

	first := fmt.Sprintf(`{"task_id":"task_abc","prompt":%s,"candidates":%s,"tools":%s,"signature":%s,"meta":%s,"scaffold":%s,"scaffold_version":"v1"}`,
		rawPrompt, rawCandidates, rawTools, rawSignature, rawMeta, rawScaffold)
	second := fmt.Sprintf(`{"scaffold_version":"v1","meta":%s,"signature":%s,"tools":%s,"candidates":%s,"prompt":%s,"task_id":"task_abc","scaffold":%s}`,
		rawMeta, rawSignature, rawTools, rawCandidates, rawPrompt, rawScaffold)

	hash1, err := CodexTraceExportDedupeHash([]byte(first))
	require.NoError(t, err)
	hash2, err := CodexTraceExportDedupeHash([]byte(second))
	require.NoError(t, err)
	require.Equal(t, hash1, hash2)
}

func TestCodexTraceExportDedupeHash_ChangesWhenRawPromptChanges(t *testing.T) {
	t.Parallel()

	base := `{"task_id":"task_abc","prompt":[{"role":"assistant","content":[{"type":"tool_result","tool_results":[{"tool_use_id":"call_1","content":{"items":[3,2,1]}}]}]}],"candidates":[{"id":"candidate_1"}],"tools":[{"type":"function","name":"shell","parameters":{"type":"object"}}],"signature":{"alg":"sha256","value":"sig"},"meta":{"source":"codex"},"scaffold":{"root":{"tool_calls":[{"id":"call_1"}]}},"scaffold_version":"v1"}`
	changed := strings.Replace(base, `[3,2,1]`, `[1,2,3]`, 1)

	hash1, err := CodexTraceExportDedupeHash([]byte(base))
	require.NoError(t, err)
	hash2, err := CodexTraceExportDedupeHash([]byte(changed))
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash2)
}

func TestCodexTraceExportDedupeHash_IgnoresNonDedupeMetadata(t *testing.T) {
	t.Parallel()

	base := `{"task_id":"task_abc","prompt":[{"role":"user","content":"hello"}],"candidates":[{"id":"candidate_1"}],"tools":[{"type":"function","name":"shell","parameters":{"type":"object"}}],"signature":{"value":"sig-1"},"meta":{"source":"codex","model":"a"},"scaffold":{"raw":"short"},"scaffold_version":"v1"}`
	richer := `{"task_id":"task_xyz","prompt":[{"role":"user","content":"hello"}],"candidates":[{"id":"candidate_1"}],"tools":[{"type":"function","name":"shell","parameters":{"type":"object"}}],"signature":{"value":"sig-2","extra":"more"},"meta":{"source":"codex","model":"b","request_id":"req-2"},"scaffold":{"raw":"short","captures":[{"body":"more complete"}]},"scaffold_version":"v2"}`

	hash1, err := CodexTraceExportDedupeHash([]byte(base))
	require.NoError(t, err)
	hash2, err := CodexTraceExportDedupeHash([]byte(richer))
	require.NoError(t, err)
	require.Equal(t, hash1, hash2)
}

func validCodexTraceExportFixture() map[string]any {
	return map[string]any{
		"task_id": "task_123",
		"prompt": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": "hello"},
				},
			},
		},
		"candidates": []any{
			map[string]any{
				"id": "candidate_1",
				"content": []any{
					map[string]any{"type": "output_text", "text": "done"},
				},
			},
		},
		"tools": []any{
			map[string]any{
				"type":       "function",
				"name":       "shell",
				"parameters": map[string]any{"type": "object"},
			},
		},
		"signature": map[string]any{
			"alg":   "sha256",
			"value": "sig_abc123",
		},
		"meta": map[string]any{
			"source":     "codex",
			"admin_only": true,
		},
		"scaffold": map[string]any{
			"root": map[string]any{
				"task_id": "task_123",
			},
		},
		"scaffold_version": "2026-05-27",
	}
}
