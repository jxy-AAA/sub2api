package service

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	settingKeyBackupS3Config = "backup_s3_config"
	settingKeyBackupSchedule = "backup_schedule"
	settingKeyBackupRecords  = "backup_records"

	maxBackupRecords         = 100
	backupCronStopTimeout    = 3 * time.Second
	backupShutdownGrace      = 30 * time.Second
	backupShutdownCancelWait = 5 * time.Second
	backupLockLease          = 35 * time.Minute
	backupLockKey            = "backup_operation_lock"
)

var (
	ErrBackupS3NotConfigured = infraerrors.BadRequest("BACKUP_S3_NOT_CONFIGURED", "backup S3 storage is not configured")
	ErrBackupNotFound        = infraerrors.NotFound("BACKUP_NOT_FOUND", "backup record not found")
	ErrBackupInProgress      = infraerrors.Conflict("BACKUP_IN_PROGRESS", "a backup is already in progress")
	ErrRestoreInProgress     = infraerrors.Conflict("RESTORE_IN_PROGRESS", "a restore is already in progress")
	ErrBackupOperationBusy   = infraerrors.Conflict("BACKUP_OPERATION_BUSY", "another backup or restore operation is already in progress")
	ErrBackupRecordsCorrupt  = infraerrors.InternalServer("BACKUP_RECORDS_CORRUPT", "backup records data is corrupted")
	ErrBackupS3ConfigCorrupt = infraerrors.InternalServer("BACKUP_S3_CONFIG_CORRUPT", "backup S3 config data is corrupted")
)

type backupOperationLockStore interface {
	AcquireBackupOperationLock(ctx context.Context, owner string, operation string, lease time.Duration) (bool, error)
	ReleaseBackupOperationLock(ctx context.Context, owner string) error
}

type backupOperationLock struct {
	owner     string
	operation string
}

type backupOperationLockState struct {
	Owner      string `json:"owner"`
	Operation  string `json:"operation"`
	ExpiresAt  string `json:"expires_at"`
	AcquiredAt string `json:"acquired_at,omitempty"`
}

func (s *BackupService) now() time.Time {
	if s == nil || s.nowFunc == nil {
		return time.Now()
	}
	return s.nowFunc()
}

func (s *BackupService) tryAcquireLocalOperation(operation string) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if s.backingUp || s.restoring {
		if operation == "backup" {
			return ErrBackupInProgress
		}
		if operation == "restore" {
			return ErrRestoreInProgress
		}
		return ErrBackupOperationBusy
	}

	switch operation {
	case "backup":
		s.backingUp = true
	case "restore":
		s.restoring = true
	default:
		return ErrBackupOperationBusy
	}
	return nil
}

func (s *BackupService) releaseLocalOperation(operation string) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	switch operation {
	case "backup":
		s.backingUp = false
	case "restore":
		s.restoring = false
	}
}

func (s *BackupService) acquireOperationLock(ctx context.Context, operation string) (*backupOperationLock, error) {
	if err := s.tryAcquireLocalOperation(operation); err != nil {
		return nil, err
	}

	owner := uuid.NewString()
	if s.lockStore == nil {
		return &backupOperationLock{owner: owner, operation: operation}, nil
	}

	ok, err := s.lockStore.AcquireBackupOperationLock(ctx, owner, operation, backupLockLease)
	if err != nil {
		s.releaseLocalOperation(operation)
		return nil, err
	}
	if !ok {
		s.releaseLocalOperation(operation)
		switch operation {
		case "backup":
			return nil, ErrBackupInProgress
		case "restore":
			return nil, ErrRestoreInProgress
		default:
			return nil, ErrBackupOperationBusy
		}
	}

	return &backupOperationLock{owner: owner, operation: operation}, nil
}

func (s *BackupService) releaseOperationLock(ctx context.Context, lock *backupOperationLock) {
	if lock == nil {
		return
	}
	if s.lockStore != nil {
		releaseCtx := ctx
		if releaseCtx == nil {
			releaseCtx = context.Background()
		}
		if err := s.lockStore.ReleaseBackupOperationLock(releaseCtx, lock.owner); err != nil {
			logger.LegacyPrintf("service.backup", "[Backup] release operation lock failed: op=%s err=%v", lock.operation, err)
		}
	}
	s.releaseLocalOperation(lock.operation)
}

type settingBackupOperationLockStore struct {
	repo    SettingRepository
	nowFunc func() time.Time
}

func (s *settingBackupOperationLockStore) now() time.Time {
	if s == nil || s.nowFunc == nil {
		return time.Now()
	}
	return s.nowFunc()
}

func (s *settingBackupOperationLockStore) AcquireBackupOperationLock(ctx context.Context, owner string, operation string, lease time.Duration) (bool, error) {
	if s == nil || s.repo == nil {
		return false, nil
	}
	now := s.now().UTC()
	state := backupOperationLockState{
		Owner:      owner,
		Operation:  operation,
		ExpiresAt:  now.Add(lease).Format(time.RFC3339Nano),
		AcquiredAt: now.Format(time.RFC3339Nano),
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return false, err
	}

	if rawDBRepo, ok := s.repo.(SettingHealthRepository); ok {
		db, err := rawDBRepo.RawDB()
		if err == nil && db != nil {
			return acquireBackupOperationLockSQL(ctx, db, owner, operation, string(payload), now)
		}
	}

	currentRaw, err := s.repo.GetValue(ctx, backupLockKey)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		return false, err
	}
	if currentRaw != "" {
		current := backupOperationLockState{}
		if json.Unmarshal([]byte(currentRaw), &current) == nil && !backupOperationLockExpired(current, now) && current.Owner != owner {
			return false, nil
		}
	}

	if err := s.repo.Set(ctx, backupLockKey, string(payload)); err != nil {
		return false, err
	}
	return true, nil
}

