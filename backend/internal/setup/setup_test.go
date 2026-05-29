package setup

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestDecideAdminBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalUsers int64
		adminUsers int64
		should     bool
		reason     string
	}{
		{
			name:       "empty database should create admin",
			totalUsers: 0,
			adminUsers: 0,
			should:     true,
			reason:     adminBootstrapReasonEmptyDatabase,
		},
		{
			name:       "admin exists should skip",
			totalUsers: 10,
			adminUsers: 1,
			should:     false,
			reason:     adminBootstrapReasonAdminExists,
		},
		{
			name:       "users exist without admin should skip",
			totalUsers: 5,
			adminUsers: 0,
			should:     false,
			reason:     adminBootstrapReasonUsersExistWithoutAdmin,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := decideAdminBootstrap(tc.totalUsers, tc.adminUsers)
			if got.shouldCreate != tc.should {
				t.Fatalf("shouldCreate=%v, want %v", got.shouldCreate, tc.should)
			}
			if got.reason != tc.reason {
				t.Fatalf("reason=%q, want %q", got.reason, tc.reason)
			}
		})
	}
}

func TestSetupDefaultAdminConcurrency(t *testing.T) {
	t.Run("simple mode admin uses higher concurrency", func(t *testing.T) {
		t.Setenv("RUN_MODE", "simple")
		if got := setupDefaultAdminConcurrency(); got != simpleModeAdminConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, simpleModeAdminConcurrency)
		}
	})

	t.Run("standard mode keeps existing default", func(t *testing.T) {
		t.Setenv("RUN_MODE", "standard")
		if got := setupDefaultAdminConcurrency(); got != defaultUserConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, defaultUserConcurrency)
		}
	})
}

func TestWriteConfigFileKeepsDefaultUserConcurrency(t *testing.T) {
	t.Setenv("RUN_MODE", "simple")
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("DATABASE_PASSWORD", "db-secret")
	t.Setenv("REDIS_PASSWORD", "redis-secret")
	t.Setenv("JWT_SECRET", "jwt-secret")

	if err := writeConfigFile(&SetupConfig{}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}

	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "user_concurrency: 5") {
		t.Fatalf("config missing default user concurrency, got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "password: ${DATABASE_PASSWORD}") {
		t.Fatalf("config should persist database password by env reference, got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "secret: ${JWT_SECRET}") {
		t.Fatalf("config should persist jwt secret by env reference, got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "secrets_from_env: true") {
		t.Fatalf("config should persist setup bootstrap state, got:\n%s", string(data))
	}
}

func TestValidateSetupDatabaseName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dbName  string
		wantErr bool
	}{
		{name: "valid name", dbName: "sub2api_prod", wantErr: false},
		{name: "starts with number", dbName: "1invalid", wantErr: true},
		{name: "contains dash", dbName: "sub2api-prod", wantErr: true},
		{name: "too long", dbName: strings.Repeat("a", 64), wantErr: true},
		{name: "empty", dbName: "", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateSetupDatabaseName(tc.dbName)
			if tc.wantErr && err == nil {
				t.Fatalf("validateSetupDatabaseName(%q) expected error, got nil", tc.dbName)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validateSetupDatabaseName(%q) unexpected error: %v", tc.dbName, err)
			}
		})
	}
}

func TestManagementDatabaseCandidates(t *testing.T) {
	t.Parallel()

	candidates := managementDatabaseCandidates()
	if len(candidates) < 2 {
		t.Fatalf("expected at least 2 candidates, got %d", len(candidates))
	}
	if candidates[0] != "postgres" {
		t.Fatalf("first candidate = %q, want %q", candidates[0], "postgres")
	}
	if candidates[1] != "template1" {
		t.Fatalf("second candidate = %q, want %q", candidates[1], "template1")
	}
}

func TestGetMigrationTimeout(t *testing.T) {
	t.Run("uses default when env missing", func(t *testing.T) {
		t.Setenv(migrationTimeoutEnvKey, "")
		t.Setenv(legacyMigrationTimeoutKey, "")
		if got := getMigrationTimeout(); got != defaultMigrationTimeout {
			t.Fatalf("getMigrationTimeout()=%s, want %s", got, defaultMigrationTimeout)
		}
	})

	t.Run("uses env seconds when valid", func(t *testing.T) {
		t.Setenv(migrationTimeoutEnvKey, "720")
		t.Setenv(legacyMigrationTimeoutKey, "")
		if got := getMigrationTimeout(); got != 12*time.Minute {
			t.Fatalf("getMigrationTimeout()=%s, want %s", got, 12*time.Minute)
		}
	})

	t.Run("uses legacy duration when seconds env missing", func(t *testing.T) {
		t.Setenv(migrationTimeoutEnvKey, "")
		t.Setenv(legacyMigrationTimeoutKey, "12m")
		if got := getMigrationTimeout(); got != 12*time.Minute {
			t.Fatalf("getMigrationTimeout()=%s, want %s", got, 12*time.Minute)
		}
	})

	t.Run("falls back when env invalid", func(t *testing.T) {
		t.Setenv(migrationTimeoutEnvKey, "invalid")
		t.Setenv(legacyMigrationTimeoutKey, "")
		if got := getMigrationTimeout(); got != defaultMigrationTimeout {
			t.Fatalf("getMigrationTimeout()=%s, want %s", got, defaultMigrationTimeout)
		}
	})

	t.Run("falls back when env non-positive", func(t *testing.T) {
		t.Setenv(migrationTimeoutEnvKey, "0")
		t.Setenv(legacyMigrationTimeoutKey, "")
		if got := getMigrationTimeout(); got != defaultMigrationTimeout {
			t.Fatalf("getMigrationTimeout()=%s, want %s", got, defaultMigrationTimeout)
		}
	})
}
