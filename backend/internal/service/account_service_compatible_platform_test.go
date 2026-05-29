package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type testCredentialsAccountRepo struct {
	AccountRepository
	account *Account
	err     error
}

func (r *testCredentialsAccountRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	return r.account, r.err
}

func TestAccountService_TestCredentialsSupportsCompatiblePlatforms(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		platform string
	}{
		{name: "openai compatible", platform: PlatformOpenAICompatible},
		{name: "anthropic compatible", platform: PlatformAnthropicCompatible},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &AccountService{
				accountRepo: &testCredentialsAccountRepo{
					account: &Account{ID: 1, Platform: tc.platform},
				},
			}

			err := svc.TestCredentials(context.Background(), 1)
			require.NoError(t, err)
		})
	}
}
