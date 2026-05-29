package httpclient

import (
	"container/list"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/proxyurl"
	"github.com/Wei-Shaw/sub2api/internal/pkg/proxyutil"
)

const (
	defaultMaxIdleConns              = 100
	defaultMaxIdleConnsPerHost       = 10
	defaultIdleConnTimeout           = 90 * time.Second
	defaultDialTimeout               = 5 * time.Second
	defaultTLSHandshakeTimeout       = 5 * time.Second
	defaultSharedClientCacheTTL      = 15 * time.Minute
	defaultSharedClientCacheCapacity = 64
)

type Options struct {
	ProxyURL              string
	Timeout               time.Duration
	ResponseHeaderTimeout time.Duration
	InsecureSkipVerify    bool
	ValidateResolvedIP    bool
	AllowPrivateHosts     bool

	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
}

type lookupIPFunc func(ctx context.Context, network, host string) ([]net.IP, error)
type dialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

var resolveIPs lookupIPFunc = net.DefaultResolver.LookupIP

type idleConnectionCloser interface {
	CloseIdleConnections()
}

type sharedClientCacheEntry struct {
	key      string
	client   *http.Client
	lastUsed time.Time
}

type sharedClientCache struct {
	mu       sync.Mutex
	entries  map[string]*list.Element
	lru      *list.List
	capacity int
	ttl      time.Duration
	now      func() time.Time
}

func newSharedClientCache(capacity int, ttl time.Duration, now func() time.Time) *sharedClientCache {
	if capacity <= 0 {
		capacity = defaultSharedClientCacheCapacity
	}
	if ttl <= 0 {
		ttl = defaultSharedClientCacheTTL
	}
	if now == nil {
		now = time.Now
	}

	return &sharedClientCache{
		entries:  make(map[string]*list.Element, capacity),
		lru:      list.New(),
		capacity: capacity,
		ttl:      ttl,
		now:      now,
	}
}

func (c *sharedClientCache) get(key string) (*http.Client, bool) {
	if c == nil {
		return nil, false
	}

	now := c.now()
	var stale *http.Client

	c.mu.Lock()
	if elem := c.entries[key]; elem != nil {
		entry := elem.Value.(*sharedClientCacheEntry)
		if !c.isExpired(entry, now) {
			entry.lastUsed = now
			c.lru.MoveToFront(elem)
			client := entry.client
			c.mu.Unlock()
			return client, true
		}
		stale = c.removeElementLocked(elem)
	}
	c.mu.Unlock()

	closeIdleHTTPClient(stale)
	return nil, false
}

func (c *sharedClientCache) store(key string, client *http.Client) (*http.Client, bool, []*http.Client) {
	if c == nil {
		return client, true, nil
	}

	now := c.now()
	var evicted []*http.Client

	c.mu.Lock()
	if elem := c.entries[key]; elem != nil {
		entry := elem.Value.(*sharedClientCacheEntry)
		if !c.isExpired(entry, now) {
			entry.lastUsed = now
			c.lru.MoveToFront(elem)
			actual := entry.client
			c.mu.Unlock()
			return actual, false, nil
		}
		evicted = append(evicted, c.removeElementLocked(elem))
	}

	evicted = append(evicted, c.removeExpiredLocked(now)...)
	entry := &sharedClientCacheEntry{
		key:      key,
		client:   client,
		lastUsed: now,
	}
	c.entries[key] = c.lru.PushFront(entry)

	for len(c.entries) > c.capacity {
		evicted = append(evicted, c.removeOldestLocked())
	}

	c.mu.Unlock()
	return client, true, evicted
}

func (c *sharedClientCache) closeAll() []*http.Client {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	clients := make([]*http.Client, 0, len(c.entries))
	for _, elem := range c.entries {
		entry := elem.Value.(*sharedClientCacheEntry)
		clients = append(clients, entry.client)
	}
	c.entries = make(map[string]*list.Element, c.capacity)
	c.lru.Init()
	c.mu.Unlock()

	return clients
}

func (c *sharedClientCache) len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func (c *sharedClientCache) removeExpiredLocked(now time.Time) []*http.Client {
	var evicted []*http.Client
	for key, elem := range c.entries {
		entry := elem.Value.(*sharedClientCacheEntry)
		if c.isExpired(entry, now) {
			evicted = append(evicted, entry.client)
			delete(c.entries, key)
			c.lru.Remove(elem)
		}
	}
	return evicted
}

func (c *sharedClientCache) removeOldestLocked() *http.Client {
	elem := c.lru.Back()
	if elem == nil {
		return nil
	}
	return c.removeElementLocked(elem)
}

func (c *sharedClientCache) removeElementLocked(elem *list.Element) *http.Client {
	if elem == nil {
		return nil
	}
	entry := elem.Value.(*sharedClientCacheEntry)
	delete(c.entries, entry.key)
	c.lru.Remove(elem)
	return entry.client
}