func (s *settingBackupOperationLockStore) ReleaseBackupOperationLock(ctx context.Context, owner string) error {
	if s == nil || s.repo == nil || owner == "" {
		return nil
	}
	if rawDBRepo, ok := s.repo.(SettingHealthRepository); ok {
		db, err := rawDBRepo.RawDB()
		if err == nil && db != nil {
			return releaseBackupOperationLockSQL(ctx, db, owner)
		}
	}

	raw, err := s.repo.GetValue(ctx, backupLockKey)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return nil
		}
		return err
	}
	state := backupOperationLockState{}
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil
	}
	if state.Owner != owner {
		return nil
	}
	return s.repo.Delete(ctx, backupLockKey)
}

func acquireBackupOperationLockSQL(ctx context.Context, db *sql.DB, owner string, operation string, payload string, now time.Time) (bool, error) {
	if db == nil {
		return false, nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var current sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT value
		FROM settings
		WHERE key = $1
		FOR UPDATE
	`, backupLockKey).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if current.Valid {
		state := backupOperationLockState{}
		if json.Unmarshal([]byte(current.String), &state) == nil && !backupOperationLockExpired(state, now) && state.Owner != owner {
			return false, nil
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE settings
			SET value = $2, updated_at = NOW()
			WHERE key = $1
		`, backupLockKey, payload)
	} else {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO settings (key, value, updated_at)
			VALUES ($1, $2, NOW())
		`, backupLockKey, payload)
	}
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	committed = true
	return true, nil
}

func releaseBackupOperationLockSQL(ctx context.Context, db *sql.DB, owner string) error {
	if db == nil || owner == "" {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var current sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT value
		FROM settings
		WHERE key = $1
		FOR UPDATE
	`, backupLockKey).Scan(&current)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if !current.Valid {
		return nil
	}
	state := backupOperationLockState{}
	if err := json.Unmarshal([]byte(current.String), &state); err != nil {
		return nil
	}
	if state.Owner != owner {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM settings WHERE key = $1`, backupLockKey); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func backupOperationLockExpired(state backupOperationLockState, now time.Time) bool {
	if state.ExpiresAt == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, state.ExpiresAt)
	if err != nil {
		return true
	}
	return !expiresAt.After(now)
}

// ─── 接口定义 ───

// DBDumper abstracts database dump/restore operations
type DBDumper interface {
	Dump(ctx context.Context) (io.ReadCloser, error)
	Restore(ctx context.Context, data io.Reader) error
}

// BackupObjectStore abstracts object storage for backup files
type BackupObjectStore interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string) (sizeBytes int64, err error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	PresignURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	HeadBucket(ctx context.Context) error
}

// BackupObjectStoreFactory creates an object store from S3 config
type BackupObjectStoreFactory func(ctx context.Context, cfg *BackupS3Config) (BackupObjectStore, error)

// ─── 数据模型 ───

// BackupS3Config S3 兼容存储配置（支持 Cloudflare R2）
type BackupS3Config struct {
	Endpoint        string `json:"endpoint"` // e.g. https://<account_id>.r2.cloudflarestorage.com
	Region          string `json:"region"`   // R2 用 "auto"
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key,omitempty"` //nolint:revive // field name follows AWS convention
	Prefix          string `json:"prefix"`                      // S3 key 前缀，如 "backups/"
	ForcePathStyle  bool   `json:"force_path_style"`
}

// IsConfigured 检查必要字段是否已配置
func (c *BackupS3Config) IsConfigured() bool {
	return c.Bucket != "" && c.AccessKeyID != "" && c.SecretAccessKey != ""
}

// BackupScheduleConfig 定时备份配置
type BackupScheduleConfig struct {
	Enabled     bool   `json:"enabled"`
	CronExpr    string `json:"cron_expr"`    // cron 表达式，如 "0 2 * * *" 每天凌晨2点
	RetainDays  int    `json:"retain_days"`  // 备份文件过期天数，默认14，0=不自动清理
	RetainCount int    `json:"retain_count"` // 最多保留份数，0=不限制
}

// BackupRecord 备份记录
type BackupRecord struct {
	ID            string `json:"id"`
	Status        string `json:"status"`      // pending, running, completed, failed
	BackupType    string `json:"backup_type"` // postgres
	FileName      string `json:"file_name"`
	S3Key         string `json:"s3_key"`
	SizeBytes     int64  `json:"size_bytes"`
	TriggeredBy   string `json:"triggered_by"` // manual, scheduled
	ErrorMsg      string `json:"error_message,omitempty"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`     // 过期时间
	Progress      string `json:"progress,omitempty"`       // "dumping", "uploading", ""
	RestoreStatus string `json:"restore_status,omitempty"` // "", "running", "completed", "failed"
	RestoreError  string `json:"restore_error,omitempty"`
	RestoredAt    string `json:"restored_at,omitempty"`
}

