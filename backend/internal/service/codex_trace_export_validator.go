package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"strings"
)

// CodexTraceExport is a shape-preserving helper for trace capture/export payloads.
// Raw JSON fields stay byte-for-byte intact so callers can persist or re-emit
// prompt/tools/candidates payloads without summarizing, merging, or reordering.
type CodexTraceExport struct {
	TaskID          string          `json:"task_id"`
	Prompt          json.RawMessage `json:"prompt"`
	Candidates      json.RawMessage `json:"candidates"`
	Tools           json.RawMessage `json:"tools"`
	Signature       json.RawMessage `json:"signature"`
	Meta            json.RawMessage `json:"meta"`
	Scaffold        json.RawMessage `json:"scaffold"`
	ScaffoldVersion string          `json:"scaffold_version"`
}

// ParseCodexTraceExport validates a raw trace export payload and preserves the
// raw JSON bytes for prompt/candidates/tools/signature/meta/scaffold fields.
func ParseCodexTraceExport(raw []byte) (*CodexTraceExport, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("trace export is empty")
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("trace export is invalid JSON")
	}

	var trace CodexTraceExport
	if err := json.Unmarshal(raw, &trace); err != nil {
		return nil, fmt.Errorf("unmarshal trace export: %w", err)
	}
	if err := trace.Validate(); err != nil {
		return nil, err
	}
	return &trace, nil
}

// Validate enforces the required top-level trace export fields and their
// container shapes.
func (t *CodexTraceExport) Validate() error {
	if t == nil {
		return fmt.Errorf("trace export is nil")
	}
	if strings.TrimSpace(t.TaskID) == "" {
		return fmt.Errorf("task_id is required")
	}
	if err := requireCodexTraceJSON("prompt", t.Prompt, codexTraceJSONArray); err != nil {
		return err
	}
	if err := requireCodexTraceJSON("candidates", t.Candidates, codexTraceJSONArray); err != nil {
		return err
	}
	if err := requireCodexTraceJSON("tools", t.Tools, codexTraceJSONArray); err != nil {
		return err
	}
	if err := requireCodexTraceJSON("signature", t.Signature, codexTraceJSONAny); err != nil {
		return err
	}
	if err := requireCodexTraceJSON("meta", t.Meta, codexTraceJSONObject); err != nil {
		return err
	}
	if err := requireCodexTraceJSON("scaffold", t.Scaffold, codexTraceJSONObject); err != nil {
		return err
	}
	if strings.TrimSpace(t.ScaffoldVersion) == "" {
		return fmt.Errorf("scaffold_version is required")
	}
	return nil
}

// DedupeHash returns a stable digest for the PDF-required duplicate key:
// prompt + candidates + tools. Other metadata may become more complete later
// and must not create a separate trace.
func (t *CodexTraceExport) DedupeHash() string {
	if t == nil {
		return ""
	}

	h := sha256.New()
	hashCodexTraceField(h, "prompt", bytes.TrimSpace(t.Prompt))
	hashCodexTraceField(h, "candidates", bytes.TrimSpace(t.Candidates))
	hashCodexTraceField(h, "tools", bytes.TrimSpace(t.Tools))
	return hex.EncodeToString(h.Sum(nil))
}

// CodexTraceExportDedupeHash validates and hashes a raw trace export payload.
func CodexTraceExportDedupeHash(raw []byte) (string, error) {
	trace, err := ParseCodexTraceExport(raw)
	if err != nil {
		return "", err
	}
	return trace.DedupeHash(), nil
}

type codexTraceJSONKind int

const (
	codexTraceJSONAny codexTraceJSONKind = iota
	codexTraceJSONArray
	codexTraceJSONObject
)

func requireCodexTraceJSON(name string, raw json.RawMessage, kind codexTraceJSONKind) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return fmt.Errorf("%s is required", name)
	}
	if !json.Valid(trimmed) {
		return fmt.Errorf("%s must be valid JSON", name)
	}

	switch kind {
	case codexTraceJSONArray:
		if len(trimmed) == 0 || trimmed[0] != '[' {
			return fmt.Errorf("%s must be a JSON array", name)
		}
	case codexTraceJSONObject:
		if len(trimmed) == 0 || trimmed[0] != '{' {
			return fmt.Errorf("%s must be a JSON object", name)
		}
	}
	return nil
}

func hashCodexTraceField(h hash.Hash, name string, value []byte) {
	var sizeBuf [8]byte
	_, _ = h.Write([]byte(name))
	_, _ = h.Write([]byte{0})
	binary.BigEndian.PutUint64(sizeBuf[:], uint64(len(value)))
	_, _ = h.Write(sizeBuf[:])
	_, _ = h.Write(value)
}
