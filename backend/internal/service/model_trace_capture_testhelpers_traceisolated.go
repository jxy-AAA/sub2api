//go:build traceisolated

package service

func safeUpstreamURL(rawURL string) string {
	return rawURL
}