// BackupService 数据库备份恢复服务
type BackupService struct {
	settingRepo  SettingRepository
	dbCfg        *config.DatabaseConfig
	encryptor    SecretEncryptor
	storeFactory BackupObjectStoreFactory
	dumper       DBDumper

	opMu      sync.Mutex // 保护 backingUp/restoring 标志
	backingUp bool
	restoring bool

	storeMu   sync.Mutex // 保护 store/s3Cfg 缓存
	store     BackupObjectStore
	s3Cfg     *BackupS3Config
	lockStore backupOperationLockStore
	nowFunc   func() time.Time

	recordsMu sync.Mutex // 保护 records 的 load/save 操作

	cronMu      sync.Mutex
	cronSched   *cron.Cron
	cronEntryID cron.EntryID

	wg           sync.WaitGroup     // 追踪活跃的备份/恢复 goroutine
	shuttingDown atomic.Bool        // 阻止新备份启动
	bgCtx        context.Context    // 所有后台操作的 parent context
	bgCancel     context.CancelFunc // 取消所有活跃后台操作

	cronStopTimeout     time.Duration
	shutdownGracePeriod time.Duration
	shutdownCancelWait  time.Duration
}

func NewBackupService(
	settingRepo SettingRepository,
	cfg *config.Config,
	encryptor SecretEncryptor,
	storeFactory BackupObjectStoreFactory,
	dumper DBDumper,
) *BackupService {
	bgCtx, bgCancel := context.WithCancel(context.Background())
	svc := &BackupService{
		settingRepo:         settingRepo,
		dbCfg:               &cfg.Database,
		encryptor:           encryptor,
		storeFactory:        storeFactory,
		dumper:              dumper,
		bgCtx:               bgCtx,
		bgCancel:            bgCancel,
		cronStopTimeout:     backupCronStopTimeout,
		shutdownGracePeriod: backupShutdownGrace,
		shutdownCancelWait:  backupShutdownCancelWait,
		nowFunc:             time.Now,
	}
	svc.lockStore = &settingBackupOperationLockStore{
		repo:    settingRepo,
		nowFunc: svc.now,
	}
	return svc
}

// Start 启动定时备份调度器并清理孤立记录
func (s *BackupService) Start() {
	s.cronSched = cron.New()
	s.cronSched.Start()

	// 清理重启后孤立的 running 记录
	s.recoverStaleRecords()

	// 加载已有的定时配置
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	schedule, err := s.GetSchedule(ctx)
	if err != nil {
		logger.LegacyPrintf("service.backup", "[Backup] 加载定时备份配置失败: %v", err)
		return
	}
	if schedule.Enabled && schedule.CronExpr != "" {
		if err := s.applyCronSchedule(schedule); err != nil {
			logger.LegacyPrintf("service.backup", "[Backup] 应用定时备份配置失败: %v", err)
		}
	}
}

// recoverStaleRecords 启动时将孤立的 running 记录标记为 failed
func (s *BackupService) recoverStaleRecords() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	records, err := s.loadRecords(ctx)
	if err != nil {
		return
	}
	for i := range records {
		if records[i].Status == "running" {
			records[i].Status = "failed"
			records[i].ErrorMsg = "interrupted by server restart"
			records[i].Progress = ""
			records[i].FinishedAt = time.Now().Format(time.RFC3339)
			_ = s.saveRecord(ctx, &records[i])
			logger.LegacyPrintf("service.backup", "[Backup] recovered stale running record: %s", records[i].ID)
		}
		if records[i].RestoreStatus == "running" {
			records[i].RestoreStatus = "failed"
			records[i].RestoreError = "interrupted by server restart"
			_ = s.saveRecord(ctx, &records[i])
			logger.LegacyPrintf("service.backup", "[Backup] recovered stale restoring record: %s", records[i].ID)
		}
	}
}

// Stop 停止定时备份并等待活跃操作完成
func (s *BackupService) Stop() {
	_ = s.StopWithContext(context.Background())
}

func (s *BackupService) StopWithContext(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.shuttingDown.Store(true)
	s.stopCronScheduler()

	done := s.waitForActiveOperations()
	select {
	case <-done:
		logger.LegacyPrintf("service.backup", "[Backup] all active operations finished")
		return nil
	case <-ctx.Done():
		logger.LegacyPrintf("service.backup", "[Backup] shutdown interrupted by caller, cancelling active operations: %v", ctx.Err())
	case <-time.After(s.shutdownGracePeriodValue()):
		logger.LegacyPrintf("service.backup", "[Backup] active operations still running after %s, cancelling", s.shutdownGracePeriodValue())
	}

	if s.bgCancel != nil {
		s.bgCancel()
	}

	select {
	case <-done:
		logger.LegacyPrintf("service.backup", "[Backup] active operations cancelled and cleaned up")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.shutdownCancelWaitValue()):
		err := fmt.Errorf("backup shutdown timed out after cancellation wait %s: %w", s.shutdownCancelWaitValue(), context.DeadlineExceeded)
		logger.LegacyPrintf("service.backup", "[Backup] %v", err)
		return err
	}
}

func (s *BackupService) stopWithContext(ctx context.Context) error {
	return s.StopWithContext(ctx)
}

func (s *BackupService) stopCronScheduler() {
	s.cronMu.Lock()
	cronSched := s.cronSched
	s.cronSched = nil
	s.cronEntryID = 0
	s.cronMu.Unlock()

	if cronSched == nil {
		return
	}

	stopCtx := cronSched.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(s.cronStopTimeoutValue()):
		logger.LegacyPrintf("service.backup", "[Backup] cron stop timed out after %s", s.cronStopTimeoutValue())
	}
}

func (s *BackupService) waitForActiveOperations() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	return done
}

func (s *BackupService) cronStopTimeoutValue() time.Duration {
	if s == nil || s.cronStopTimeout <= 0 {
		return backupCronStopTimeout
	}
	return s.cronStopTimeout
}

func (s *BackupService) shutdownGracePeriodValue() time.Duration {
	if s == nil || s.shutdownGracePeriod <= 0 {
		return backupShutdownGrace
	}
	return s.shutdownGracePeriod
}

