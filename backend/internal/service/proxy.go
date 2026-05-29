package service

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Proxy struct {
	ID        int64
	Name      string
	Protocol  string
	Host      string
	Port      int
	Username  string
	Password  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *Proxy) IsActive() bool {
	return p.Status == StatusActive
}

func (p *Proxy) URL() string {
	u := &url.URL{
		Scheme: strings.TrimSpace(p.Protocol),
		Host:   net.JoinHostPort(NormalizeProxyHost(p.Host), strconv.Itoa(p.Port)),
	}
	if p.Username != "" && p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	return u.String()
}

func NormalizeProxyHost(host string) string {
	trimmed := strings.TrimSpace(host)
	if len(trimmed) >= 2 && strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return trimmed[1 : len(trimmed)-1]
	}
	return trimmed
}

func HasPartialProxyAuth(username, password string) bool {
	return (username == "") != (password == "")
}

func ProxyIdentityKey(protocol, host string, port int, username, password string) string {
	return strings.ToLower(strings.TrimSpace(protocol)) +
		"\x00" + strings.ToLower(NormalizeProxyHost(host)) +
		"\x00" + strconv.Itoa(port) +
		"\x00" + username +
		"\x00" + password
}

type ProxyWithAccountCount struct {
	Proxy
	AccountCount   int64
	LatencyMs      *int64
	LatencyStatus  string
	LatencyMessage string
	IPAddress      string
	Country        string
	CountryCode    string
	Region         string
	City           string
	QualityStatus  string
	QualityScore   *int
	QualityGrade   string
	QualitySummary string
	QualityChecked *int64
}

type ProxyAccountSummary struct {
	ID       int64
	Name     string
	Platform string
	Type     string
	Notes    *string
}
