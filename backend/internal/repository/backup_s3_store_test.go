package repository

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

type recordingReadCloser struct {
	reader     io.Reader
	closeCount int
}

func (r *recordingReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *recordingReadCloser) Close() error {
	r.closeCount++
	return nil
}

type captureRoundTripper struct {
	t             *testing.T
	expectedBody  []byte
	contentType   string
	contentLength int64
}

func (rt *captureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.contentType = req.Header.Get("Content-Type")
	rt.contentLength = req.ContentLength

	body, err := io.ReadAll(req.Body)
	require.NoError(rt.t, err)
	require.Equal(rt.t, rt.expectedBody, body)

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func TestS3BackupStoreUpload_StreamsViaTempFile(t *testing.T) {
	payload := bytes.Repeat([]byte("backup-data-"), 1024)
	input := &recordingReadCloser{reader: bytes.NewReader(payload)}
	transport := &captureRoundTripper{t: t, expectedBody: payload}

	client := s3.New(s3.Options{
		Region:       "auto",
		Credentials:  aws.AnonymousCredentials{},
		BaseEndpoint: aws.String("https://example.com"),
		HTTPClient:   &http.Client{Transport: transport},
		UsePathStyle: true,
	})
	client = s3.New(client.Options(), func(o *s3.Options) {
		o.APIOptions = append(o.APIOptions, v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware)
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})

	store := &S3BackupStore{client: client, bucket: "test-bucket"}
	size, err := store.Upload(context.Background(), "backup.sql.gz", input, "application/gzip")
	require.NoError(t, err)
	require.Equal(t, int64(len(payload)), size)
	require.Equal(t, 1, input.closeCount)
	require.Equal(t, "application/gzip", transport.contentType)
	require.Equal(t, int64(len(payload)), transport.contentLength)
}