func (s *BackupService) shutdownCancelWaitValue() time.Duration {
	if s == nil || s.shutdownCancelWait <= 0 {
		return backupShutdownCancelWait
	}
	return s.shutdownCancelWait
}

// ─── S3 配置管理 ───

func (s *BackupService) GetS3Config(ctx context.Context) (*BackupS3Config, error) {
	cfg, err := s.loadS3Config(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return &BackupS3Config{}, nil
	}
	// 脱敏返回
	cfg.SecretAccessKey = ""
	return cfg, nil
}

func (s *BackupService) UpdateS3Config(ctx context.Context, cfg BackupS3Config) (*BackupS3Config, error) {
	// 如果没提供 secret，保留原有值
	if cfg.SecretAccessKey == "" {
		old, _ := s.loadS3Config(ctx)
		if old != nil {
			cfg.SecretAccessKey = old.SecretAccessKey
		}
	} else {
		// 加密 SecretAccessKey
		encrypted, err := s.encryptor.Encrypt(cfg.SecretAccessKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt secret: %w", err)
		}
		cfg.SecretAccessKey = encrypted
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal s3 config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, settingKeyBackupS3Config, string(data)); err != nil {
		return nil, fmt.Errorf("save s3 config: %w", err)
	}

	// 清除缓存的 S3 客户端
	s.storeMu.Lock()
	s.store = nil
	s.s3Cfg = nil
	s.storeMu.Unlock()

	cfg.SecretAccessKey = ""
	return &cfg, nil
}

func (s *BackupService) TestS3Connection(ctx context.Context, cfg BackupS3Config) error {
	// 如果没提供 secret，用已保存的
	if cfg.SecretAccessKey == "" {
		old, _ := s.loadS3Config(ctx)
		if old != nil {
			cfg.SecretAccessKey = old.SecretAccessKey
		}
	}

	if cfg.Bucket == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return fmt.Errorf("incomplete S3 config: bucket, access_key_id, secret_access_key are required")
	}

	store, err := s.storeFactory(ctx, &cfg)
	if err != nil {
		return err
	}
	return store.HeadBucket(ctx)
}

// ─── 定时备份管理 ───

func (s *BackupService) GetSchedule(ctx context.Context) (*BackupScheduleConfig, error) {
	raw, err := s.settingRepo.GetValue(ctx, settingKeyBackupSchedule)
	if err != nil || raw == "" {
		return &BackupScheduleConfig{}, nil
	}
	var cfg BackupScheduleConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return &BackupScheduleConfig{}, nil
	}
	return &cfg, nil
}

func (s *BackupService) UpdateSchedule(ctx context.Context, cfg BackupScheduleConfig) (*BackupScheduleConfig, error) {
	if cfg.Enabled && cfg.CronExpr == "" {
		return nil, infraerrors.BadRequest("INVALID_CRON", "cron expression is required when schedule is enabled")
	}
	// 验证 cron 表达式
	if cfg.CronExpr != "" {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(cfg.CronExpr); err != nil {
			return nil, infraerrors.BadRequest("INVALID_CRON", fmt.Sprintf("invalid cron expression: %v", err))
		}
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal schedule config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, settingKeyBackupSchedule, string(data)); err != nil {
		return nil, fmt.Errorf("save schedule config: %w", err)
	}

	// 应用或停止定时任务
	if cfg.Enabled {
		if err := s.applyCronSchedule(&cfg); err != nil {
			return nil, err
		}
	} else {
		s.removeCronSchedule()
	}

	return &cfg, nil
}

func (s *BackupService) applyCronSchedule(cfg *BackupScheduleConfig) error {
	s.cronMu.Lock()
	defer s.cronMu.Unlock()

	if s.cronSched == nil {
		return fmt.Errorf("cron scheduler not initialized")
	}

	// 移除旧任务
	if s.cronEntryID != 0 {
		s.cronSched.Remove(s.cronEntryID)
		s.cronEntryID = 0
	}

	entryID, err := s.cronSched.AddFunc(cfg.CronExpr, func() {
		s.runScheduledBackup()
	})
	if err != nil {
		return infraerrors.BadRequest("INVALID_CRON", fmt.Sprintf("failed to schedule: %v", err))
	}
	s.cronEntryID = entryID
	logger.LegacyPrintf("service.backup", "[Backup] 定时备份已启用: %s", cfg.CronExpr)
	return nil
}

func (s *BackupService) removeCronSchedule() {
	s.cronMu.Lock()
	defer s.cronMu.Unlock()
	if s.cronSched != nil && s.cronEntryID != 0 {
		s.cronSched.Remove(s.cronEntryID)
		s.cronEntryID = 0
		logger.LegacyPrintf("service.backup", "[Backup] 定时备份已停用")
	}
}

func (s *BackupService) runScheduledBackup() {
	s.wg.Add(1)
	defer s.wg.Done()

	ctx, cancel := context.WithTimeout(s.bgCtx, 30*time.Minute)
	defer cancel()

	// 读取定时备份配置中的过期天数
	schedule, _ := s.GetSchedule(ctx)
	expireDays := 14 // 默认14天过期
	if schedule != nil && schedule.RetainDays > 0 {
		expireDays = schedule.RetainDays
	}

	logger.LegacyPrintf("service.backup", "[Backup] 开始执行定时备份, 过期天数: %d", expireDays)
	record, err := s.CreateBackup(ctx, "scheduled", expireDays)
	if err != nil {
		if errors.Is(err, ErrBackupInProgress) {
			logger.LegacyPrintf("service.backup", "[Backup] 定时备份跳过: 已有备份正在进行中")
		} else {
			logger.LegacyPrintf("service.backup", "[Backup] 定时备份失败: %v", err)
		}
		return
	}
	logger.LegacyPrintf("service.backup", "[Backup] 定时备份完成: id=%s size=%d", record.ID, record.SizeBytes)

	// 清理过期备份（复用已加载的 schedule）
	if schedule == nil {
		return
	}
	if err := s.cleanupOldBackups(ctx, schedule); err != nil {
		logger.LegacyPrintf("service.backup", "[Backup] 清理过期备份失败: %v", err)
	}
}

