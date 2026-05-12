package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryDeleteRejectsDirectMutation(t *testing.T) {
	repo := &usageLogRepository{}

	err := repo.Delete(context.Background(), 42)
	require.ErrorIs(t, err, service.ErrUsageLogImmutable)
}
