package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type tokenAwareSchedulerCacheStub struct {
	SchedulerCache
	token           string
	tryTokenCalls   int
	unlockedTokens  []string
	snapshotBuckets []SchedulerBucket
	snapshotSizes   []int
}

func (c *tokenAwareSchedulerCacheStub) TryLockBucketWithToken(ctx context.Context, bucket SchedulerBucket, ttl time.Duration) (string, bool, error) {
	c.tryTokenCalls++
	return c.token, true, nil
}

func (c *tokenAwareSchedulerCacheStub) UnlockBucketWithToken(ctx context.Context, bucket SchedulerBucket, token string) error {
	c.unlockedTokens = append(c.unlockedTokens, token)
	return nil
}

func (c *tokenAwareSchedulerCacheStub) SetSnapshot(ctx context.Context, bucket SchedulerBucket, accounts []Account) error {
	c.snapshotBuckets = append(c.snapshotBuckets, bucket)
	c.snapshotSizes = append(c.snapshotSizes, len(accounts))
	return nil
}

type schedulerSnapshotAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (r *schedulerSnapshotAccountRepoStub) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	out := make([]Account, len(r.accounts))
	copy(out, r.accounts)
	return out, nil
}

func TestSchedulerSnapshotServiceRebuildBucketPrefersTokenAwareLocking(t *testing.T) {
	cache := &tokenAwareSchedulerCacheStub{token: "owner-token"}
	repo := &schedulerSnapshotAccountRepoStub{
		accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformGemini,
				Status:      StatusActive,
				Schedulable: true,
			},
		},
	}
	svc := &SchedulerSnapshotService{
		cache:       cache,
		accountRepo: repo,
	}

	bucket := SchedulerBucket{GroupID: 7, Platform: PlatformGemini, Mode: SchedulerModeSingle}
	err := svc.rebuildBucket(context.Background(), bucket, "test")
	require.NoError(t, err)
	require.Equal(t, 1, cache.tryTokenCalls)
	require.Equal(t, []string{"owner-token"}, cache.unlockedTokens)
	require.Equal(t, []SchedulerBucket{bucket}, cache.snapshotBuckets)
	require.Equal(t, []int{1}, cache.snapshotSizes)
}
