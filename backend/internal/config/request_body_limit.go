package config

// EffectiveRequestBodyLimit returns the configured request body limit applied at
// the HTTP server boundary. When server.max_request_body_size is unset, it
// falls back to gateway.max_body_size.
func (c *Config) EffectiveRequestBodyLimit() int64 {
	if c == nil {
		return 0
	}
	if c.Server.MaxRequestBodySize > 0 {
		return c.Server.MaxRequestBodySize
	}
	return c.Gateway.MaxBodySize
}