func (c *sharedClientCache) isExpired(entry *sharedClientCacheEntry, now time.Time) bool {
	return c.ttl > 0 && now.Sub(entry.lastUsed) >= c.ttl
}

var sharedClients = newSharedClientCache(defaultSharedClientCacheCapacity, defaultSharedClientCacheTTL, time.Now)

func GetClient(opts Options) (*http.Client, error) {
	key := buildClientKey(opts)
	if cached, ok := sharedClients.get(key); ok {
		return cached, nil
	}

	client, err := buildClient(opts)
	if err != nil {
		return nil, err
	}

	actual, inserted, evicted := sharedClients.store(key, client)
	if !inserted && actual != client {
		evicted = append(evicted, client)
	}
	closeIdleHTTPClients(evicted)
	return actual, nil
}

func buildClient(opts Options) (*http.Client, error) {
	transport, err := buildTransport(opts)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
	}, nil
}

func buildTransport(opts Options) (*http.Transport, error) {
	maxIdleConns := opts.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = defaultMaxIdleConns
	}

	maxIdleConnsPerHost := opts.MaxIdleConnsPerHost
	if maxIdleConnsPerHost <= 0 {
		maxIdleConnsPerHost = defaultMaxIdleConnsPerHost
	}

	baseDialer := &net.Dialer{Timeout: defaultDialTimeout}
	dialContext := baseDialer.DialContext
	if opts.ValidateResolvedIP && !opts.AllowPrivateHosts {
		dialContext = buildValidatedDialContext(baseDialer.DialContext, resolveIPs)
	}

	transport := &http.Transport{
		DialContext:           dialContext,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		MaxConnsPerHost:       opts.MaxConnsPerHost,
		IdleConnTimeout:       defaultIdleConnTimeout,
		ResponseHeaderTimeout: opts.ResponseHeaderTimeout,
	}

	if opts.InsecureSkipVerify {
		return nil, fmt.Errorf("insecure_skip_verify is not allowed; install a trusted certificate instead")
	}

	_, parsed, err := proxyurl.Parse(opts.ProxyURL)
	if err != nil {
		return nil, err
	}
	if parsed == nil {
		return transport, nil
	}

	if err := proxyutil.ConfigureTransportProxy(transport, parsed); err != nil {
		return nil, err
	}

	return transport, nil
}

func buildClientKey(opts Options) string {
	return fmt.Sprintf("%s|%s|%s|%t|%t|%t|%d|%d|%d",
		strings.TrimSpace(opts.ProxyURL),
		opts.Timeout.String(),
		opts.ResponseHeaderTimeout.String(),
		opts.InsecureSkipVerify,
		opts.ValidateResolvedIP,
		opts.AllowPrivateHosts,
		opts.MaxIdleConns,
		opts.MaxIdleConnsPerHost,
		opts.MaxConnsPerHost,
	)
}

func CloseSharedClients() {
	closeIdleHTTPClients(sharedClients.closeAll())
}

func closeIdleHTTPClients(clients []*http.Client) {
	for _, client := range clients {
		closeIdleHTTPClient(client)
	}
}

func closeIdleHTTPClient(client *http.Client) {
	if client == nil {
		return
	}

	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	closer, ok := transport.(idleConnectionCloser)
	if !ok {
		return
	}
	closer.CloseIdleConnections()
}

func buildValidatedDialContext(dialFn dialContextFunc, resolver lookupIPFunc) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid dial address %q: %w", address, err)
		}
		host = strings.TrimSpace(host)
		if host == "" {
			return nil, fmt.Errorf("invalid dial address %q: empty host", address)
		}

		candidates := make([]net.IP, 0, 4)
		if parsedIP := net.ParseIP(host); parsedIP != nil {
			candidates = append(candidates, parsedIP)
		} else {
			ips, lookupErr := resolver(ctx, "ip", host)
			if lookupErr != nil {
				return nil, fmt.Errorf("dns resolution failed for %s: %w", host, lookupErr)
			}
			candidates = append(candidates, ips...)
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("no resolved ip for %s", host)
		}

		allowed := make([]net.IP, 0, len(candidates))
		for _, ip := range candidates {
			if ip == nil || isDisallowedDialIP(ip) {
				continue
			}
			allowed = append(allowed, ip)
		}
		if len(allowed) == 0 {
			return nil, fmt.Errorf("all resolved ips are not allowed for %s", host)
		}

		var lastErr error
		for _, ip := range allowed {
			target := net.JoinHostPort(ip.String(), port)
			conn, dialErr := dialFn(ctx, network, target)
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		return nil, fmt.Errorf("dial to validated ips failed for %s: %w", host, lastErr)
	}
}

func isDisallowedDialIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified()
}
