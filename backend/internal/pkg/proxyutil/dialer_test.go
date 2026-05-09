package proxyutil

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureTransportProxy_Nil(t *testing.T) {
	transport := &http.Transport{}
	err := ConfigureTransportProxy(transport, nil)

	require.NoError(t, err)
	assert.Nil(t, transport.Proxy, "nil proxy should not set Proxy")
	assert.Nil(t, transport.DialContext, "nil proxy should not set DialContext")
}

func TestConfigureTransportProxy_HTTP(t *testing.T) {
	transport := &http.Transport{}
	proxyURL, _ := url.Parse("http://proxy.example.com:8080")

	err := ConfigureTransportProxy(transport, proxyURL)

	require.NoError(t, err)
	assert.NotNil(t, transport.Proxy, "HTTP proxy should set Proxy")
	assert.Nil(t, transport.DialContext, "HTTP proxy should not set DialContext")
}

func TestConfigureTransportProxy_HTTPS(t *testing.T) {
	transport := &http.Transport{}
	proxyURL, _ := url.Parse("https://secure-proxy.example.com:8443")

	err := ConfigureTransportProxy(transport, proxyURL)

	require.NoError(t, err)
	assert.NotNil(t, transport.Proxy, "HTTPS proxy should set Proxy")
	assert.Nil(t, transport.DialContext, "HTTPS proxy should not set DialContext")
}

func TestConfigureTransportProxy_SOCKS5(t *testing.T) {
	transport := &http.Transport{}
	proxyURL, _ := url.Parse("socks5://socks.example.com:1080")

	err := ConfigureTransportProxy(transport, proxyURL)

	require.NoError(t, err)
	assert.Nil(t, transport.Proxy, "SOCKS5 proxy should not set Proxy")
	assert.NotNil(t, transport.DialContext, "SOCKS5 proxy should set DialContext")
}

func TestConfigureTransportProxy_SOCKS5H(t *testing.T) {
	transport := &http.Transport{}
	proxyURL, _ := url.Parse("socks5h://socks.example.com:1080")

	err := ConfigureTransportProxy(transport, proxyURL)

	require.NoError(t, err)
	assert.Nil(t, transport.Proxy, "SOCKS5H proxy should not set Proxy")
	assert.NotNil(t, transport.DialContext, "SOCKS5H proxy should set DialContext")
}

func TestConfigureTransportProxy_CaseInsensitive(t *testing.T) {
	testCases := []struct {
		scheme   string
		useProxy bool // true = uses Transport.Proxy, false = uses DialContext
	}{
		{"HTTP://proxy.example.com:8080", true},
		{"Http://proxy.example.com:8080", true},
		{"HTTPS://proxy.example.com:8443", true},
		{"Https://proxy.example.com:8443", true},
		{"SOCKS5://socks.example.com:1080", false},
		{"Socks5://socks.example.com:1080", false},
		{"SOCKS5H://socks.example.com:1080", false},
		{"Socks5h://socks.example.com:1080", false},
	}

	for _, tc := range testCases {
		t.Run(tc.scheme, func(t *testing.T) {
			transport := &http.Transport{}
			proxyURL, _ := url.Parse(tc.scheme)

			err := ConfigureTransportProxy(transport, proxyURL)

			require.NoError(t, err)
			if tc.useProxy {
				assert.NotNil(t, transport.Proxy)
				assert.Nil(t, transport.DialContext)
			} else {
				assert.Nil(t, transport.Proxy)
				assert.NotNil(t, transport.DialContext)
			}
		})
	}
}

func TestConfigureTransportProxy_Unsupported(t *testing.T) {
	testCases := []string{
		"ftp://ftp.example.com",
		"file:///path/to/file",
		"unknown://example.com",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			transport := &http.Transport{}
			proxyURL, _ := url.Parse(tc)

			err := ConfigureTransportProxy(transport, proxyURL)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported proxy scheme")
		})
	}
}

func TestConfigureTransportProxy_WithAuth(t *testing.T) {
	transport := &http.Transport{}
	proxyURL, _ := url.Parse("socks5://user:password@socks.example.com:1080")

	err := ConfigureTransportProxy(transport, proxyURL)

	require.NoError(t, err)
	assert.NotNil(t, transport.DialContext, "SOCKS5 with auth should set DialContext")
}

func TestConfigureTransportProxy_EmptyScheme(t *testing.T) {
	transport := &http.Transport{}
	// 空 scheme 的 URL
	proxyURL := &url.URL{Host: "proxy.example.com:8080"}

	err := ConfigureTransportProxy(transport, proxyURL)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported proxy scheme")
}

func TestConfigureTransportProxy_PreservesExistingConfig(t *testing.T) {
	// 验证代理配置不会覆盖 Transport 的其他配置
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}
	proxyURL, _ := url.Parse("socks5://socks.example.com:1080")

	err := ConfigureTransportProxy(transport, proxyURL)

	require.NoError(t, err)
	assert.Equal(t, 100, transport.MaxIdleConns, "MaxIdleConns should be preserved")
	assert.Equal(t, 10, transport.MaxIdleConnsPerHost, "MaxIdleConnsPerHost should be preserved")
	assert.NotNil(t, transport.DialContext, "DialContext should be set")
}

