//go:build integration || contract

package repository

import "testing"

func TestMigrationsRunner_ModelTraceCaptureRulesSchemaStayAligned(t *testing.T) {
	tx := testTx(t)

	requireTableExists(t, tx, "model_trace_capture_rules")
	requireColumn(t, tx, "model_trace_capture_rules", "name", "character varying", 200, false)
	requireColumn(t, tx, "model_trace_capture_rules", "enabled", "boolean", 0, false)
	requireColumn(t, tx, "model_trace_capture_rules", "priority", "integer", 0, false)
	requireColumn(t, tx, "model_trace_capture_rules", "model_patterns", "jsonb", 0, false)
	requireColumn(t, tx, "model_trace_capture_rules", "user_ids", "jsonb", 0, false)
	requireColumn(t, tx, "model_trace_capture_rules", "api_key_ids", "jsonb", 0, false)
	requireColumn(t, tx, "model_trace_capture_rules", "keywords", "jsonb", 0, false)
	requireColumn(t, tx, "model_trace_capture_rules", "min_tokens", "bigint", 0, true)
	requireColumn(t, tx, "model_trace_capture_rules", "max_tokens", "bigint", 0, true)
	requireColumn(t, tx, "model_trace_capture_rules", "sampling_ratio", "double precision", 0, false)
	requireColumn(t, tx, "model_trace_capture_rules", "active_from", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "model_trace_capture_rules", "active_to", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "model_trace_capture_rules", "created_at", "timestamp with time zone", 0, false)
	requireColumn(t, tx, "model_trace_capture_rules", "updated_at", "timestamp with time zone", 0, false)

	requireIndex(t, tx, "model_trace_capture_rules", "idx_model_trace_capture_rules_enabled_priority")
	requireIndex(t, tx, "model_trace_capture_rules", "idx_model_trace_capture_rules_active_window")
	requireIndex(t, tx, "model_trace_capture_rules", "idx_model_trace_capture_rules_updated_at")
}
