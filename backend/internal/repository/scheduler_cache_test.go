//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSchedulerCacheUnlockBucketWithTokenRejectsStaleOwner(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)
	cache := &schedulerCache{
		rdb:            rdb,
		mgetChunkSize:  defaultSchedulerSnapshotMGetChunkSize,
		writeChunkSize: defaultSchedulerSnapshotWriteChunkSize,
	}
	bucket := service.SchedulerBucket{
		GroupID:  11,
		Platform: service.PlatformOpenAI,
		Mode:     service.SchedulerModeSingle,
	}

	token, locked, err := cache.TryLockBucketWithToken(ctx, bucket, time.Minute)
	require.NoError(t, err)
	require.True(t, locked)
	require.NotEmpty(t, token)

	key := schedulerBucketKey(schedulerLockPrefix, bucket)
	require.NoError(t, rdb.Set(ctx, key, "new-owner", time.Minute).Err())

	require.NoError(t, cache.UnlockBucketWithToken(ctx, bucket, token))

	currentOwner, err := rdb.Get(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, "new-owner", currentOwner)

	require.NoError(t, cache.UnlockBucketWithToken(ctx, bucket, "new-owner"))
	require.ErrorIs(t, rdb.Get(ctx, key).Err(), redis.Nil)
}

func TestSchedulerCacheLegacyUnlockBucketDoesNotDeleteForeignOwner(t *testing.T) {
	ctx := context.Background()
	rdb := testRedis(t)
	cache := &schedulerCache{
		rdb:            rdb,
		mgetChunkSize:  defaultSchedulerSnapshotMGetChunkSize,
		writeChunkSize: defaultSchedulerSnapshotWriteChunkSize,
	}
	bucket := service.SchedulerBucket{
		GroupID:  12,
		Platform: service.PlatformOpenAI,
		Mode:     service.SchedulerModeSingle,
	}

	token, locked, err := cache.TryLockBucketWithToken(ctx, bucket, time.Minute)
	require.NoError(t, err)
	require.True(t, locked)
	require.NotEmpty(t, token)

	key := schedulerBucketKey(schedulerLockPrefix, bucket)
	require.NoError(t, rdb.Set(ctx, key, "new-owner", time.Minute).Err())

	require.NoError(t, cache.UnlockBucket(ctx, bucket))

	currentOwner, err := rdb.Get(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, "new-owner", currentOwner)
}