func TestConfigureTransportProxy_IPv6(t *testing.T) {
	testCases := []struct {
		name     string
		proxyURL string
	}{
		{"SOCKS5H with IPv6 loopback", "socks5h://[::1]:1080"},
		{"SOCKS5 with full IPv6", "socks5://[2001:db8::1]:1080"},
		{"HTTP with IPv6", "http://[::1]:8080"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &http.Transport{}
			proxyURL, err := url.Parse(tc.proxyURL)
			require.NoError(t, err, "URL should be parseable")

			err = ConfigureTransportProxy(transport, proxyURL)
			require.NoError(t, err)
		})
	}
}

func TestConfigureTransportProxy_SpecialCharsInPassword(t *testing.T) {
	testCases := []struct {
		name     string
		proxyURL string
	}{
		// 密码包含 @ 符号（URL 编码为 %40）
		{"password with @", "socks5://user:p%40ssword@proxy.example.com:1080"},
		// 密码包含 : 符号（URL 编码为 %3A）
		{"password with :", "socks5://user:pass%3Aword@proxy.example.com:1080"},
		// 密码包含 / 符号（URL 编码为 %2F）
		{"password with /", "socks5://user:pass%2Fword@proxy.example.com:1080"},
		// 复杂密码
		{"complex password", "socks5h://admin:P%40ss%3Aw0rd%2F123@proxy.example.com:1080"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &http.Transport{}
			proxyURL, err := url.Parse(tc.proxyURL)
			require.NoError(t, err, "URL should be parseable")

			err = ConfigureTransportProxy(transport, proxyURL)
			require.NoError(t, err)
			assert.NotNil(t, transport.DialContext, "SOCKS5 should set DialContext")
		})
	}
}

func TestConfigureTransportProxyWithTargetResolver_SOCKS5UsesResolvedTarget(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	targetCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()

		greeting := make([]byte, 3)
		if _, err := io.ReadFull(conn, greeting); err != nil {
			errCh <- err
			return
		}
		if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
			errCh <- err
			return
		}

		header := make([]byte, 4)
		if _, err := io.ReadFull(conn, header); err != nil {
			errCh <- err
			return
		}
		var host string
		switch header[3] {
		case 0x01:
			buf := make([]byte, net.IPv4len)
			if _, err := io.ReadFull(conn, buf); err != nil {
				errCh <- err
				return
			}
			host = net.IP(buf).String()
		case 0x03:
			length := make([]byte, 1)
			if _, err := io.ReadFull(conn, length); err != nil {
				errCh <- err
				return
			}
			buf := make([]byte, int(length[0]))
			if _, err := io.ReadFull(conn, buf); err != nil {
				errCh <- err
				return
			}
			host = string(buf)
		case 0x04:
			buf := make([]byte, net.IPv6len)
			if _, err := io.ReadFull(conn, buf); err != nil {
				errCh <- err
				return
			}
			host = net.IP(buf).String()
		default:
			errCh <- errors.New("unexpected SOCKS address type")
			return
		}
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, portBuf); err != nil {
			errCh <- err
			return
		}
		port := int(portBuf[0])<<8 | int(portBuf[1])
		targetCh <- net.JoinHostPort(host, strconv.Itoa(port))
		_, _ = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		errCh <- nil
	}()

	transport := &http.Transport{}
	proxyURL, err := url.Parse("socks5://" + ln.Addr().String())
	require.NoError(t, err)
	resolverCalled := false
	err = ConfigureTransportProxyWithTargetResolver(transport, proxyURL, func(ctx context.Context, network, addr string) (string, error) {
		resolverCalled = true
		require.Equal(t, "tcp", network)
		require.Equal(t, "example.com:443", addr)
		return "203.0.113.10:443", nil
	})
	require.NoError(t, err)
	require.NotNil(t, transport.DialContext)

	conn, err := transport.DialContext(context.Background(), "tcp", "example.com:443")
	if conn != nil {
		_ = conn.Close()
	}
	require.NoError(t, err)
	require.True(t, resolverCalled)
	require.Equal(t, "203.0.113.10:443", <-targetCh)
	require.NoError(t, <-errCh)
}

func TestConfigureTransportProxyWithTargetResolver_RejectsResolverErrorBeforeSOCKSConnect(t *testing.T) {
	transport := &http.Transport{}
	proxyURL, err := url.Parse("socks5://127.0.0.1:1")
	require.NoError(t, err)

	err = ConfigureTransportProxyWithTargetResolver(transport, proxyURL, func(ctx context.Context, network, addr string) (string, error) {
		return "", errors.New("blocked private upstream ip")
	})
	require.NoError(t, err)

	conn, err := transport.DialContext(context.Background(), "tcp", "internal.example:443")
	if conn != nil {
		_ = conn.Close()
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "blocked private upstream ip")
}

func TestConfigureTransportProxyWithTargetResolver_RejectsHTTPProxy(t *testing.T) {
	transport := &http.Transport{}
	proxyURL, err := url.Parse("http://proxy.example.com:8080")
	require.NoError(t, err)

	err = ConfigureTransportProxyWithTargetResolver(transport, proxyURL, func(ctx context.Context, network, addr string) (string, error) {
		return addr, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot bind validated upstream IP")
}
