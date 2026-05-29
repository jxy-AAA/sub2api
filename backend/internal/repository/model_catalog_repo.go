package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type modelCatalogRepository struct {
	db *sql.DB
}

func NewModelCatalogRepository(db *sql.DB) service.ModelCatalogRepository {
	return &modelCatalogRepository{db: db}
}

func (r *modelCatalogRepository) List(ctx context.Context) ([]*service.ModelCatalogItem, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, model_id, display_name, provider_key, protocol, capabilities, context_window, description, tags, status, sort_order, metadata, created_at, updated_at, deleted_at
FROM model_catalog_items
WHERE deleted_at IS NULL
ORDER BY sort_order ASC, display_name ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query model catalog items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []*service.ModelCatalogItem
	for rows.Next() {
		item, scanErr := scanModelCatalogItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model catalog items: %w", err)
	}
	return items, nil
}

func (r *modelCatalogRepository) GetByID(ctx context.Context, id int64) (*service.ModelCatalogItem, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, model_id, display_name, provider_key, protocol, capabilities, context_window, description, tags, status, sort_order, metadata, created_at, updated_at, deleted_at
FROM model_catalog_items
WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, fmt.Errorf("query model catalog item: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate model catalog item: %w", err)
		}
		return nil, nil
	}

	item, err := scanModelCatalogItem(rows)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *modelCatalogRepository) Create(ctx context.Context, item *service.ModelCatalogItem) (*service.ModelCatalogItem, error) {
	result := &service.ModelCatalogItem{}
	err := r.db.QueryRowContext(ctx, `
INSERT INTO model_catalog_items (
	model_id, display_name, provider_key, protocol, capabilities, context_window, description, tags, status, sort_order, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, model_id, display_name, provider_key, protocol, capabilities, context_window, description, tags, status, sort_order, metadata, created_at, updated_at, deleted_at`,
		item.ModelID,
		item.DisplayName,
		item.ProviderKey,
		item.Protocol,
		mustMarshalJSON(item.Capabilities),
		item.ContextWindow,
		nullIfEmpty(item.Description),
		mustMarshalJSON(item.Tags),
		item.Status,
		item.SortOrder,
		mustMarshalJSON(item.Metadata),
	).Scan(
		&result.ID,
		&result.ModelID,
		&result.DisplayName,
		&result.ProviderKey,
		&result.Protocol,
		newJSONRaw(&result.Capabilities),
		&result.ContextWindow,
		newNullStringTarget(&result.Description),
		newJSONRaw(&result.Tags),
		&result.Status,
		&result.SortOrder,
		newJSONMapRaw(&result.Metadata),
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.DeletedAt,
	)
	if err != nil {
		if isModelCatalogUniqueViolation(err) {
			return nil, service.ErrModelCatalogItemExists
		}
		return nil, fmt.Errorf("insert model catalog item: %w", err)
	}
	normalizeModelCatalogRepositoryItem(result)
	return result, nil
}

func (r *modelCatalogRepository) Update(ctx context.Context, item *service.ModelCatalogItem) (*service.ModelCatalogItem, error) {
	result := &service.ModelCatalogItem{}
	now := time.Now()
	err := r.db.QueryRowContext(ctx, `
UPDATE model_catalog_items
SET model_id = $1,
	display_name = $2,
	provider_key = $3,
	protocol = $4,
	capabilities = $5,
	context_window = $6,
	description = $7,
	tags = $8,
	status = $9,
	sort_order = $10,
	metadata = $11,
	updated_at = $12
WHERE id = $13 AND deleted_at IS NULL
RETURNING id, model_id, display_name, provider_key, protocol, capabilities, context_window, description, tags, status, sort_order, metadata, created_at, updated_at, deleted_at`,
		item.ModelID,
		item.DisplayName,
		item.ProviderKey,
		item.Protocol,
		mustMarshalJSON(item.Capabilities),
		item.ContextWindow,
		nullIfEmpty(item.Description),
		mustMarshalJSON(item.Tags),
		item.Status,
		item.SortOrder,
		mustMarshalJSON(item.Metadata),
		now,
		item.ID,
	).Scan(
		&result.ID,
		&result.ModelID,
		&result.DisplayName,
		&result.ProviderKey,
		&result.Protocol,
		newJSONRaw(&result.Capabilities),
		&result.ContextWindow,
		newNullStringTarget(&result.Description),
		newJSONRaw(&result.Tags),
		&result.Status,
		&result.SortOrder,
		newJSONMapRaw(&result.Metadata),
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, service.ErrModelCatalogItemNotFound
	}
	if err != nil {
		if isModelCatalogUniqueViolation(err) {
			return nil, service.ErrModelCatalogItemExists
		}
		return nil, fmt.Errorf("update model catalog item: %w", err)
	}
	normalizeModelCatalogRepositoryItem(result)
	return result, nil
}

