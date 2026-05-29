//go:build integration || contract

package repository

import (
	"context"
	"database/sql"
	"io/fs"
	"sort"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate(t *testing.T) {
	tx := testTx(t)

	// Re-apply migrations to verify idempotency (no errors, no duplicate rows).
	require.NoError(t, ApplyMigrations(context.Background(), integrationDB))

	// schema_migrations must track the exact embedded migration set.
	expectedMigrations := embeddedMigrationFilenames(t)
	require.Equal(t, expectedMigrations, appliedMigrationFilenames(t, tx), "schema_migrations must match embedded SQL migrations exactly")
	requireMigrationApplied(t, tx, expectedMigrations[len(expectedMigrations)-1])

	// users: columns required by repository queries
	requireColumn(t, tx, "users", "username", "character varying", 100, false)
	requireColumn(t, tx, "users", "notes", "text", 0, false)

	// accounts: schedulable and rate-limit fields
	requireColumn(t, tx, "accounts", "notes", "text", 0, true)
	requireColumn(t, tx, "accounts", "schedulable", "boolean", 0, false)
	requireColumn(t, tx, "accounts", "rate_limited_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "rate_limit_reset_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "overload_until", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "session_window_status", "character varying", 20, true)

	// api_keys: key length should be 128
	requireColumn(t, tx, "api_keys", "key", "character varying", 128, false)

	// redeem_codes: subscription fields
	requireColumn(t, tx, "redeem_codes", "group_id", "bigint", 0, true)
	requireColumn(t, tx, "redeem_codes", "validity_days", "integer", 0, false)

	// usage_logs: billing_type used by filters/stats
	requireColumn(t, tx, "usage_logs", "billing_type", "smallint", 0, false)
	requireColumn(t, tx, "usage_logs", "request_type", "smallint", 0, false)
	requireColumn(t, tx, "usage_logs", "openai_ws_mode", "boolean", 0, false)

	// usage_billing_dedup: billing idempotency narrow table
	var usageBillingDedupRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.usage_billing_dedup')").Scan(&usageBillingDedupRegclass))
	require.True(t, usageBillingDedupRegclass.Valid, "expected usage_billing_dedup table to exist")
	requireColumn(t, tx, "usage_billing_dedup", "request_fingerprint", "character varying", 64, false)
	requireIndex(t, tx, "usage_billing_dedup", "idx_usage_billing_dedup_request_api_key")
	requireIndex(t, tx, "usage_billing_dedup", "idx_usage_billing_dedup_created_at_brin")

	var usageBillingDedupArchiveRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.usage_billing_dedup_archive')").Scan(&usageBillingDedupArchiveRegclass))
	require.True(t, usageBillingDedupArchiveRegclass.Valid, "expected usage_billing_dedup_archive table to exist")
	requireColumn(t, tx, "usage_billing_dedup_archive", "request_fingerprint", "character varying", 64, false)
	requireIndex(t, tx, "usage_billing_dedup_archive", "usage_billing_dedup_archive_pkey")

	// settings table should exist
	var settingsRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.settings')").Scan(&settingsRegclass))
	require.True(t, settingsRegclass.Valid, "expected settings table to exist")

	// security_secrets table should exist
	var securitySecretsRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.security_secrets')").Scan(&securitySecretsRegclass))
	require.True(t, securitySecretsRegclass.Valid, "expected security_secrets table to exist")

	// user_allowed_groups table should exist
	var uagRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.user_allowed_groups')").Scan(&uagRegclass))
	require.True(t, uagRegclass.Valid, "expected user_allowed_groups table to exist")

	// user_subscriptions: deleted_at for soft delete support (migration 012)
	requireColumn(t, tx, "user_subscriptions", "deleted_at", "timestamp with time zone", 0, true)

	// orphan_allowed_groups_audit table should exist (migration 013)
	var orphanAuditRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.orphan_allowed_groups_audit')").Scan(&orphanAuditRegclass))
	require.True(t, orphanAuditRegclass.Valid, "expected orphan_allowed_groups_audit table to exist")

	// account_groups: created_at should be timestamptz
	requireColumn(t, tx, "account_groups", "created_at", "timestamp with time zone", 0, false)

	// user_allowed_groups: created_at should be timestamptz
	requireColumn(t, tx, "user_allowed_groups", "created_at", "timestamp with time zone", 0, false)

	// model_catalog_items: model market persistence
	requireTableExists(t, tx, "model_catalog_items")
	requireColumn(t, tx, "model_catalog_items", "model_id", "character varying", 200, false)
	requireColumn(t, tx, "model_catalog_items", "display_name", "character varying", 200, false)
	requireColumn(t, tx, "model_catalog_items", "provider_key", "character varying", 50, false)
	requireColumn(t, tx, "model_catalog_items", "protocol", "character varying", 50, false)
	requireColumn(t, tx, "model_catalog_items", "context_window", "integer", 0, true)
	requireColumn(t, tx, "model_catalog_items", "description", "text", 0, true)
	requireColumn(t, tx, "model_catalog_items", "status", "character varying", 20, false)
	requireColumn(t, tx, "model_catalog_items", "deleted_at", "timestamp with time zone", 0, true)
	requireIndex(t, tx, "model_catalog_items", "idx_model_catalog_items_model_id")
	requireIndex(t, tx, "model_catalog_items", "idx_model_catalog_items_provider_key")
	requireIndex(t, tx, "model_catalog_items", "idx_model_catalog_items_protocol")
	requireIndex(t, tx, "model_catalog_items", "idx_model_catalog_items_status")
	requireIndex(t, tx, "model_catalog_items", "idx_model_catalog_items_sort_order")
	requireIndex(t, tx, "model_catalog_items", "idx_model_catalog_items_deleted_at")
	requirePartialUniqueIndexDefinition(
		t,
		tx,
		"model_catalog_items",
		"model_catalog_items_provider_model_unique_active",
		"provider_key",
		"model_id",
		"WHERE",
	)

	// model_interaction_traces: raw model trace export storage
	requireTableExists(t, tx, "model_interaction_traces")
	requireColumn(t, tx, "model_interaction_traces", "task_id", "text", 0, false)
	requireColumn(t, tx, "model_interaction_traces", "prompt", "json", 0, false)
	requireColumn(t, tx, "model_interaction_traces", "candidates", "json", 0, false)
	requireColumn(t, tx, "model_interaction_traces", "tools", "json", 0, false)
	requireColumn(t, tx, "model_interaction_traces", "signature", "json", 0, false)
	requireColumn(t, tx, "model_interaction_traces", "meta", "json", 0, false)
	requireColumn(t, tx, "model_interaction_traces", "scaffold", "json", 0, false)
	requireColumn(t, tx, "model_interaction_traces", "scaffold_version", "text", 0, false)
	requireColumn(t, tx, "model_interaction_traces", "model", "text", 0, true)
	requireColumn(t, tx, "model_interaction_traces", "user_id", "bigint", 0, true)
	requireColumn(t, tx, "model_interaction_traces", "api_key_id", "bigint", 0, true)
	requireColumn(t, tx, "model_interaction_traces", "request_id", "text", 0, true)
	requireColumn(t, tx, "model_interaction_traces", "dedupe_hash", "text", 0, false)
	requireColumn(t, tx, "model_interaction_traces", "created_at", "timestamp with time zone", 0, false)
	requireIndex(t, tx, "model_interaction_traces", "idx_model_interaction_traces_task_id")
	requireIndex(t, tx, "model_interaction_traces", "model_interaction_traces_dedupe_hash_unique")
	requireIndex(t, tx, "model_interaction_traces", "idx_model_interaction_traces_created_at_id")
	requireIndex(t, tx, "model_interaction_traces", "idx_model_interaction_traces_request_id")
	requireIndex(t, tx, "model_interaction_traces", "idx_model_interaction_traces_model")
	requireIndex(t, tx, "model_interaction_traces", "idx_model_interaction_traces_user_id")
	requireIndex(t, tx, "model_interaction_traces", "idx_model_interaction_traces_api_key_id")
}

func TestMigrationsRunner_AuthIdentityAndPaymentSchemaStayAligned(t *testing.T) {
	tx := testTx(t)

	requireColumn(t, tx, "auth_identity_migration_reports", "report_type", "character varying", 80, false)
	requireColumn(t, tx, "users", "signup_source", "character varying", 20, false)
	requireColumnDefaultContains(t, tx, "users", "signup_source", "email")
	requireConstraintDefinitionContains(
		t,
		tx,
		"users",
		"users_signup_source_check",
		"signup_source",
		"'email'",
		"'linuxdo'",
		"'wechat'",
		"'oidc'",
	)

	requireForeignKeyOnDelete(t, tx, "auth_identities", "user_id", "users", "CASCADE")
	requireForeignKeyOnDelete(t, tx, "auth_identity_channels", "identity_id", "auth_identities", "CASCADE")
	requireForeignKeyOnDelete(t, tx, "pending_auth_sessions", "target_user_id", "users", "SET NULL")
	requireForeignKeyOnDelete(t, tx, "identity_adoption_decisions", "pending_auth_session_id", "pending_auth_sessions", "CASCADE")
	requireForeignKeyOnDelete(t, tx, "identity_adoption_decisions", "identity_id", "auth_identities", "SET NULL")

	requireColumn(t, tx, "payment_orders", "refund_gateway_confirmed_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "payment_orders", "refund_gateway_refund_id", "character varying", 128, true)
	requireColumn(t, tx, "payment_orders", "refund_idempotency_key", "character varying", 128, true)
	requireIndex(t, tx, "payment_orders", "paymentorder_out_trade_no")
	requireIndex(t, tx, "payment_orders", "idx_payment_orders_refund_gateway_confirmed_at")
	requirePartialUniqueIndexDefinition(t, tx, "payment_orders", "paymentorder_out_trade_no", "out_trade_no", "WHERE")
	requireIndexAbsent(t, tx, "payment_orders", "paymentorder_out_trade_no_unique")
}

func TestMigrationsRunner_UsageCleanupAndAffiliateProtectionSchemaStayAligned(t *testing.T) {
	tx := testTx(t)

	requireTableExists(t, tx, "usage_cleanup_tasks")
	requireForeignKeyOnDelete(t, tx, "usage_cleanup_tasks", "created_by", "users", "RESTRICT")
	requireIndex(t, tx, "usage_cleanup_tasks", "idx_usage_cleanup_tasks_status_created_at")
	requireIndex(t, tx, "usage_cleanup_tasks", "idx_usage_cleanup_tasks_created_at")

	requireTableExists(t, tx, "affiliate_distribution_paid_credit_balances")
	requireForeignKeyOnDelete(t, tx, "affiliate_distribution_paid_credit_balances", "user_id", "users", "CASCADE")

	requireTableExists(t, tx, "affiliate_distribution_paid_credit_events")
	requireForeignKeyOnDelete(t, tx, "affiliate_distribution_paid_credit_events", "user_id", "users", "CASCADE")
	requireIndex(t, tx, "affiliate_distribution_paid_credit_events", "idx_aff_dist_paid_credit_events_user_time")

	requireTableExists(t, tx, "affiliate_distribution_paid_credit_reversals")
	requireForeignKeyOnDelete(t, tx, "affiliate_distribution_paid_credit_reversals", "credit_event_id", "affiliate_distribution_paid_credit_events", "RESTRICT")
	requireForeignKeyOnDelete(t, tx, "affiliate_distribution_paid_credit_reversals", "user_id", "users", "CASCADE")
	requireIndex(t, tx, "affiliate_distribution_paid_credit_reversals", "idx_aff_dist_paid_credit_reversals_event")
	requireIndex(t, tx, "affiliate_distribution_paid_credit_reversals", "idx_aff_dist_paid_credit_reversals_order")

	requireTableExists(t, tx, "affiliate_distribution_paid_usage_settlements")
	requireForeignKeyOnDelete(t, tx, "affiliate_distribution_paid_usage_settlements", "usage_log_id", "usage_logs", "RESTRICT")
	requireForeignKeyOnDelete(t, tx, "affiliate_distribution_paid_usage_settlements", "user_id", "users", "CASCADE")
	requireIndex(t, tx, "affiliate_distribution_paid_usage_settlements", "idx_aff_dist_paid_usage_user_day")

	requireTableExists(t, tx, "affiliate_distribution_paid_credit_allocations")
	requireForeignKeyOnDelete(t, tx, "affiliate_distribution_paid_credit_allocations", "credit_event_id", "affiliate_distribution_paid_credit_events", "RESTRICT")
	requireForeignKeyOnDelete(t, tx, "affiliate_distribution_paid_credit_allocations", "usage_log_id", "affiliate_distribution_paid_usage_settlements", "CASCADE")
	requireForeignKeyOnDelete(t, tx, "affiliate_distribution_paid_credit_allocations", "user_id", "users", "CASCADE")
	requireIndex(t, tx, "affiliate_distribution_paid_credit_allocations", "idx_aff_dist_paid_credit_allocations_usage")
	requireIndex(t, tx, "affiliate_distribution_paid_credit_allocations", "idx_aff_dist_paid_credit_allocations_user")
}

func embeddedMigrationFilenames(t *testing.T) []string {
	t.Helper()

	filenames, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err, "list embedded migrations")
	sort.Strings(filenames)
	require.NotEmpty(t, filenames, "expected embedded SQL migrations")
	return filenames
}

func appliedMigrationFilenames(t *testing.T, tx *sql.Tx) []string {
	t.Helper()

	rows, err := tx.QueryContext(context.Background(), "SELECT filename FROM schema_migrations ORDER BY filename")
	require.NoError(t, err, "query schema_migrations filenames")
	defer func() { _ = rows.Close() }()

	var filenames []string
	for rows.Next() {
		var filename string
		require.NoError(t, rows.Scan(&filename), "scan schema_migrations filename")
		filenames = append(filenames, filename)
	}
	require.NoError(t, rows.Err(), "iterate schema_migrations filenames")
	return filenames
}

func requireMigrationApplied(t *testing.T, tx *sql.Tx, filename string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM schema_migrations
	WHERE filename = $1
)
`, filename).Scan(&exists)
	require.NoError(t, err, "query schema_migrations for %s", filename)
	require.True(t, exists, "expected migration %s to be recorded", filename)
}

func requireTableExists(t *testing.T, tx *sql.Tx, table string) {
	t.Helper()

	var regclass sql.NullString
	err := tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.' || $1)", table).Scan(&regclass)
	require.NoError(t, err, "query to_regclass for %s", table)
	require.True(t, regclass.Valid, "expected table %s to exist", table)
}

func requireIndex(t *testing.T, tx *sql.Tx, table, index string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM pg_indexes
	WHERE schemaname = 'public'
	  AND tablename = $1
	  AND indexname = $2
)
`, table, index).Scan(&exists)
	require.NoError(t, err, "query pg_indexes for %s.%s", table, index)
	require.True(t, exists, "expected index %s on %s", index, table)
}

