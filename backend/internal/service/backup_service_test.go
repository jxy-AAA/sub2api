//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// ─── Mocks ───

type mockSettingRepo struct {
	mu           sync.Mutex
	data         map[string]string
	getValueErrs map[string]error
}

func newMockSettingRepo() *mockSettingRepo {
	return &mockSettingRepo{
		data:         make(map[string]string),
		getValueErrs: make(map[string]error),
	}
}

func (m *mockSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: v}, nil
}

func (m *mockSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err, ok := m.getValueErrs[key]; ok {
		return "", err
	}
	v, ok := m.data[key]
	if !ok {
		return "", nil
	}
	return v, nil
}

func (m *mockSettingRepo) Set(_ context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *mockSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]string)
	for _, k := range keys {
		if v, ok := m.data[k]; ok {
			result[k] = v
		}
	}
	return result, nil
}

func (m *mockSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range settings {
		m.data[k] = v
	}
	return nil
}

func (m *mockSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]string, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result, nil
}

func (m *mockSettingRepo) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

// plainEncryptor 仅做 base64-like 包装，用于测试
type plainEncryptor struct{}

func (e *plainEncryptor) Encrypt(plaintext string) (string, error) {
	return "ENC:" + plaintext, nil
}

func (e *plainEncryptor) Decrypt(ciphertext string) (string, error) {
	if strings.HasPrefix(ciphertext, "ENC:") {
		return strings.TrimPrefix(ciphertext, "ENC:"), nil
	}
	return ciphertext, fmt.Errorf("not encrypted")
}

type mockDumper struct {
	dumpData []byte
	dumpErr  error
	restored []byte
	restErr  error
}

func (m *mockDumper) Dump(_ context.Context) (io.ReadCloser, error) {
	if m.dumpErr != nil {
		return nil, m.dumpErr
	}
	return io.NopCloser(bytes.NewReader(m.dumpData)), nil
}

func (m *mockDumper) Restore(_ context.Context, data io.Reader) error {
	if m.restErr != nil {
		return m.restErr
	}
	d, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	m.restored = d
	return nil
}

// blockingDumper 可控延迟的 dumper，用于测试异步行为
type blockingDumper struct {
	blockCh   chan struct{}
	startedCh chan struct{}
	data      []byte
	restErr   error
}

func (d *blockingDumper) Dump(ctx context.Context) (io.ReadCloser, error) {
	if d.startedCh != nil {
		select {
		case d.startedCh <- struct{}{}:
		default:
		}
	}
	select {
	case <-d.blockCh:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return io.NopCloser(bytes.NewReader(d.data)), nil
}

func (d *blockingDumper) Restore(_ context.Context, data io.Reader) error {
	if d.restErr != nil {
		return d.restErr
	}
	_, _ = io.ReadAll(data)
	return nil
}

type mockObjectStore struct {
	objects map[string][]byte
	mu      sync.Mutex
}

func newMockObjectStore() *mockObjectStore {
	return &mockObjectStore{objects: make(map[string][]byte)}
}

func (m *mockObjectStore) Upload(_ context.Context, key string, body io.Reader, _ string) (int64, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return 0, err
	}
	m.mu.Lock()
	m.objects[key] = data
	m.mu.Unlock()
	return int64(len(data)), nil
}

func (m *mockObjectStore) Download(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	data, ok := m.objects[key]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("not found: %s", key)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockObjectStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.objects, key)
	m.mu.Unlock()
	return nil
}

func (m *mockObjectStore) PresignURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://presigned.example.com/" + key, nil
}

func (m *mockObjectStore) HeadBucket(_ context.Context) error {
	return nil
}

func newTestBackupService(repo *mockSettingRepo, dumper DBDumper, store *mockObjectStore) *BackupService {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:   "localhost",
			Port:   5432,
			User:   "test",
			DBName: "testdb",
		},
	}
	factory := func(_ context.Context, _ *BackupS3Config) (BackupObjectStore, error) {
		return store, nil
	}
	return NewBackupService(repo, cfg, &plainEncryptor{}, factory, dumper)
}

