package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// AccountExpiryService periodically pauses expired accounts when auto-pause is enabled.
type AccountExpiryService struct {
	accountRepo AccountRepository
	interval    time.Duration
	stopCh      chan struct{}
	stopOnce    sync.Once
	wg          sync.WaitGroup
	locker      periodicTaskLocker
}

func NewAccountExpiryService(accountRepo AccountRepository, interval time.Duration) *AccountExpiryService {
	return &AccountExpiryService{
		accountRepo: accountRepo,
		interval:    interval,
		stopCh:      make(chan struct{}),
		locker:      sharedPeriodicTaskLocker(),
	}
}

func (s *AccountExpiryService) Start() {
	if s == nil || s.accountRepo == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *AccountExpiryService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *AccountExpiryService) runOnce() {
	release, acquired := acquirePeriodicTaskRunLock(
		s.locker,
		"sub2api:periodic:account-expiry",
		periodicTaskLockTTL(s.interval),
		"[AccountExpiry]",
	)
	if !acquired {
		return
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	updated, err := s.accountRepo.AutoPauseExpiredAccounts(ctx, time.Now())
	if err != nil {
		log.Printf("[AccountExpiry] Auto pause expired accounts failed: %v", err)
		return
	}
	if updated > 0 {
		log.Printf("[AccountExpiry] Auto paused %d expired accounts", updated)
	}
}

type periodicTaskLocker interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (release func(), acquired bool, err error)
}

type noopPeriodicTaskLocker struct{}

func (noopPeriodicTaskLocker) TryLock(_ context.Context, _ string, _ time.Duration) (func(), bool, error) {
	return func() {}, true, nil
}

type redisPeriodicTaskLocker struct {
	client     *redis.Client
	instanceID string
}

var periodicTaskUnlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`)

func (l *redisPeriodicTaskLocker) TryLock(ctx context.Context, key string, ttl time.Duration) (func(), bool, error) {
	if l == nil || l.client == nil || key == "" {
		return func() {}, true, nil
	}
	ok, err := l.client.SetNX(ctx, key, l.instanceID, ttl).Result()
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	release := func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = periodicTaskUnlockScript.Run(releaseCtx, l.client, []string{key}, l.instanceID).Result()
	}
	return release, true, nil
}

var (
	sharedPeriodicTaskLockerOnce sync.Once
	sharedPeriodicLocker         periodicTaskLocker = noopPeriodicTaskLocker{}
)

func sharedPeriodicTaskLocker() periodicTaskLocker {
	sharedPeriodicTaskLockerOnce.Do(func() {
		locker := buildPeriodicTaskLockerFromEnv()
		if locker != nil {
			sharedPeriodicLocker = locker
		}
	})
	return sharedPeriodicLocker
}

func buildPeriodicTaskLockerFromEnv() periodicTaskLocker {
	addr := resolvePeriodicTaskRedisAddress()
	if addr == "" {
		return nil
	}

	db := 0
	if rawDB := strings.TrimSpace(os.Getenv("REDIS_DB")); rawDB != "" {
		if parsed, err := strconv.Atoi(rawDB); err == nil {
			db = parsed
		}
	}
	options := &redis.Options{
		Addr:         addr,
		Password:     strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		DB:           db,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     16,
		MinIdleConns: 1,
	}
	if periodicTaskEnvBool("REDIS_ENABLE_TLS") {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[PeriodicTaskLock] redis ping failed, distributed lock disabled: %v", err)
		_ = client.Close()
		return nil
	}

	hostName, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s:%d", hostName, os.Getpid())
	return &redisPeriodicTaskLocker{
		client:     client,
		instanceID: instanceID,
	}
}

func resolvePeriodicTaskRedisAddress() string {
	if addr := strings.TrimSpace(os.Getenv("REDIS_ADDR")); addr != "" {
		return addr
	}
	host := strings.TrimSpace(os.Getenv("REDIS_HOST"))
	port := strings.TrimSpace(os.Getenv("REDIS_PORT"))
	if host == "" {
		return ""
	}
	if port == "" {
		port = "6379"
	}
	return host + ":" + port
}

func periodicTaskEnvBool(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func periodicTaskLockTTL(interval time.Duration) time.Duration {
	if interval <= 0 {
		return 45 * time.Second
	}
	ttl := interval + interval/2
	if ttl < 45*time.Second {
		return 45 * time.Second
	}
	if ttl > 10*time.Minute {
		return 10 * time.Minute
	}
	return ttl
}

func acquirePeriodicTaskRunLock(locker periodicTaskLocker, key string, ttl time.Duration, logPrefix string) (func(), bool) {
	if locker == nil {
		return func() {}, true
	}
	lockCtx, lockCancel := context.WithTimeout(context.Background(), 2*time.Second)
	release, acquired, err := locker.TryLock(lockCtx, key, ttl)
	lockCancel()
	if err != nil {
		log.Printf("%s distributed lock failed, skip this cycle: %v", logPrefix, err)
		return nil, false
	}
	if !acquired {
		return nil, false
	}
	if release != nil {
		return release, true
	}
	return func() {}, true
}