func requireIndexAbsent(t *testing.T, tx *sql.Tx, table, index string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM pg_indexes
	WHERE schemaname = 'public'
	  AND tablename = $1
	  AND indexname = $2
)
`, table, index).Scan(&exists)
	require.NoError(t, err, "query pg_indexes for %s.%s", table, index)
	require.False(t, exists, "expected index %s on %s to be absent", index, table)
}

func requirePartialUniqueIndexDefinition(t *testing.T, tx *sql.Tx, table, index string, fragments ...string) {
	t.Helper()

	var (
		unique bool
		def    string
	)

	err := tx.QueryRowContext(context.Background(), `
SELECT
	i.indisunique,
	pg_get_indexdef(i.indexrelid)
FROM pg_class idx
JOIN pg_index i ON i.indexrelid = idx.oid
JOIN pg_class tbl ON tbl.oid = i.indrelid
JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
WHERE ns.nspname = 'public'
  AND tbl.relname = $1
  AND idx.relname = $2
`, table, index).Scan(&unique, &def)
	require.NoError(t, err, "query index definition for %s.%s", table, index)
	require.True(t, unique, "expected index %s on %s to be unique", index, table)

	for _, fragment := range fragments {
		require.Contains(t, def, fragment, "expected index definition for %s.%s to contain %q", table, index, fragment)
	}
}

func requireForeignKeyOnDelete(t *testing.T, tx *sql.Tx, table, column, refTable, expected string) {
	t.Helper()

	var actual string
	err := tx.QueryRowContext(context.Background(), `
