package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type countingRoundTripper struct {
	closed int
}

func (rt *countingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("not implemented")
}

func (rt *countingRoundTripper) CloseIdleConnections() {
	rt.closed++
}

func useSharedClientCacheForTest(t *testing.T, capacity int, ttl time.Duration, now func() time.Time) *sharedClientCache {
	t.Helper()

	original := sharedClients
	testCache := newSharedClientCache(capacity, ttl, now)
	sharedClients = testCache
	t.Cleanup(func() {
		closeIdleHTTPClients(testCache.closeAll())
		sharedClients = original
	})
	return testCache
}

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

func TestGetClientEvictsLeastRecentlyUsedEntryWhenCapacityReached(t *testing.T) {
	cache := useSharedClientCacheForTest(t, 2, time.Hour, time.Now)

	first, err := GetClient(Options{ProxyURL: "http://proxy-a.local:8080", Timeout: time.Second})
	require.NoError(t, err)

	second, err := GetClient(Options{ProxyURL: "http://proxy-b.local:8080", Timeout: time.Second})
	require.NoError(t, err)

	third, err := GetClient(Options{ProxyURL: "http://proxy-c.local:8080", Timeout: time.Second})
	require.NoError(t, err)

	require.NotNil(t, third)
	require.Equal(t, 2, cache.len())

	secondAgain, err := GetClient(Options{ProxyURL: "http://proxy-b.local:8080", Timeout: time.Second})
	require.NoError(t, err)
	require.Same(t, second, secondAgain)

	firstAgain, err := GetClient(Options{ProxyURL: "http://proxy-a.local:8080", Timeout: time.Second})
	require.NoError(t, err)
	require.NotSame(t, first, firstAgain)
}

func TestGetClientExpiresStaleEntry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	cache := useSharedClientCacheForTest(t, 2, time.Minute, func() time.Time {
		return now
	})

	first, err := GetClient(Options{ProxyURL: "http://proxy.local:8080", Timeout: time.Second})
	require.NoError(t, err)
	require.Equal(t, 1, cache.len())

	now = now.Add(2 * time.Minute)

	second, err := GetClient(Options{ProxyURL: "http://proxy.local:8080", Timeout: time.Second})
	require.NoError(t, err)
	require.NotSame(t, first, second)
	require.Equal(t, 1, cache.len())
}

func TestCloseSharedClientsClearsCacheAndClosesIdleConnections(t *testing.T) {
	cache := useSharedClientCacheForTest(t, 2, time.Hour, time.Now)

	firstTransport := &countingRoundTripper{}
	secondTransport := &countingRoundTripper{}

	_, _, _ = cache.store("first", &http.Client{Transport: firstTransport})
	_, _, _ = cache.store("second", &http.Client{Transport: secondTransport})

	CloseSharedClients()

	require.Equal(t, 0, cache.len())
	require.Equal(t, 1, firstTransport.closed)
	require.Equal(t, 1, secondTransport.closed)
}