// ─── 备份/恢复核心 ───

// CreateBackup 创建全量数据库备份并上传到 S3（流式处理）
// expireDays: 备份过期天数，0=永不过期，默认14天
func (s *BackupService) CreateBackup(ctx context.Context, triggeredBy string, expireDays int) (*BackupRecord, error) {
	if s.shuttingDown.Load() {
		return nil, infraerrors.ServiceUnavailable("SERVER_SHUTTING_DOWN", "server is shutting down")
	}

	lock, err := s.acquireOperationLock(ctx, "backup")
	if err != nil {
		return nil, err
	}
	defer s.releaseOperationLock(context.Background(), lock)

	s3Cfg, err := s.loadS3Config(ctx)
	if err != nil {
		return nil, err
	}
	if s3Cfg == nil || !s3Cfg.IsConfigured() {
		return nil, ErrBackupS3NotConfigured
	}

	objectStore, err := s.getOrCreateStore(ctx, s3Cfg)
	if err != nil {
		return nil, fmt.Errorf("init object store: %w", err)
	}

	now := time.Now()
	backupID := uuid.New().String()[:8]
	fileName := fmt.Sprintf("%s_%s.sql.gz", s.dbCfg.DBName, now.Format("20060102_150405"))
	s3Key := s.buildS3Key(s3Cfg, fileName)

	var expiresAt string
	if expireDays > 0 {
		expiresAt = now.AddDate(0, 0, expireDays).Format(time.RFC3339)
	}

	record := &BackupRecord{
		ID:          backupID,
		Status:      "running",
		BackupType:  "postgres",
		FileName:    fileName,
		S3Key:       s3Key,
		TriggeredBy: triggeredBy,
		StartedAt:   now.Format(time.RFC3339),
		ExpiresAt:   expiresAt,
	}

	// 流式执行: pg_dump -> gzip -> S3 upload
	dumpReader, err := s.dumper.Dump(ctx)
	if err != nil {
		record.Status = "failed"
		record.ErrorMsg = fmt.Sprintf("pg_dump failed: %v", err)
		record.FinishedAt = time.Now().Format(time.RFC3339)
		_ = s.saveRecord(ctx, record)
		return record, fmt.Errorf("pg_dump: %w", err)
	}

	// 使用 io.Pipe 将 gzip 压缩数据流式传递给 S3 上传
	pr, pw := io.Pipe()
	gzipDone := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				pw.CloseWithError(fmt.Errorf("gzip goroutine panic: %v", r)) //nolint:errcheck
				gzipDone <- fmt.Errorf("gzip goroutine panic: %v", r)
			}
		}()
		gzWriter := gzip.NewWriter(pw)
		var gzErr error
		_, gzErr = io.Copy(gzWriter, dumpReader)
		if closeErr := gzWriter.Close(); closeErr != nil && gzErr == nil {
			gzErr = closeErr
		}
		if closeErr := dumpReader.Close(); closeErr != nil && gzErr == nil {
			gzErr = closeErr
		}
		if gzErr != nil {
			_ = pw.CloseWithError(gzErr)
		} else {
			_ = pw.Close()
		}
		gzipDone <- gzErr
	}()

	contentType := "application/gzip"
	sizeBytes, err := objectStore.Upload(ctx, s3Key, pr, contentType)
	if err != nil {
		_ = pr.CloseWithError(err) // 确保 gzip goroutine 不会悬挂
		gzErr := <-gzipDone        // 安全等待 gzip goroutine 完成
		record.Status = "failed"
		errMsg := fmt.Sprintf("S3 upload failed: %v", err)
		if gzErr != nil {
			errMsg = fmt.Sprintf("gzip/dump failed: %v", gzErr)
		}
		record.ErrorMsg = errMsg
		record.FinishedAt = time.Now().Format(time.RFC3339)
		_ = s.saveRecord(ctx, record)
		return record, fmt.Errorf("backup upload: %w", err)
	}
	<-gzipDone // 确保 gzip goroutine 已退出

	record.SizeBytes = sizeBytes
	record.Status = "completed"
	record.FinishedAt = time.Now().Format(time.RFC3339)
	if err := s.saveRecord(ctx, record); err != nil {
		logger.LegacyPrintf("service.backup", "[Backup] 保存备份记录失败: %v", err)
	}

	return record, nil
}