SELECT CASE c.confdeltype
	WHEN 'a' THEN 'NO ACTION'
	WHEN 'r' THEN 'RESTRICT'
	WHEN 'c' THEN 'CASCADE'
	WHEN 'n' THEN 'SET NULL'
	WHEN 'd' THEN 'SET DEFAULT'
END
FROM pg_constraint c
JOIN pg_class tbl ON tbl.oid = c.conrelid
JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
JOIN pg_class ref_tbl ON ref_tbl.oid = c.confrelid
JOIN pg_attribute attr ON attr.attrelid = tbl.oid AND attr.attnum = ANY(c.conkey)
WHERE ns.nspname = 'public'
  AND c.contype = 'f'
  AND tbl.relname = $1
  AND attr.attname = $2
  AND ref_tbl.relname = $3
LIMIT 1
`, table, column, refTable).Scan(&actual)
	require.NoError(t, err, "query foreign key action for %s.%s -> %s", table, column, refTable)
	require.Equal(t, expected, actual, "unexpected ON DELETE action for %s.%s -> %s", table, column, refTable)
}

func requireConstraintDefinitionContains(t *testing.T, tx *sql.Tx, table, constraint string, fragments ...string) {
	t.Helper()

	var def string
	err := tx.QueryRowContext(context.Background(), `
