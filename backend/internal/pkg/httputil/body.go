package httputil

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

const (
	requestBodyReadInitCap    = 512
	requestBodyReadMaxInitCap = 1 << 20
	defaultDecodedBodyLimit   = 64 << 20
)

// ReadRequestBodyWithPrealloc reads request body with preallocated buffer based
// on content length, transparently decoding any Content-Encoding the upstream
// client used to compress the body (zstd, gzip, deflate, br).
func ReadRequestBodyWithPrealloc(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}

	effectiveLimit := RequestBodyLimit(req)

	capHint := requestBodyReadInitCap
	if req.ContentLength > 0 {
		switch {
		case req.ContentLength < int64(requestBodyReadInitCap):
			capHint = requestBodyReadInitCap
		case req.ContentLength > int64(requestBodyReadMaxInitCap):
			capHint = requestBodyReadMaxInitCap
		default:
			capHint = int(req.ContentLength)
		}
	}
	if effectiveLimit > 0 && int64(capHint) > effectiveLimit {
		capHint = int(effectiveLimit)
		if capHint < 1 {
			capHint = 1
		}
	}

	raw, err := readAllWithLimit(req.Body, capHint, effectiveLimit)
	if err != nil {
		return nil, err
	}

	enc := strings.ToLower(strings.TrimSpace(req.Header.Get("Content-Encoding")))
	if enc == "" || enc == "identity" {
		return raw, nil
	}

	decoded, err := decompressRequestBody(enc, raw, resolvedDecodedBodyLimit(effectiveLimit))
	if err != nil {
		return nil, fmt.Errorf("decode Content-Encoding %q: %w", enc, err)
	}

	req.Header.Del("Content-Encoding")
	req.Header.Del("Content-Length")
	req.ContentLength = int64(len(decoded))

	return decoded, nil
}

func decompressRequestBody(encoding string, raw []byte, limit int64) ([]byte, error) {
	switch encoding {
	case "zstd":
		dec, err := zstd.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		defer dec.Close()
		return readAllWithLimit(dec, decodedCapHint(len(raw), limit), limit)
	case "gzip", "x-gzip":
		gr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		defer func() { _ = gr.Close() }()
		return readAllWithLimit(gr, decodedCapHint(len(raw), limit), limit)
	case "deflate":
		zr, err := zlib.NewReader(bytes.NewReader(raw))
		if err == nil {
			defer func() { _ = zr.Close() }()
			return readAllWithLimit(zr, decodedCapHint(len(raw), limit), limit)
		}
		fr := flate.NewReader(bytes.NewReader(raw))
		defer func() { _ = fr.Close() }()
		return readAllWithLimit(fr, decodedCapHint(len(raw), limit), limit)
	case "br":
		return readAllWithLimit(brotli.NewReader(bytes.NewReader(raw)), decodedCapHint(len(raw), limit), limit)
	default:
		return nil, errors.New("unsupported Content-Encoding")
	}
}

func resolvedDecodedBodyLimit(limit int64) int64 {
	if limit > 0 {
		return limit
	}
	return defaultDecodedBodyLimit
}

func decodedCapHint(rawLen int, limit int64) int {
	capHint := requestBodyReadInitCap
	if rawLen > capHint {
		capHint = rawLen
		if capHint > requestBodyReadMaxInitCap {
			capHint = requestBodyReadMaxInitCap
		}
	}
	if limit > 0 && int64(capHint) > limit {
		capHint = int(limit)
		if capHint < 1 {
			capHint = 1
		}
	}
	return capHint
}

func readAllWithLimit(reader io.Reader, capHint int, limit int64) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, capHint))
	if limit > 0 {
		reader = io.LimitReader(reader, limitPlusSentinel(limit))
	}
	if _, err := io.Copy(buf, reader); err != nil {
		return nil, err
	}
	if limit > 0 && int64(buf.Len()) > limit {
		return nil, &http.MaxBytesError{Limit: limit}
	}
	return buf.Bytes(), nil
}

func limitPlusSentinel(limit int64) int64 {
	if limit >= math.MaxInt64 {
		return math.MaxInt64
	}
	return limit + 1
}