type sharedBackupLockStore struct {
	mu    sync.Mutex
	state *backupOperationLockState
}

func (s *sharedBackupLockStore) AcquireBackupOperationLock(ctx context.Context, owner string, operation string, lease time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if s.state != nil && !backupOperationLockExpired(*s.state, now) && s.state.Owner != owner {
		return false, nil
	}
	s.state = &backupOperationLockState{
		Owner:      owner,
		Operation:  operation,
		ExpiresAt:  now.Add(lease).Format(time.RFC3339Nano),
		AcquiredAt: now.Format(time.RFC3339Nano),
	}
	return true, nil
}

func (s *sharedBackupLockStore) ReleaseBackupOperationLock(ctx context.Context, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != nil && s.state.Owner == owner {
		s.state = nil
	}
	return nil
}

func seedS3Config(t *testing.T, repo *mockSettingRepo) {
	t.Helper()
	cfg := BackupS3Config{
		Bucket:          "test-bucket",
		AccessKeyID:     "AKID",
		SecretAccessKey: "ENC:secret123",
		Prefix:          "backups",
	}
	data, _ := json.Marshal(cfg)
	require.NoError(t, repo.Set(context.Background(), settingKeyBackupS3Config, string(data)))
}

// ─── Tests ───

func TestBackupService_S3ConfigEncryption(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	// 保存配置 -> SecretAccessKey 应被加密
	_, err := svc.UpdateS3Config(context.Background(), BackupS3Config{
		Bucket:          "my-bucket",
		AccessKeyID:     "AKID",
		SecretAccessKey: "my-secret",
		Prefix:          "backups",
	})
	require.NoError(t, err)

	// 直接读取数据库中存储的值，应该是加密后的
	raw, _ := repo.GetValue(context.Background(), settingKeyBackupS3Config)
	var stored BackupS3Config
	require.NoError(t, json.Unmarshal([]byte(raw), &stored))
	require.Equal(t, "ENC:my-secret", stored.SecretAccessKey)

	// 通过 GetS3Config 获取应该脱敏
	cfg, err := svc.GetS3Config(context.Background())
	require.NoError(t, err)
	require.Empty(t, cfg.SecretAccessKey)
	require.Equal(t, "my-bucket", cfg.Bucket)

	// loadS3Config 内部应解密
	internal, err := svc.loadS3Config(context.Background())
	require.NoError(t, err)
	require.Equal(t, "my-secret", internal.SecretAccessKey)
}

func TestBackupService_S3ConfigKeepExistingSecret(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	// 先保存一个有 secret 的配置
	_, err := svc.UpdateS3Config(context.Background(), BackupS3Config{
		Bucket:          "my-bucket",
		AccessKeyID:     "AKID",
		SecretAccessKey: "original-secret",
	})
	require.NoError(t, err)

	// 再更新时不提供 secret，应保留原值
	_, err = svc.UpdateS3Config(context.Background(), BackupS3Config{
		Bucket:      "my-bucket",
		AccessKeyID: "AKID-NEW",
	})
	require.NoError(t, err)

	internal, err := svc.loadS3Config(context.Background())
	require.NoError(t, err)
	require.Equal(t, "original-secret", internal.SecretAccessKey)
	require.Equal(t, "AKID-NEW", internal.AccessKeyID)
}

func TestBackupService_SaveRecordConcurrency(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	var wg sync.WaitGroup
	n := 20
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			record := &BackupRecord{
				ID:        fmt.Sprintf("rec-%d", idx),
				Status:    "completed",
				StartedAt: time.Now().Format(time.RFC3339),
			}
			_ = svc.saveRecord(context.Background(), record)
		}(i)
	}
	wg.Wait()

	records, err := svc.loadRecords(context.Background())
	require.NoError(t, err)
	require.Len(t, records, n)
}

func TestBackupService_LoadRecords_Empty(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	records, err := svc.loadRecords(context.Background())
	require.NoError(t, err)
	require.Nil(t, records) // 无数据时返回 nil
}

