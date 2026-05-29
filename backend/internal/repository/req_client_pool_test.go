package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func useSharedReqClientCacheForTest(t *testing.T, capacity int, ttl time.Duration, now func() time.Time) *sharedReqClientCache {
	t.Helper()

	original := sharedReqClients
	testCache := newSharedReqClientCache(capacity, ttl, now)
	sharedReqClients = testCache
	t.Cleanup(func() {
		closeIdleReqClients(testCache.closeAll())
		sharedReqClients = original
	})
	return testCache
}

func TestGetSharedReqClientForceHTTP2SeparatesCache(t *testing.T) {
	useSharedReqClientCacheForTest(t, 8, time.Hour, time.Now)

	base := reqClientOptions{
		ProxyURL: "http://proxy.local:8080",
		Timeout:  time.Second,
	}
	clientDefault, err := getSharedReqClient(base)
	require.NoError(t, err)

	force := base
	force.ForceHTTP2 = true
	clientForce, err := getSharedReqClient(force)
	require.NoError(t, err)

	require.NotSame(t, clientDefault, clientForce)
	require.NotEqual(t, buildReqClientKey(base), buildReqClientKey(force))
}

func TestGetSharedReqClientReuseCachedClient(t *testing.T) {
	useSharedReqClientCacheForTest(t, 8, time.Hour, time.Now)

	opts := reqClientOptions{
		ProxyURL: "http://proxy.local:8080",
		Timeout:  2 * time.Second,
	}

	first, err := getSharedReqClient(opts)
	require.NoError(t, err)

	second, err := getSharedReqClient(opts)
	require.NoError(t, err)

	require.Same(t, first, second)
}

func TestGetSharedReqClientEvictsLeastRecentlyUsedEntry(t *testing.T) {
	cache := useSharedReqClientCacheForTest(t, 2, time.Hour, time.Now)

	first, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "http://proxy-a.local:8080",
		Timeout:  time.Second,
	})
	require.NoError(t, err)

	second, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "http://proxy-b.local:8080",
		Timeout:  time.Second,
	})
	require.NoError(t, err)

	third, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "http://proxy-c.local:8080",
		Timeout:  time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, third)
	require.Equal(t, 2, cache.len())

	secondAgain, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "http://proxy-b.local:8080",
		Timeout:  time.Second,
	})
	require.NoError(t, err)
	require.Same(t, second, secondAgain)

	firstAgain, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "http://proxy-a.local:8080",
		Timeout:  time.Second,
	})
	require.NoError(t, err)
	require.NotSame(t, first, firstAgain)
}

func TestGetSharedReqClientExpiresStaleEntry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	cache := useSharedReqClientCacheForTest(t, 2, time.Minute, func() time.Time {
		return now
	})

	first, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "http://proxy.local:8080",
		Timeout:  time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, 1, cache.len())

	now = now.Add(2 * time.Minute)

	second, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "http://proxy.local:8080",
		Timeout:  time.Second,
	})
	require.NoError(t, err)
	require.NotSame(t, first, second)
	require.Equal(t, 1, cache.len())
}

func TestGetSharedReqClientImpersonateAndProxy(t *testing.T) {
	useSharedReqClientCacheForTest(t, 8, time.Hour, time.Now)

	opts := reqClientOptions{
		ProxyURL:    "  http://proxy.local:8080  ",
		Timeout:     4 * time.Second,
		Impersonate: true,
	}

	client, err := getSharedReqClient(opts)
	require.NoError(t, err)

	require.NotNil(t, client)
	require.Equal(t, "http://proxy.local:8080|4s|true|false", buildReqClientKey(opts))
}

func TestGetSharedReqClientInvalidProxyURL(t *testing.T) {
	useSharedReqClientCacheForTest(t, 8, time.Hour, time.Now)

	_, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "://missing-scheme",
		Timeout:  time.Second,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid proxy URL")
}

func TestGetSharedReqClientProxyURLMissingHost(t *testing.T) {
	useSharedReqClientCacheForTest(t, 8, time.Hour, time.Now)

	_, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "http://",
		Timeout:  time.Second,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "proxy URL missing host")
}

func TestCreatePrivacyReqClientTimeout30Seconds(t *testing.T) {
	useSharedReqClientCacheForTest(t, 8, time.Hour, time.Now)

	client, err := CreatePrivacyReqClient("http://proxy.local:8080")
	require.NoError(t, err)
	require.Equal(t, 30*time.Second, client.GetClient().Timeout)
}

func TestCloseSharedReqClientsClearsCache(t *testing.T) {
	cache := useSharedReqClientCacheForTest(t, 2, time.Hour, time.Now)

	_, err := getSharedReqClient(reqClientOptions{
		ProxyURL: "http://proxy-a.local:8080",
		Timeout:  time.Second,
	})
	require.NoError(t, err)

	_, err = getSharedReqClient(reqClientOptions{
		ProxyURL: "http://proxy-b.local:8080",
		Timeout:  time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, 2, cache.len())

	CloseSharedReqClients()

	require.Equal(t, 0, cache.len())
}