// StartBackup 异步创建备份，立即返回 running 状态的记录
func (s *BackupService) StartBackup(ctx context.Context, triggeredBy string, expireDays int) (*BackupRecord, error) {
	if s.shuttingDown.Load() {
		return nil, infraerrors.ServiceUnavailable("SERVER_SHUTTING_DOWN", "server is shutting down")
	}

	lock, err := s.acquireOperationLock(ctx, "backup")
	if err != nil {
		return nil, err
	}

	// 初始化阶段出错时自动重置标志
	launched := false
	defer func() {
		if !launched {
			s.releaseOperationLock(context.Background(), lock)
		}
	}()

	// 在返回前加载 S3 配置和创建 store，避免 goroutine 中配置被修改
	s3Cfg, err := s.loadS3Config(ctx)
	if err != nil {
		return nil, err
	}
	if s3Cfg == nil || !s3Cfg.IsConfigured() {
		return nil, ErrBackupS3NotConfigured
	}

	objectStore, err := s.getOrCreateStore(ctx, s3Cfg)
	if err != nil {
		return nil, fmt.Errorf("init object store: %w", err)
	}

	now := time.Now()
	backupID := uuid.New().String()[:8]
	fileName := fmt.Sprintf("%s_%s.sql.gz", s.dbCfg.DBName, now.Format("20060102_150405"))
	s3Key := s.buildS3Key(s3Cfg, fileName)

	var expiresAt string
	if expireDays > 0 {
		expiresAt = now.AddDate(0, 0, expireDays).Format(time.RFC3339)
	}

	record := &BackupRecord{
		ID:          backupID,
		Status:      "running",
		BackupType:  "postgres",
		FileName:    fileName,
		S3Key:       s3Key,
		TriggeredBy: triggeredBy,
		StartedAt:   now.Format(time.RFC3339),
		ExpiresAt:   expiresAt,
		Progress:    "pending",
	}

	if err := s.saveRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("save initial record: %w", err)
	}

	launched = true
	// 在启动 goroutine 前完成拷贝，避免数据竞争
	result := *record

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.releaseOperationLock(context.Background(), lock)
		defer func() {
			if r := recover(); r != nil {
				logger.LegacyPrintf("service.backup", "[Backup] panic recovered: %v", r)
				record.Status = "failed"
				record.ErrorMsg = fmt.Sprintf("internal panic: %v", r)
				record.Progress = ""
				record.FinishedAt = time.Now().Format(time.RFC3339)
				_ = s.saveRecord(context.Background(), record)
			}
		}()
		s.executeBackup(record, objectStore)
	}()

	return &result, nil
}

// executeBackup 后台执行备份（独立于 HTTP context）
func (s *BackupService) executeBackup(record *BackupRecord, objectStore BackupObjectStore) {
	ctx, cancel := context.WithTimeout(s.bgCtx, 30*time.Minute)
	defer cancel()

	// 阶段1: pg_dump
	record.Progress = "dumping"
	_ = s.saveRecord(ctx, record)

	dumpReader, err := s.dumper.Dump(ctx)
	if err != nil {
		record.Status = "failed"
		record.ErrorMsg = fmt.Sprintf("pg_dump failed: %v", err)
		record.Progress = ""
		record.FinishedAt = time.Now().Format(time.RFC3339)
		_ = s.saveRecord(context.Background(), record)
		return
	}

	// 阶段2: gzip + upload
	record.Progress = "uploading"
	_ = s.saveRecord(ctx, record)

	pr, pw := io.Pipe()
	gzipDone := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				pw.CloseWithError(fmt.Errorf("gzip goroutine panic: %v", r)) //nolint:errcheck
				gzipDone <- fmt.Errorf("gzip goroutine panic: %v", r)
			}
		}()
		gzWriter := gzip.NewWriter(pw)
		var gzErr error
		_, gzErr = io.Copy(gzWriter, dumpReader)
		if closeErr := gzWriter.Close(); closeErr != nil && gzErr == nil {
			gzErr = closeErr
		}
		if closeErr := dumpReader.Close(); closeErr != nil && gzErr == nil {
			gzErr = closeErr
		}
		if gzErr != nil {
			_ = pw.CloseWithError(gzErr)
		} else {
			_ = pw.Close()
		}
		gzipDone <- gzErr
	}()

	contentType := "application/gzip"
	sizeBytes, err := objectStore.Upload(ctx, record.S3Key, pr, contentType)
	if err != nil {
		_ = pr.CloseWithError(err) // 确保 gzip goroutine 不会悬挂
		gzErr := <-gzipDone        // 安全等待 gzip goroutine 完成
		record.Status = "failed"
		errMsg := fmt.Sprintf("S3 upload failed: %v", err)
		if gzErr != nil {
			errMsg = fmt.Sprintf("gzip/dump failed: %v", gzErr)
		}
		record.ErrorMsg = errMsg
		record.Progress = ""
		record.FinishedAt = time.Now().Format(time.RFC3339)
		_ = s.saveRecord(context.Background(), record)
		return
	}
	<-gzipDone // 确保 gzip goroutine 已退出

	record.SizeBytes = sizeBytes
	record.Status = "completed"
	record.Progress = ""
	record.FinishedAt = time.Now().Format(time.RFC3339)
	if err := s.saveRecord(context.Background(), record); err != nil {
		logger.LegacyPrintf("service.backup", "[Backup] 保存备份记录失败: %v", err)
	}
}

// RestoreBackup 从 S3 下载备份并流式恢复到数据库
func (s *BackupService) RestoreBackup(ctx context.Context, backupID string) error {
	lock, err := s.acquireOperationLock(ctx, "restore")
	if err != nil {
		return err
	}
	defer s.releaseOperationLock(context.Background(), lock)

	record, err := s.GetBackupRecord(ctx, backupID)
	if err != nil {
		return err
	}
	if record.Status != "completed" {
		return infraerrors.BadRequest("BACKUP_NOT_COMPLETED", "can only restore from a completed backup")
	}

	s3Cfg, err := s.loadS3Config(ctx)
	if err != nil {
		return err
	}
	objectStore, err := s.getOrCreateStore(ctx, s3Cfg)
	if err != nil {
		return fmt.Errorf("init object store: %w", err)
	}

	// 从 S3 流式下载
	body, err := objectStore.Download(ctx, record.S3Key)
	if err != nil {
		return fmt.Errorf("S3 download failed: %w", err)
	}
	defer func() { _ = body.Close() }()

	// 流式解压 gzip -> psql（不将全部数据加载到内存）
	gzReader, err := gzip.NewReader(body)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer func() { _ = gzReader.Close() }()

	// 流式恢复
	if err := s.dumper.Restore(ctx, gzReader); err != nil {
		return fmt.Errorf("pg restore: %w", err)
	}

	return nil
}

