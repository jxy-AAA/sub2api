CREATE TABLE IF NOT EXISTS model_catalog_items (
    id BIGSERIAL PRIMARY KEY,
    model_id VARCHAR(200) NOT NULL,
    display_name VARCHAR(200) NOT NULL,
    provider_key VARCHAR(50) NOT NULL,
    protocol VARCHAR(50) NOT NULL,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    context_window INTEGER NULL,
    description TEXT NULL,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    sort_order INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS model_catalog_items_provider_model_unique_active
    ON model_catalog_items (provider_key, model_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_model_catalog_items_model_id
    ON model_catalog_items (model_id);

CREATE INDEX IF NOT EXISTS idx_model_catalog_items_provider_key
    ON model_catalog_items (provider_key);

CREATE INDEX IF NOT EXISTS idx_model_catalog_items_protocol
    ON model_catalog_items (protocol);

CREATE INDEX IF NOT EXISTS idx_model_catalog_items_status
    ON model_catalog_items (status);

CREATE INDEX IF NOT EXISTS idx_model_catalog_items_sort_order
    ON model_catalog_items (sort_order);

CREATE INDEX IF NOT EXISTS idx_model_catalog_items_deleted_at
    ON model_catalog_items (deleted_at);
