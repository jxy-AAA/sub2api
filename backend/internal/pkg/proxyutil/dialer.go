// Package proxyutil 提供统一的代理配置功能
//
// 支持的代理协议：
//   - HTTP/HTTPS: 通过 Transport.Proxy 设置
//   - SOCKS5: 通过 Transport.DialContext 设置（客户端本地解析 DNS）
//   - SOCKS5H: 通过 Transport.DialContext 设置（代理端远程解析 DNS，推荐）
//
// 注意：proxyurl.Parse() 会自动将 socks5:// 升级为 socks5h://，
// 确保 DNS 也由代理端解析，防止 DNS 泄漏。
package proxyutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/proxy"
)

// ConfigureTransportProxy 根据代理 URL 配置 Transport
//
// 支持的协议：
//   - http/https: 设置 transport.Proxy
//   - socks5: 设置 transport.DialContext（客户端本地解析 DNS）
//   - socks5h: 设置 transport.DialContext（代理端远程解析 DNS，推荐）
//
// 参数：
//   - transport: 需要配置的 http.Transport
//   - proxyURL: 代理地址，nil 表示直连
//
// 返回：
//   - error: 代理配置错误（协议不支持或 dialer 创建失败）
type TargetResolver func(ctx context.Context, network, addr string) (string, error)

func ConfigureTransportProxy(transport *http.Transport, proxyURL *url.URL) error {
	return ConfigureTransportProxyWithTargetResolver(transport, proxyURL, nil)
}

func ConfigureTransportProxyWithTargetResolver(transport *http.Transport, proxyURL *url.URL, targetResolver TargetResolver) error {
	if proxyURL == nil {
		return nil
	}

	scheme := strings.ToLower(proxyURL.Scheme)
	switch scheme {
	case "http", "https":
		if targetResolver != nil {
			return fmt.Errorf("%s proxy cannot bind validated upstream IP", scheme)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		return nil

	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return fmt.Errorf("create socks5 dialer: %w", err)
		}
		// ?????? context ? DialContext???????????
		if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialAddr := addr
				if targetResolver != nil {
					resolvedAddr, err := targetResolver(ctx, network, addr)
					if err != nil {
						return nil, err
					}
					dialAddr = resolvedAddr
				}
				return contextDialer.DialContext(ctx, network, dialAddr)
			}
		} else {
			// ??????? dialer ??? ContextDialer???????? DialContext
			// ??????????????????
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialAddr := addr
				if targetResolver != nil {
					resolvedAddr, err := targetResolver(ctx, network, addr)
					if err != nil {
						return nil, err
					}
					dialAddr = resolvedAddr
				}
				return dialer.Dial(network, dialAddr)
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported proxy scheme: %s", scheme)
	}
}