func TestBackupService_LoadRecords_Corrupted(t *testing.T) {
	repo := newMockSettingRepo()
	_ = repo.Set(context.Background(), settingKeyBackupRecords, "not valid json{{{")
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	records, err := svc.loadRecords(context.Background())
	require.Error(t, err)
	require.Nil(t, records)
}

func TestBackupService_SaveRecord_DoesNotOverwriteOnReadError(t *testing.T) {
	repo := newMockSettingRepo()
	existingRaw := `[{"id":"existing","status":"completed","started_at":"2026-01-02T03:04:05Z"}]`
	require.NoError(t, repo.Set(context.Background(), settingKeyBackupRecords, existingRaw))
	repo.getValueErrs[settingKeyBackupRecords] = fmt.Errorf("settings store unavailable")
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	err := svc.saveRecord(context.Background(), &BackupRecord{
		ID:        "new-record",
		Status:    "completed",
		StartedAt: time.Now().Format(time.RFC3339),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "load backup records")
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Equal(t, existingRaw, repo.data[settingKeyBackupRecords])
}

func TestBackupService_SaveRecord_DoesNotOverwriteCorruptRecords(t *testing.T) {
	repo := newMockSettingRepo()
	require.NoError(t, repo.Set(context.Background(), settingKeyBackupRecords, "not valid json{{{"))
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	err := svc.saveRecord(context.Background(), &BackupRecord{
		ID:        "new-record",
		Status:    "completed",
		StartedAt: time.Now().Format(time.RFC3339),
	})
	require.ErrorIs(t, err, ErrBackupRecordsCorrupt)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Equal(t, "not valid json{{{", repo.data[settingKeyBackupRecords])
}

func TestBackupService_CreateBackup_Streaming(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	dumpContent := "-- PostgreSQL dump\nCREATE TABLE test (id int);\n"
	dumper := &mockDumper{dumpData: []byte(dumpContent)}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	record, err := svc.CreateBackup(context.Background(), "manual", 14)
	require.NoError(t, err)
	require.Equal(t, "completed", record.Status)
	require.Greater(t, record.SizeBytes, int64(0))
	require.NotEmpty(t, record.S3Key)

	// 验证 S3 上确实有文件
	store.mu.Lock()
	require.Len(t, store.objects, 1)
	store.mu.Unlock()
}

func TestBackupService_CreateBackup_DumpFailure(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	dumper := &mockDumper{dumpErr: fmt.Errorf("pg_dump failed")}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	record, err := svc.CreateBackup(context.Background(), "manual", 14)
	require.Error(t, err)
	require.Equal(t, "failed", record.Status)
	require.Contains(t, record.ErrorMsg, "pg_dump")
}

func TestBackupService_CreateBackup_NoS3Config(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	_, err := svc.CreateBackup(context.Background(), "manual", 14)
	require.ErrorIs(t, err, ErrBackupS3NotConfigured)
}

func TestBackupService_CreateBackup_ConcurrentBlocked(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	// 使用一个慢速 dumper 来模拟正在进行的备份
	dumper := &mockDumper{dumpData: []byte("data")}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	// 手动设置 backingUp 标志
	svc.opMu.Lock()
	svc.backingUp = true
	svc.opMu.Unlock()

	_, err := svc.CreateBackup(context.Background(), "manual", 14)
	require.ErrorIs(t, err, ErrBackupInProgress)
}

func TestBackupService_RestoreBackup_Streaming(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	dumpContent := "-- PostgreSQL dump\nCREATE TABLE test (id int);\n"
	dumper := &mockDumper{dumpData: []byte(dumpContent)}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	// 先创建一个备份
	record, err := svc.CreateBackup(context.Background(), "manual", 14)
	require.NoError(t, err)

	// 恢复
	err = svc.RestoreBackup(context.Background(), record.ID)
	require.NoError(t, err)

	// 验证 psql 收到的数据是否与原始 dump 内容一致
	require.Equal(t, dumpContent, string(dumper.restored))
}

func TestBackupService_RestoreBackup_NotCompleted(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	// 手动插入一条 failed 记录
	_ = svc.saveRecord(context.Background(), &BackupRecord{
		ID:     "fail-1",
		Status: "failed",
	})

	err := svc.RestoreBackup(context.Background(), "fail-1")
	require.Error(t, err)
}

func TestBackupService_DeleteBackup(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	dumpContent := "data"
	dumper := &mockDumper{dumpData: []byte(dumpContent)}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	record, err := svc.CreateBackup(context.Background(), "manual", 14)
	require.NoError(t, err)

	// S3 中应有文件
	store.mu.Lock()
	require.Len(t, store.objects, 1)
	store.mu.Unlock()

	// 删除
	err = svc.DeleteBackup(context.Background(), record.ID)
	require.NoError(t, err)

	// S3 中文件应被删除
	store.mu.Lock()
	require.Len(t, store.objects, 0)
	store.mu.Unlock()

	// 记录应不存在
	_, err = svc.GetBackupRecord(context.Background(), record.ID)
	require.ErrorIs(t, err, ErrBackupNotFound)
}

func TestBackupService_GetDownloadURL(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	dumper := &mockDumper{dumpData: []byte("data")}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	record, err := svc.CreateBackup(context.Background(), "manual", 14)
	require.NoError(t, err)

	url, err := svc.GetBackupDownloadURL(context.Background(), record.ID)
	require.NoError(t, err)
	require.Contains(t, url, "https://presigned.example.com/")
}

func TestBackupService_ListBackups_Sorted(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	now := time.Now()
	for i := 0; i < 3; i++ {
		_ = svc.saveRecord(context.Background(), &BackupRecord{
			ID:        fmt.Sprintf("rec-%d", i),
			Status:    "completed",
			StartedAt: now.Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
		})
	}

	records, err := svc.ListBackups(context.Background())
	require.NoError(t, err)
	require.Len(t, records, 3)
	// 最新在前
	require.Equal(t, "rec-2", records[0].ID)
	require.Equal(t, "rec-0", records[2].ID)
}

func TestBackupService_TestS3Connection(t *testing.T) {
	repo := newMockSettingRepo()
	store := newMockObjectStore()
	svc := newTestBackupService(repo, &mockDumper{}, store)

	err := svc.TestS3Connection(context.Background(), BackupS3Config{
		Bucket:          "test",
		AccessKeyID:     "ak",
		SecretAccessKey: "sk",
	})
	require.NoError(t, err)
}

func TestBackupService_TestS3Connection_Incomplete(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	err := svc.TestS3Connection(context.Background(), BackupS3Config{
		Bucket: "test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "incomplete")
}

func TestBackupService_Schedule_CronValidation(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	svc.cronSched = nil // 未初始化 cron

	// 启用但 cron 为空
	_, err := svc.UpdateSchedule(context.Background(), BackupScheduleConfig{
		Enabled:  true,
		CronExpr: "",
	})
	require.Error(t, err)

	// 无效的 cron 表达式
	_, err = svc.UpdateSchedule(context.Background(), BackupScheduleConfig{
		Enabled:  true,
		CronExpr: "invalid",
	})
	require.Error(t, err)
}

func TestBackupService_LoadS3Config_Corrupted(t *testing.T) {
	repo := newMockSettingRepo()
	_ = repo.Set(context.Background(), settingKeyBackupS3Config, "not json!!!!")
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	cfg, err := svc.loadS3Config(context.Background())
	require.Error(t, err)
	require.Nil(t, cfg)
}

// ─── Async Backup Tests ───

func TestStartBackup_ReturnsImmediately(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	dumper := &blockingDumper{blockCh: make(chan struct{}), data: []byte("data")}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	record, err := svc.StartBackup(context.Background(), "manual", 14)
	require.NoError(t, err)
	require.Equal(t, "running", record.Status)
	require.NotEmpty(t, record.ID)

	// 释放 dumper 让后台完成
	close(dumper.blockCh)
	svc.wg.Wait()

	// 验证最终状态
	final, err := svc.GetBackupRecord(context.Background(), record.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", final.Status)
	require.Greater(t, final.SizeBytes, int64(0))
}

func TestStartBackup_ConcurrentBlocked(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	dumper := &blockingDumper{blockCh: make(chan struct{}), data: []byte("data")}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	// 第一次启动
	_, err := svc.StartBackup(context.Background(), "manual", 14)
	require.NoError(t, err)

	// 第二次应被阻塞
	_, err = svc.StartBackup(context.Background(), "manual", 14)
	require.ErrorIs(t, err, ErrBackupInProgress)

	close(dumper.blockCh)
	svc.wg.Wait()
}

func TestStartBackup_MultiInstanceLockBlocksConcurrentBackup(t *testing.T) {
	repo1 := newMockSettingRepo()
	repo2 := newMockSettingRepo()
	seedS3Config(t, repo1)
	seedS3Config(t, repo2)

	dumper1 := &blockingDumper{blockCh: make(chan struct{}), data: []byte("data")}
	dumper2 := &mockDumper{dumpData: []byte("other")}
	store := newMockObjectStore()
	lockStore := &sharedBackupLockStore{}

	svc1 := newTestBackupService(repo1, dumper1, store)
	svc2 := newTestBackupService(repo2, dumper2, store)
	svc1.lockStore = lockStore
	svc2.lockStore = lockStore

	_, err := svc1.StartBackup(context.Background(), "manual", 14)
	require.NoError(t, err)

	_, err = svc2.StartBackup(context.Background(), "manual", 14)
	require.ErrorIs(t, err, ErrBackupInProgress)

	close(dumper1.blockCh)
	svc1.wg.Wait()
}

func TestStartBackup_ShuttingDown(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	svc := newTestBackupService(repo, &mockDumper{dumpData: []byte("data")}, newMockObjectStore())

	svc.shuttingDown.Store(true)

	_, err := svc.StartBackup(context.Background(), "manual", 14)
	require.Error(t, err)
	require.Contains(t, err.Error(), "shutting down")
}

func TestRecoverStaleRecords(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	// 模拟一条孤立的 running 记录
	_ = svc.saveRecord(context.Background(), &BackupRecord{
		ID:        "stale-1",
		Status:    "running",
		StartedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	})
	// 模拟一条孤立的恢复中记录
	_ = svc.saveRecord(context.Background(), &BackupRecord{
		ID:            "stale-2",
		Status:        "completed",
		RestoreStatus: "running",
		StartedAt:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	})

	svc.recoverStaleRecords()

	r1, _ := svc.GetBackupRecord(context.Background(), "stale-1")
	require.Equal(t, "failed", r1.Status)
	require.Contains(t, r1.ErrorMsg, "server restart")

	r2, _ := svc.GetBackupRecord(context.Background(), "stale-2")
	require.Equal(t, "failed", r2.RestoreStatus)
	require.Contains(t, r2.RestoreError, "server restart")
}

func TestGracefulShutdown(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	dumper := &blockingDumper{blockCh: make(chan struct{}), data: []byte("data")}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	_, err := svc.StartBackup(context.Background(), "manual", 14)
	require.NoError(t, err)

	// Stop 应该等待备份完成
	done := make(chan struct{})
	go func() {
		svc.Stop()
		close(done)
	}()

	// 短暂等待确认 Stop 还在等待
	select {
	case <-done:
		t.Fatal("Stop returned before backup finished")
	case <-time.After(100 * time.Millisecond):
		// 预期：Stop 还在等待
	}

	// 释放备份
	close(dumper.blockCh)

	// 现在 Stop 应该完成
	select {
	case <-done:
		// 预期
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return after backup finished")
	}
}

func TestStop_CancelsActiveBackupAfterGracePeriod(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	dumper := &blockingDumper{
		blockCh:   make(chan struct{}),
		startedCh: make(chan struct{}, 1),
		data:      []byte("data"),
	}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)
	svc.shutdownGracePeriod = 20 * time.Millisecond
	svc.shutdownCancelWait = time.Second
	svc.cronStopTimeout = 20 * time.Millisecond
	record, err := svc.StartBackup(context.Background(), "manual", 14)
	require.NoError(t, err)
	select {
	case <-dumper.startedCh:
	case <-time.After(time.Second):
		t.Fatal("backup did not start dumping")
	}
	done := make(chan struct{})
	go func() {
		svc.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return after cancelling the active backup")
	}
	final, err := svc.GetBackupRecord(context.Background(), record.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", final.Status)
	require.Contains(t, final.ErrorMsg, "context canceled")
}

func TestStopWithContext_TimesOutWhenOperationsDoNotExit(t *testing.T) {
	svc := newTestBackupService(newMockSettingRepo(), &mockDumper{}, newMockObjectStore())
	svc.shutdownGracePeriod = 10 * time.Millisecond
	svc.shutdownCancelWait = 10 * time.Millisecond
	svc.cronStopTimeout = 10 * time.Millisecond
	svc.wg.Add(1)
	defer svc.wg.Done()
	err := svc.stopWithContext(context.Background())
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestStartRestore_Async(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)

	dumpContent := "-- PostgreSQL dump\nCREATE TABLE test (id int);\n"
	dumper := &mockDumper{dumpData: []byte(dumpContent)}
	store := newMockObjectStore()
	svc := newTestBackupService(repo, dumper, store)

	// 先创建一个备份（同步方式）
	record, err := svc.CreateBackup(context.Background(), "manual", 14)
	require.NoError(t, err)

	// 异步恢复
	restored, err := svc.StartRestore(context.Background(), record.ID)
	require.NoError(t, err)
	require.Equal(t, "running", restored.RestoreStatus)

	svc.wg.Wait()

	// 验证最终状态
	final, err := svc.GetBackupRecord(context.Background(), record.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", final.RestoreStatus)
}

func TestStartRestore_MultiInstanceLockBlocksConcurrentBackup(t *testing.T) {
	repo1 := newMockSettingRepo()
	repo2 := newMockSettingRepo()
	seedS3Config(t, repo1)
	seedS3Config(t, repo2)

	dumpContent := "-- PostgreSQL dump\nCREATE TABLE test (id int);\n"
	dumperRestore := &blockingDumper{blockCh: make(chan struct{}), data: []byte(dumpContent)}
	createDumper := &mockDumper{dumpData: []byte(dumpContent)}
	store := newMockObjectStore()
	lockStore := &sharedBackupLockStore{}

	svc1 := newTestBackupService(repo1, createDumper, store)
	record, err := svc1.CreateBackup(context.Background(), "manual", 14)
	require.NoError(t, err)

	svcRestore := newTestBackupService(repo1, dumperRestore, store)
	svcRestore.lockStore = lockStore
	restoreRecord, err := svcRestore.StartRestore(context.Background(), record.ID)
	require.NoError(t, err)
	require.Equal(t, "running", restoreRecord.RestoreStatus)

	svcBackup := newTestBackupService(repo2, &mockDumper{dumpData: []byte("other")}, store)
	svcBackup.lockStore = lockStore
	_, err = svcBackup.StartBackup(context.Background(), "manual", 14)
	require.ErrorIs(t, err, ErrBackupInProgress)

	close(dumperRestore.blockCh)
	svcRestore.wg.Wait()
}

func TestSettingBackupOperationLockStoreReleaseRequiresOwner(t *testing.T) {
	repo := newMockSettingRepo()
	store := &settingBackupOperationLockStore{repo: repo, nowFunc: time.Now}

	ok, err := store.AcquireBackupOperationLock(context.Background(), "owner-1", "backup", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, store.ReleaseBackupOperationLock(context.Background(), "owner-2"))

	raw, err := repo.GetValue(context.Background(), backupLockKey)
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	require.NoError(t, store.ReleaseBackupOperationLock(context.Background(), "owner-1"))
	raw, err = repo.GetValue(context.Background(), backupLockKey)
	require.NoError(t, err)
	require.Empty(t, raw)
}
