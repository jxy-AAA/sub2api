package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func newModelCatalogRepoSQLite(t *testing.T) *modelCatalogRepository {
	t.Helper()

	db, err := sql.Open("sqlite", "file:model_catalog_repo?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
CREATE TABLE model_catalog_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	model_id TEXT NOT NULL,
	display_name TEXT NOT NULL,
	provider_key TEXT NOT NULL,
	protocol TEXT NOT NULL,
	capabilities TEXT NOT NULL DEFAULT '[]',
	context_window INTEGER NULL,
	description TEXT NULL,
	tags TEXT NOT NULL DEFAULT '[]',
	status TEXT NOT NULL DEFAULT 'active',
	sort_order INTEGER NOT NULL DEFAULT 0,
	metadata TEXT NOT NULL DEFAULT '{}',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP NULL
);
CREATE UNIQUE INDEX model_catalog_items_provider_model_unique_active
	ON model_catalog_items (provider_key, model_id)
	WHERE deleted_at IS NULL;
`)
	require.NoError(t, err)

	return &modelCatalogRepository{db: db}
}

func TestModelCatalogRepositoryCRUD(t *testing.T) {
	repo := newModelCatalogRepoSQLite(t)
	ctx := context.Background()

	item, err := repo.Create(ctx, &service.ModelCatalogItem{
		ModelID:      "gpt-4o",
		DisplayName:  "GPT-4o",
		ProviderKey:  service.PlatformOpenAI,
		Protocol:     service.PlatformOpenAI,
		Capabilities: []string{"chat"},
		Tags:         []string{"recommended"},
		Status:       service.ModelCatalogStatusActive,
		Metadata:     map[string]any{"source": "test"},
	})
	require.NoError(t, err)
	require.NotZero(t, item.ID)

	items, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "gpt-4o", items[0].ModelID)

	contextWindow := 128000
	item.ContextWindow = &contextWindow
	item.Description = "Updated"
	item.Status = service.ModelCatalogStatusHidden
	updated, err := repo.Update(ctx, item)
	require.NoError(t, err)
	require.NotNil(t, updated.ContextWindow)
	require.Equal(t, 128000, *updated.ContextWindow)
	require.Equal(t, service.ModelCatalogStatusHidden, updated.Status)
	require.Equal(t, "Updated", updated.Description)

	require.NoError(t, repo.Delete(ctx, item.ID))
	items, err = repo.List(ctx)
	require.NoError(t, err)
	require.Empty(t, items)

	got, err := repo.GetByID(ctx, item.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestModelCatalogRepositoryCreateDuplicate(t *testing.T) {
	repo := newModelCatalogRepoSQLite(t)
	ctx := context.Background()

	first := &service.ModelCatalogItem{
		ModelID:      "gpt-4o",
		DisplayName:  "GPT-4o",
		ProviderKey:  service.PlatformOpenAI,
		Protocol:     service.PlatformOpenAI,
		Capabilities: []string{"chat"},
		Tags:         []string{},
		Status:       service.ModelCatalogStatusActive,
		Metadata:     map[string]any{},
	}
	second := &service.ModelCatalogItem{
		ModelID:      "gpt-4o",
		DisplayName:  "GPT-4o duplicate",
		ProviderKey:  service.PlatformOpenAI,
		Protocol:     service.PlatformOpenAI,
		Capabilities: []string{"chat"},
		Tags:         []string{},
		Status:       service.ModelCatalogStatusActive,
		Metadata:     map[string]any{},
	}

	_, err := repo.Create(ctx, first)
	require.NoError(t, err)

	_, err = repo.Create(ctx, second)
	require.ErrorIs(t, err, service.ErrModelCatalogItemExists)
}
