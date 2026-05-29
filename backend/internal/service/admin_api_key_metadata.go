package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

const (
	AdminAPIKeyPrincipalType = "admin_api_key"
	AdminAPIKeyDisplayName   = "global"
)

type AdminAPIKeyPrincipal struct {
	PrincipalID   string   `json:"principal_id"`
	PrincipalType string   `json:"principal_type"`
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
}

type AdminAPIKeyStatusDetail struct {
	Exists        bool     `json:"exists"`
	MaskedKey     string   `json:"masked_key,omitempty"`
	PrincipalID   string   `json:"principal_id,omitempty"`
	PrincipalType string   `json:"principal_type,omitempty"`
	Name          string   `json:"name,omitempty"`
	Scopes        []string `json:"scopes,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
	ExpiresAt     string   `json:"expires_at,omitempty"`
}

func DeriveAdminAPIKeyPrincipal(key string) AdminAPIKeyPrincipal {
	return deriveAdminAPIKeyPrincipalFromHash(hashAdminAPIKey(key))
}

func DeriveAdminAPIKeyPrincipalFromHash(hash string) AdminAPIKeyPrincipal {
	return deriveAdminAPIKeyPrincipalFromHash(hash)
}

func deriveAdminAPIKeyPrincipalFromHash(hash string) AdminAPIKeyPrincipal {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		sum := sha256.Sum256(nil)
		hash = hex.EncodeToString(sum[:])
	}
	return AdminAPIKeyPrincipal{
		PrincipalID:   "admin-key:" + hash[:12],
		PrincipalType: AdminAPIKeyPrincipalType,
		Name:          AdminAPIKeyDisplayName,
		Scopes:        []string{"admin:*"},
	}
}

func (s *SettingService) GetAdminAPIKeyStatusDetail(ctx context.Context) (*AdminAPIKeyStatusDetail, error) {
	record, err := s.GetAdminAPIKeyRecord(ctx)
	if err != nil {
		return nil, err
	}
	detail := &AdminAPIKeyStatusDetail{
		Exists: record.Exists,
	}
	if !record.Exists {
		return detail, nil
	}

	principal := DeriveAdminAPIKeyPrincipalFromHash(record.Hash)
	detail.MaskedKey = record.MaskedKey
	detail.PrincipalID = principal.PrincipalID
	detail.PrincipalType = principal.PrincipalType
	detail.Name = principal.Name
	detail.Scopes = append([]string(nil), principal.Scopes...)
	if !record.CreatedAt.IsZero() {
		detail.CreatedAt = record.CreatedAt.UTC().Format(time.RFC3339)
	}
	if record.ExpiresAt != nil && !record.ExpiresAt.IsZero() {
		detail.ExpiresAt = record.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return detail, nil
}
