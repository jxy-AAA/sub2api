package admin

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func buildProxyKey(protocol, host string, port int, username, password string) string {
	return buildProxyURI(protocol, host, port, username, password)
}

func buildProxyURI(protocol, host string, port int, username, password string) string {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	host = service.NormalizeProxyHost(host)
	u := &url.URL{
		Scheme: protocol,
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
	}
	switch {
	case username != "" && password == "":
		u.User = url.User(username)
	case username != "" || password != "":
		u.User = url.UserPassword(username, password)
	}
	return u.String()
}

func parseProxyKey(raw string) (DataProxy, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DataProxy{}, false
	}

	if strings.Contains(trimmed, "://") {
		u, err := url.Parse(trimmed)
		if err != nil {
			return DataProxy{}, false
		}
		protocol := strings.ToLower(strings.TrimSpace(u.Scheme))
		switch protocol {
		case "http", "https", "socks5", "socks5h":
		default:
			return DataProxy{}, false
		}
		port, err := strconv.Atoi(u.Port())
		if err != nil || port <= 0 || port > 65535 {
			return DataProxy{}, false
		}
		username := ""
		password := ""
		if u.User != nil {
			username, _ = url.QueryUnescape(u.User.Username())
			passwordValue, ok := u.User.Password()
			if ok {
				password, _ = url.QueryUnescape(passwordValue)
			}
		}
		return DataProxy{
			ProxyKey: trimmed,
			Protocol: protocol,
			Host:     service.NormalizeProxyHost(u.Hostname()),
			Port:     port,
			Username: username,
			Password: password,
		}, true
	}

	parts := strings.Split(trimmed, "|")
	if len(parts) != 5 {
		return DataProxy{}, false
	}
	port, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return DataProxy{}, false
	}
	return DataProxy{
		ProxyKey: trimmed,
		Protocol: strings.TrimSpace(parts[0]),
		Host:     service.NormalizeProxyHost(parts[1]),
		Port:     port,
		Username: strings.TrimSpace(parts[3]),
		Password: strings.TrimSpace(parts[4]),
	}, true
}

func canonicalizeDataProxy(item DataProxy) DataProxy {
	if parsed, ok := parseProxyKey(item.ProxyKey); ok {
		if strings.TrimSpace(item.Protocol) == "" {
			item.Protocol = parsed.Protocol
		}
		if strings.TrimSpace(item.Host) == "" {
			item.Host = parsed.Host
		}
		if item.Port == 0 {
			item.Port = parsed.Port
		}
		if item.Username == "" {
			item.Username = parsed.Username
		}
		if item.Password == "" {
			item.Password = parsed.Password
		}
	}
	item.Protocol = strings.ToLower(strings.TrimSpace(item.Protocol))
	item.Host = service.NormalizeProxyHost(item.Host)
	item.Username = strings.TrimSpace(item.Username)
	item.Password = strings.TrimSpace(item.Password)
	item.ProxyKey = buildProxyURI(item.Protocol, item.Host, item.Port, item.Username, item.Password)
	return item
}

func formatProxyKeyNotFound(proxyKey string) string {
	return fmt.Sprintf("proxy_key not found: %s", proxyKey)
}
