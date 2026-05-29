package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http/httpguts"
)

type readDeadlineConn interface {
	SetReadDeadline(time.Time) error
}

type deadlineResetBody struct {
	inner   io.ReadCloser
	conn    readDeadlineConn
	cleared bool
}

type connContextKey struct{}

func withConnContext(ctxKey any) func(context.Context, net.Conn) context.Context {
	return func(ctx context.Context, conn net.Conn) context.Context {
		return context.WithValue(ctx, ctxKey, conn)
	}
}

func wrapRequestBodyReadTimeout(next http.Handler, timeout time.Duration) http.Handler {
	if next == nil || timeout <= 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req == nil || req.Body == nil || req.ContentLength == 0 && req.Header.Get("Transfer-Encoding") == "" {
			next.ServeHTTP(w, req)
			return
		}
		if isUpgradeRequest(req) {
			next.ServeHTTP(w, req)
			return
		}
		if conn, _ := req.Context().Value(connContextKey{}).(net.Conn); conn != nil {
			if deadlineConn, ok := conn.(readDeadlineConn); ok {
				_ = deadlineConn.SetReadDeadline(time.Now().Add(timeout))
				req.Body = &deadlineResetBody{
					inner: req.Body,
					conn:  deadlineConn,
				}
			}
		}
		next.ServeHTTP(w, req)
	})
}

func isUpgradeRequest(req *http.Request) bool {
	if req == nil {
		return false
	}
	return httpguts.HeaderValuesContainsToken(req.Header["Connection"], "upgrade") ||
		httpguts.HeaderValuesContainsToken(req.Header["Upgrade"], "websocket")
}

func (b *deadlineResetBody) Read(p []byte) (int, error) {
	if b == nil || b.inner == nil {
		return 0, io.EOF
	}
	n, err := b.inner.Read(p)
	if err == io.EOF {
		b.clearDeadline()
	}
	return n, err
}

func (b *deadlineResetBody) Close() error {
	if b == nil || b.inner == nil {
		return nil
	}
	b.clearDeadline()
	return b.inner.Close()
}

func (b *deadlineResetBody) clearDeadline() {
	if b == nil || b.cleared || b.conn == nil {
		return
	}
	b.cleared = true
	_ = b.conn.SetReadDeadline(time.Time{})
}
