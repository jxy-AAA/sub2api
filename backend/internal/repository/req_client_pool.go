package repository

import (
	"container/list"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/proxyurl"
	"github.com/imroc/req/v3"
)

const (
	defaultReqClientCacheTTL      = 15 * time.Minute
	defaultReqClientCacheCapacity = 64
)

type reqClientOptions struct {
	ProxyURL    string
	Timeout     time.Duration
	Impersonate bool
	ForceHTTP2  bool
}

type sharedReqClientCacheEntry struct {
	key      string
	client   *req.Client
	lastUsed time.Time
}

type sharedReqClientCache struct {
	mu       sync.Mutex
	entries  map[string]*list.Element
	lru      *list.List
	capacity int
	ttl      time.Duration
	now      func() time.Time
}

func newSharedReqClientCache(capacity int, ttl time.Duration, now func() time.Time) *sharedReqClientCache {
	if capacity <= 0 {
		capacity = defaultReqClientCacheCapacity
	}
	if ttl <= 0 {
		ttl = defaultReqClientCacheTTL
	}
	if now == nil {
		now = time.Now
	}

	return &sharedReqClientCache{
		entries:  make(map[string]*list.Element, capacity),
		lru:      list.New(),
		capacity: capacity,
		ttl:      ttl,
		now:      now,
	}
}

func (c *sharedReqClientCache) get(key string) (*req.Client, bool) {
	if c == nil {
		return nil, false
	}

	now := c.now()
	var stale *req.Client

	c.mu.Lock()
	if elem := c.entries[key]; elem != nil {
		entry := elem.Value.(*sharedReqClientCacheEntry)
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

	closeIdleReqClient(stale)
	return nil, false
}

func (c *sharedReqClientCache) store(key string, client *req.Client) (*req.Client, bool, []*req.Client) {
	if c == nil {
		return client, true, nil
	}

	now := c.now()
	var evicted []*req.Client

	c.mu.Lock()
	if elem := c.entries[key]; elem != nil {
		entry := elem.Value.(*sharedReqClientCacheEntry)
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
	entry := &sharedReqClientCacheEntry{
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

func (c *sharedReqClientCache) closeAll() []*req.Client {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	clients := make([]*req.Client, 0, len(c.entries))
	for _, elem := range c.entries {
		entry := elem.Value.(*sharedReqClientCacheEntry)
		clients = append(clients, entry.client)
	}
	c.entries = make(map[string]*list.Element, c.capacity)
	c.lru.Init()
	c.mu.Unlock()

	return clients
}

func (c *sharedReqClientCache) len() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func (c *sharedReqClientCache) removeExpiredLocked(now time.Time) []*req.Client {
	var evicted []*req.Client
	for key, elem := range c.entries {
		entry := elem.Value.(*sharedReqClientCacheEntry)
		if c.isExpired(entry, now) {
			evicted = append(evicted, entry.client)
			delete(c.entries, key)
			c.lru.Remove(elem)
		}
	}
	return evicted
}

func (c *sharedReqClientCache) removeOldestLocked() *req.Client {
	elem := c.lru.Back()
	if elem == nil {
		return nil
	}
	return c.removeElementLocked(elem)
}

func (c *sharedReqClientCache) removeElementLocked(elem *list.Element) *req.Client {
	if elem == nil {
		return nil
	}
	entry := elem.Value.(*sharedReqClientCacheEntry)
	delete(c.entries, entry.key)
	c.lru.Remove(elem)
	return entry.client
}

func (c *sharedReqClientCache) isExpired(entry *sharedReqClientCacheEntry, now time.Time) bool {
	return c.ttl > 0 && now.Sub(entry.lastUsed) >= c.ttl
}

var sharedReqClients = newSharedReqClientCache(defaultReqClientCacheCapacity, defaultReqClientCacheTTL, time.Now)

func getSharedReqClient(opts reqClientOptions) (*req.Client, error) {
	key := buildReqClientKey(opts)
	if cached, ok := sharedReqClients.get(key); ok {
		return cached, nil
	}

	client := req.C().SetTimeout(opts.Timeout)
	if opts.ForceHTTP2 {
		client = client.EnableForceHTTP2()
	}
	if opts.Impersonate {
		client = client.ImpersonateChrome()
	}

	trimmed, _, err := proxyurl.Parse(opts.ProxyURL)
	if err != nil {
		return nil, err
	}
	if trimmed != "" {
		client.SetProxyURL(trimmed)
	}

	actual, inserted, evicted := sharedReqClients.store(key, client)
	if !inserted && actual != client {
		evicted = append(evicted, client)
	}
	closeIdleReqClients(evicted)
	return actual, nil
}

func buildReqClientKey(opts reqClientOptions) string {
	return fmt.Sprintf("%s|%s|%t|%t",
		strings.TrimSpace(opts.ProxyURL),
		opts.Timeout.String(),
		opts.Impersonate,
		opts.ForceHTTP2,
	)
}

func CloseSharedReqClients() {
	closeIdleReqClients(sharedReqClients.closeAll())
}

func closeIdleReqClients(clients []*req.Client) {
	for _, client := range clients {
		closeIdleReqClient(client)
	}
}

func closeIdleReqClient(client *req.Client) {
	if client == nil {
		return
	}

	transport := client.GetTransport()
	if transport == nil {
		return
	}
	transport.CloseIdleConnections()
}

// CreatePrivacyReqClient creates an HTTP client for OpenAI privacy settings API.
func CreatePrivacyReqClient(proxyURL string) (*req.Client, error) {
	return getSharedReqClient(reqClientOptions{
		ProxyURL:    proxyURL,
		Timeout:     30 * time.Second,
		Impersonate: true,
	})
}