// StartRestore 异步恢复备份，立即返回
func (s *BackupService) StartRestore(ctx context.Context, backupID string) (*BackupRecord, error) {
	if s.shuttingDown.Load() {
		return nil, infraerrors.ServiceUnavailable("SERVER_SHUTTING_DOWN", "server is shutting down")
	}

	lock, err := s.acquireOperationLock(ctx, "restore")
	if err != nil {
		return nil, err
	}

	// 初始化阶段出错时自动重置标志
	launched := false
	defer func() {
		if !launched {
			s.releaseOperationLock(context.Background(), lock)
		}
	}()

	record, err := s.GetBackupRecord(ctx, backupID)
	if err != nil {
		return nil, err
	}
	if record.Status != "completed" {
		return nil, infraerrors.BadRequest("BACKUP_NOT_COMPLETED", "can only restore from a completed backup")
	}

	s3Cfg, err := s.loadS3Config(ctx)
	if err != nil {
		return nil, err
	}
	objectStore, err := s.getOrCreateStore(ctx, s3Cfg)
	if err != nil {
		return nil, fmt.Errorf("init object store: %w", err)
	}

	record.RestoreStatus = "running"
	if err := s.saveRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("save restore record: %w", err)
	}

	launched = true
	result := *record

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.releaseOperationLock(context.Background(), lock)
		defer func() {
			if r := recover(); r != nil {
				logger.LegacyPrintf("service.backup", "[Backup] restore panic recovered: %v", r)
				record.RestoreStatus = "failed"
				record.RestoreError = fmt.Sprintf("internal panic: %v", r)
				_ = s.saveRecord(context.Background(), record)
			}
		}()
		s.executeRestore(record, objectStore)
	}()

	return &result, nil
}

// executeRestore 后台执行恢复
func (s *BackupService) executeRestore(record *BackupRecord, objectStore BackupObjectStore) {
	ctx, cancel := context.WithTimeout(s.bgCtx, 30*time.Minute)
	defer cancel()

	body, err := objectStore.Download(ctx, record.S3Key)
	if err != nil {
		record.RestoreStatus = "failed"
		record.RestoreError = fmt.Sprintf("S3 download failed: %v", err)
		_ = s.saveRecord(context.Background(), record)
		return
	}
	defer func() { _ = body.Close() }()

	gzReader, err := gzip.NewReader(body)
	if err != nil {
		record.RestoreStatus = "failed"
		record.RestoreError = fmt.Sprintf("gzip reader: %v", err)
		_ = s.saveRecord(context.Background(), record)
		return
	}
	defer func() { _ = gzReader.Close() }()

	if err := s.dumper.Restore(ctx, gzReader); err != nil {
		record.RestoreStatus = "failed"
		record.RestoreError = fmt.Sprintf("pg restore: %v", err)
		_ = s.saveRecord(context.Background(), record)
		return
	}

	record.RestoreStatus = "completed"
	record.RestoredAt = time.Now().Format(time.RFC3339)
	if err := s.saveRecord(context.Background(), record); err != nil {
		logger.LegacyPrintf("service.backup", "[Backup] 保存恢复记录失败: %v", err)
	}
}

// ─── 备份记录管理 ───

func (s *BackupService) ListBackups(ctx context.Context) ([]BackupRecord, error) {
	records, err := s.loadRecords(ctx)
	if err != nil {
		return nil, err
	}
	// 倒序返回（最新在前）
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt > records[j].StartedAt
	})
	return records, nil
}

func (s *BackupService) GetBackupRecord(ctx context.Context, backupID string) (*BackupRecord, error) {
	records, err := s.loadRecords(ctx)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == backupID {
			return &records[i], nil
		}
	}
	return nil, ErrBackupNotFound
}

func (s *BackupService) DeleteBackup(ctx context.Context, backupID string) error {
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()

	records, err := s.loadRecordsLocked(ctx)
	if err != nil {
		return err
	}

	var found *BackupRecord
	var remaining []BackupRecord
	for i := range records {
		if records[i].ID == backupID {
			found = &records[i]
		} else {
			remaining = append(remaining, records[i])
		}
	}
	if found == nil {
		return ErrBackupNotFound
	}

	// 从 S3 删除
	if found.S3Key != "" && found.Status == "completed" {
		s3Cfg, err := s.loadS3Config(ctx)
		if err == nil && s3Cfg != nil && s3Cfg.IsConfigured() {
			objectStore, err := s.getOrCreateStore(ctx, s3Cfg)
			if err == nil {
				_ = objectStore.Delete(ctx, found.S3Key)
			}
		}
	}

	return s.saveRecordsLocked(ctx, remaining)
}

// GetBackupDownloadURL 获取备份文件预签名下载 URL
func (s *BackupService) GetBackupDownloadURL(ctx context.Context, backupID string) (string, error) {
	record, err := s.GetBackupRecord(ctx, backupID)
	if err != nil {
		return "", err
	}
	if record.Status != "completed" {
		return "", infraerrors.BadRequest("BACKUP_NOT_COMPLETED", "backup is not completed")
	}

	s3Cfg, err := s.loadS3Config(ctx)
	if err != nil {
		return "", err
	}
	objectStore, err := s.getOrCreateStore(ctx, s3Cfg)
	if err != nil {
		return "", err
	}

	url, err := objectStore.PresignURL(ctx, record.S3Key, 1*time.Hour)
	if err != nil {
		return "", fmt.Errorf("presign url: %w", err)
	}
	return url, nil
}

