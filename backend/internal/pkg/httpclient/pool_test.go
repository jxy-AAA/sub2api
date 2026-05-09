package httpclient

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildValidatedDialContextRejectsPrivateResolvedIP(t *testing.T) {
	dialCalled := false
	dial := func(context.Context, string, string) (net.Conn, error) {
		dialCalled = true
		return nil, errors.New("should not dial private ip")
	}
	resolver := func(context.Context, string, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("169.254.169.254")}, nil
	}

	dialContext := buildValidatedDialContext(dial, resolver)
	_, err := dialContext(context.Background(), "tcp", "example.com:443")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
	require.False(t, dialCalled)
}

func TestBuildValidatedDialContextDialsOnlyValidatedIP(t *testing.T) {
	var dialTargets []string
	dial := func(_ context.Context, _ string, address string) (net.Conn, error) {
		dialTargets = append(dialTargets, address)
		return nil, errors.New("dial blocked in test")
	}
	resolver := func(context.Context, string, string) ([]net.IP, error) {
		return []net.IP{
			net.ParseIP("10.0.0.8"),
			net.ParseIP("203.0.113.7"),
		}, nil
	}

	dialContext := buildValidatedDialContext(dial, resolver)
	_, err := dialContext(context.Background(), "tcp", "api.openai.com:443")
	require.Error(t, err)
	require.Contains(t, err.Error(), "dial to validated ips failed")
	require.Len(t, dialTargets, 1)
	require.Equal(t, "203.0.113.7:443", dialTargets[0])
}

func TestBuildValidatedDialContextRejectsPrivateLiteralHost(t *testing.T) {
	dialCalled := false
	dial := func(context.Context, string, string) (net.Conn, error) {
		dialCalled = true
		return nil, errors.New("should not be called")
	}
	resolver := func(context.Context, string, string) ([]net.IP, error) {
		return nil, errors.New("resolver should not be called for literal ip")
	}

	dialContext := buildValidatedDialContext(dial, resolver)
	_, err := dialContext(context.Background(), "tcp", "127.0.0.1:443")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
	require.False(t, dialCalled)
}
