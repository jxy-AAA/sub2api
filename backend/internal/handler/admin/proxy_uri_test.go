package admin

import "testing"

func TestParseProxyKeySupportsURIAndLegacyFormat(t *testing.T) {
	t.Run("uri", func(t *testing.T) {
		got, ok := parseProxyKey("http://first%20last%40corp:p%40%20ss%3A%23word@[2001:db8::1]:3128")
		if !ok {
			t.Fatal("expected uri proxy key to parse")
		}
		if got.Protocol != "http" || got.Host != "2001:db8::1" || got.Port != 3128 {
			t.Fatalf("unexpected target: %#v", got)
		}
		if got.Username != "first last@corp" || got.Password != "p@ ss:#word" {
			t.Fatalf("unexpected credentials: %#v", got)
		}
	})

	t.Run("legacy", func(t *testing.T) {
		got, ok := parseProxyKey("https|10.0.0.2|443|u|p")
		if !ok {
			t.Fatal("expected legacy proxy key to parse")
		}
		if got.Protocol != "https" || got.Host != "10.0.0.2" || got.Port != 443 || got.Username != "u" || got.Password != "p" {
			t.Fatalf("unexpected target: %#v", got)
		}
	})
}

func TestCanonicalizeDataProxyBuildsRoundTripURI(t *testing.T) {
	item := canonicalizeDataProxy(DataProxy{
		Protocol: "http",
		Host:     "[2001:db8::10]",
		Port:     8080,
		Username: "first last@corp",
		Password: "p@ ss:#word",
	})

	want := "http://first%20last%40corp:p%40%20ss%3A%23word@[2001:db8::10]:8080"
	if item.ProxyKey != want {
		t.Fatalf("ProxyKey = %q, want %q", item.ProxyKey, want)
	}
}
