package config

import "testing"

func TestEffectiveRequestBodyLimit(t *testing.T) {
	t.Run("prefers server limit", func(t *testing.T) {
		cfg := &Config{}
		cfg.Server.MaxRequestBodySize = 1024
		cfg.Gateway.MaxBodySize = 2048

		if got := cfg.EffectiveRequestBodyLimit(); got != 1024 {
			t.Fatalf("EffectiveRequestBodyLimit() = %d, want 1024", got)
		}
	})

	t.Run("falls back to gateway limit", func(t *testing.T) {
		cfg := &Config{}
		cfg.Gateway.MaxBodySize = 2048

		if got := cfg.EffectiveRequestBodyLimit(); got != 2048 {
			t.Fatalf("EffectiveRequestBodyLimit() = %d, want 2048", got)
		}
	})
}