SELECT pg_get_constraintdef(c.oid)
FROM pg_constraint c
JOIN pg_class tbl ON tbl.oid = c.conrelid
JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
WHERE ns.nspname = 'public'
  AND tbl.relname = $1
  AND c.conname = $2
`, table, constraint).Scan(&def)
	require.NoError(t, err, "query constraint definition for %s.%s", table, constraint)

	for _, fragment := range fragments {
		require.Contains(t, def, fragment, "expected constraint definition for %s.%s to contain %q", table, constraint, fragment)
	}
}

func requireColumnDefaultContains(t *testing.T, tx *sql.Tx, table, column string, fragments ...string) {
	t.Helper()

	var columnDefault sql.NullString
	err := tx.QueryRowContext(context.Background(), `
SELECT column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = $1
  AND column_name = $2
`, table, column).Scan(&columnDefault)
	require.NoError(t, err, "query column_default for %s.%s", table, column)
	require.True(t, columnDefault.Valid, "expected column_default for %s.%s", table, column)

	for _, fragment := range fragments {
		require.Contains(t, columnDefault.String, fragment, "expected default for %s.%s to contain %q", table, column, fragment)
	}
}

func requireColumn(t *testing.T, tx *sql.Tx, table, column, dataType string, maxLen int, nullable bool) {
	t.Helper()

	var row struct {
		DataType string
		MaxLen   sql.NullInt64
		Nullable string
	}

	err := tx.QueryRowContext(context.Background(), `
SELECT
  data_type,
  character_maximum_length,
  is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = $1
  AND column_name = $2
`, table, column).Scan(&row.DataType, &row.MaxLen, &row.Nullable)
	require.NoError(t, err, "query information_schema.columns for %s.%s", table, column)
	require.Equal(t, dataType, row.DataType, "data_type mismatch for %s.%s", table, column)

	if maxLen > 0 {
		require.True(t, row.MaxLen.Valid, "expected maxLen for %s.%s", table, column)
		require.Equal(t, int64(maxLen), row.MaxLen.Int64, "maxLen mismatch for %s.%s", table, column)
	}

	if nullable {
		require.Equal(t, "YES", row.Nullable, "nullable mismatch for %s.%s", table, column)
	} else {
		require.Equal(t, "NO", row.Nullable, "nullable mismatch for %s.%s", table, column)
	}
}