// ─── 内部方法 ───

func (s *BackupService) loadS3Config(ctx context.Context) (*BackupS3Config, error) {
	raw, err := s.settingRepo.GetValue(ctx, settingKeyBackupS3Config)
	if err != nil || raw == "" {
		return nil, nil //nolint:nilnil // no config is a valid state
	}
	var cfg BackupS3Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, ErrBackupS3ConfigCorrupt
	}
	// 解密 SecretAccessKey
	if cfg.SecretAccessKey != "" {
		decrypted, err := s.encryptor.Decrypt(cfg.SecretAccessKey)
		if err != nil {
			// 兼容未加密的旧数据：如果解密失败，保持原值
			logger.LegacyPrintf("service.backup", "[Backup] S3 SecretAccessKey 解密失败（可能是旧的未加密数据）: %v", err)
		} else {
			cfg.SecretAccessKey = decrypted
		}
	}
	return &cfg, nil
}

func (s *BackupService) getOrCreateStore(ctx context.Context, cfg *BackupS3Config) (BackupObjectStore, error) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	if s.store != nil && s.s3Cfg != nil {
		return s.store, nil
	}

	if cfg == nil {
		return nil, ErrBackupS3NotConfigured
	}

	store, err := s.storeFactory(ctx, cfg)
	if err != nil {
		return nil, err
	}
	s.store = store
	s.s3Cfg = cfg
	return store, nil
}

func (s *BackupService) buildS3Key(cfg *BackupS3Config, fileName string) string {
	prefix := strings.TrimRight(cfg.Prefix, "/")
	if prefix == "" {
		prefix = "backups"
	}
	return fmt.Sprintf("%s/%s/%s", prefix, time.Now().Format("2006/01/02"), fileName)
}

// loadRecords 加载备份记录，区分"无数据"和"数据损坏"
func (s *BackupService) loadRecords(ctx context.Context) ([]BackupRecord, error) {
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()
	return s.loadRecordsLocked(ctx)
}

// loadRecordsLocked 在已持有 recordsMu 锁的情况下加载记录
func (s *BackupService) loadRecordsLocked(ctx context.Context) ([]BackupRecord, error) {
	raw, err := s.settingRepo.GetValue(ctx, settingKeyBackupRecords)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return nil, nil //nolint:nilnil // no records is a valid state
		}
		return nil, fmt.Errorf("load backup records: %w", err)
	}
	if raw == "" {
		return nil, nil //nolint:nilnil // no records is a valid state
	}
	var records []BackupRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil, ErrBackupRecordsCorrupt
	}
	return records, nil
}

// saveRecordsLocked 在已持有 recordsMu 锁的情况下保存记录
func (s *BackupService) saveRecordsLocked(ctx context.Context, records []BackupRecord) error {
	data, err := json.Marshal(records)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, settingKeyBackupRecords, string(data))
}

// saveRecord 保存单条记录（带互斥锁保护）
func (s *BackupService) saveRecord(ctx context.Context, record *BackupRecord) error {
	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()

	records, err := s.loadRecordsLocked(ctx)
	if err != nil {
		return err
	}

	// 更新已有记录或追加
	found := false
	for i := range records {
		if records[i].ID == record.ID {
			records[i] = *record
			found = true
			break
		}
	}
	if !found {
		records = append(records, *record)
	}

	// 限制记录数量
	if len(records) > maxBackupRecords {
		records = records[len(records)-maxBackupRecords:]
	}

	return s.saveRecordsLocked(ctx, records)
}

func (s *BackupService) cleanupOldBackups(ctx context.Context, schedule *BackupScheduleConfig) error {
	if schedule == nil {
		return nil
	}

	s.recordsMu.Lock()
	defer s.recordsMu.Unlock()

	records, err := s.loadRecordsLocked(ctx)
	if err != nil {
		return err
	}

	// 按时间倒序
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt > records[j].StartedAt
	})

	var toDelete []BackupRecord
	var toKeep []BackupRecord

	for i, r := range records {
		shouldDelete := false

		// 按保留份数清理
		if schedule.RetainCount > 0 && i >= schedule.RetainCount {
			shouldDelete = true
		}

		// 按保留天数清理
		if schedule.RetainDays > 0 && r.StartedAt != "" {
			startedAt, err := time.Parse(time.RFC3339, r.StartedAt)
			if err == nil && time.Since(startedAt) > time.Duration(schedule.RetainDays)*24*time.Hour {
				shouldDelete = true
			}
		}

		if shouldDelete && r.Status == "completed" {
			toDelete = append(toDelete, r)
		} else {
			toKeep = append(toKeep, r)
		}
	}

	// 删除 S3 上的文件
	for _, r := range toDelete {
		if r.S3Key != "" {
			_ = s.deleteS3Object(ctx, r.S3Key)
		}
	}

	if len(toDelete) > 0 {
		logger.LegacyPrintf("service.backup", "[Backup] 自动清理了 %d 个过期备份", len(toDelete))
		return s.saveRecordsLocked(ctx, toKeep)
	}
	return nil
}

func (s *BackupService) deleteS3Object(ctx context.Context, key string) error {
	s3Cfg, err := s.loadS3Config(ctx)
	if err != nil || s3Cfg == nil {
		return nil
	}
	objectStore, err := s.getOrCreateStore(ctx, s3Cfg)
	if err != nil {
		return err
	}
	return objectStore.Delete(ctx, key)
}