func (r *modelCatalogRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
UPDATE model_catalog_items
SET deleted_at = $1, updated_at = $1
WHERE id = $2 AND deleted_at IS NULL`, now, id)
	if err != nil {
		return fmt.Errorf("soft delete model catalog item: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrModelCatalogItemNotFound
	}
	return nil
}

func isModelCatalogUniqueViolation(err error) bool {
	if isUniqueViolation(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") || strings.Contains(msg, "constraint failed")
}

func scanModelCatalogItem(rows *sql.Rows) (*service.ModelCatalogItem, error) {
	item := &service.ModelCatalogItem{}
	var (
		capabilitiesRaw []byte
		tagsRaw         []byte
		metadataRaw     []byte
		description     sql.NullString
	)
	if err := rows.Scan(
		&item.ID,
		&item.ModelID,
		&item.DisplayName,
		&item.ProviderKey,
		&item.Protocol,
		&capabilitiesRaw,
		&item.ContextWindow,
		&description,
		&tagsRaw,
		&item.Status,
		&item.SortOrder,
		&metadataRaw,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, fmt.Errorf("scan model catalog item: %w", err)
	}
	if description.Valid {
		item.Description = description.String
	}
	if err := unmarshalJSONBytes(capabilitiesRaw, &item.Capabilities); err != nil {
		return nil, fmt.Errorf("decode model catalog capabilities: %w", err)
	}
	if err := unmarshalJSONBytes(tagsRaw, &item.Tags); err != nil {
		return nil, fmt.Errorf("decode model catalog tags: %w", err)
	}
	if err := unmarshalJSONBytes(metadataRaw, &item.Metadata); err != nil {
		return nil, fmt.Errorf("decode model catalog metadata: %w", err)
	}
	normalizeModelCatalogRepositoryItem(item)
	return item, nil
}

func normalizeModelCatalogRepositoryItem(item *service.ModelCatalogItem) {
	if item == nil {
		return
	}
	if item.Capabilities == nil {
		item.Capabilities = []string{}
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
}

func mustMarshalJSON(value any) []byte {
	if value == nil {
		return []byte("{}")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func unmarshalJSONBytes(data []byte, target any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

type jsonRawTarget struct {
	target *[]string
}

func newJSONRaw(target *[]string) *jsonRawTarget {
	return &jsonRawTarget{target: target}
}

func (t *jsonRawTarget) Scan(src any) error {
	if src == nil {
		*t.target = []string{}
		return nil
	}
	switch value := src.(type) {
	case []byte:
		return unmarshalJSONBytes(value, t.target)
	case string:
		return unmarshalJSONBytes([]byte(value), t.target)
	default:
		return fmt.Errorf("unsupported JSON scan type %T", src)
	}
}

type jsonMapRawTarget struct {
	target *map[string]any
}

func newJSONMapRaw(target *map[string]any) *jsonMapRawTarget {
	return &jsonMapRawTarget{target: target}
}

func (t *jsonMapRawTarget) Scan(src any) error {
	if src == nil {
		*t.target = map[string]any{}
		return nil
	}
	switch value := src.(type) {
	case []byte:
		return unmarshalJSONBytes(value, t.target)
	case string:
		return unmarshalJSONBytes([]byte(value), t.target)
	default:
		return fmt.Errorf("unsupported JSON scan type %T", src)
	}
}

type nullStringTarget struct {
	target *string
}

func newNullStringTarget(target *string) *nullStringTarget {
	return &nullStringTarget{target: target}
}

func (t *nullStringTarget) Scan(src any) error {
	if src == nil {
		*t.target = ""
		return nil
	}
	switch value := src.(type) {
	case string:
		*t.target = value
		return nil
	case []byte:
		*t.target = string(value)
		return nil
	default:
		return fmt.Errorf("unsupported string scan type %T", src)
	}
}
