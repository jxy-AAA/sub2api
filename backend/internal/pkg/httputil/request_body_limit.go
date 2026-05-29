package httputil

import (
	"context"
	"net/http"
)

type requestBodyLimitContextKey struct{}

// RequestBodyLimit returns the effective request body limit stored on req.
func RequestBodyLimit(req *http.Request) int64 {
	if req == nil {
		return 0
	}
	limit, _ := req.Context().Value(requestBodyLimitContextKey{}).(int64)
	return limit
}

// WithRequestBodyLimit stores limit on req context. If a smaller limit is
// already present, the smaller value wins.
func WithRequestBodyLimit(req *http.Request, limit int64) *http.Request {
	if req == nil || limit <= 0 {
		return req
	}

	effectiveLimit := limit
	if existingLimit := RequestBodyLimit(req); existingLimit > 0 && existingLimit < effectiveLimit {
		effectiveLimit = existingLimit
	}

	ctx := context.WithValue(req.Context(), requestBodyLimitContextKey{}, effectiveLimit)
	return req.WithContext(ctx)
}

// ApplyRequestBodyLimit stores limit on req and wraps req.Body with
// http.MaxBytesReader so downstream readers share the same effective cap.
func ApplyRequestBodyLimit(w http.ResponseWriter, req *http.Request, limit int64) *http.Request {
	req = WithRequestBodyLimit(req, limit)
	if req == nil || limit <= 0 || req.Body == nil || w == nil {
		return req
	}

	req.Body = http.MaxBytesReader(w, req.Body, RequestBodyLimit(req))
	return req
}

// WrapRequestBodyLimit applies the effective request body limit to every
// request handled by next.
func WrapRequestBodyLimit(next http.Handler, limit int64) http.Handler {
	if next == nil || limit <= 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		next.ServeHTTP(w, ApplyRequestBodyLimit(w, req, limit))
	})
}
