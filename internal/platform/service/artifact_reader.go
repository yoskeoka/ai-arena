package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArtifactReader loads persisted artifact bytes from stable locators.
type ArtifactReader interface {
	Read(context.Context, string) ([]byte, error)
	ReadBounded(context.Context, string, int64) ([]byte, error)
}

// DefaultArtifactReader reads local files, http(s) URLs, and optional S3 locators.
type DefaultArtifactReader struct {
	httpClient *http.Client
	s3         *S3ArtifactStore
}

// NewDefaultArtifactReader constructs the default artifact reader.
func NewDefaultArtifactReader(s3Store *S3ArtifactStore) *DefaultArtifactReader {
	return &DefaultArtifactReader{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		s3:         s3Store,
	}
}

// Read loads artifact bytes from the given locator.
func (r *DefaultArtifactReader) Read(ctx context.Context, locator string) ([]byte, error) {
	return r.ReadBounded(ctx, locator, 0)
}

// ReadBounded loads artifact bytes without materializing more than limit bytes.
// A non-positive limit leaves reads unbounded for existing operator-only callers.
func (r *DefaultArtifactReader) ReadBounded(ctx context.Context, locator string, limit int64) ([]byte, error) {
	locator = strings.TrimSpace(locator)
	if locator == "" {
		return nil, fmt.Errorf("service: artifact locator is required")
	}

	if isLocalPath(locator) {
		path := localPath(locator)
		// #nosec G304 -- the service reads the persisted local artifact selected by the caller.
		data, err := readFileBounded(path, limit)
		if err != nil {
			return nil, fmt.Errorf("service: read local artifact %s: %w", locator, err)
		}
		return data, nil
	}

	parsed, err := url.Parse(locator)
	if err != nil {
		return nil, fmt.Errorf("service: parse artifact locator %s: %w", locator, err)
	}

	switch parsed.Scheme {
	case "http", "https":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, locator, nil)
		if err != nil {
			return nil, fmt.Errorf("service: build GET %s: %w", locator, err)
		}
		resp, err := r.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("service: GET artifact %s: %w", locator, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("service: GET artifact %s: unexpected status %s", locator, resp.Status)
		}
		if limit > 0 && resp.ContentLength > limit {
			return nil, fmt.Errorf("service: artifact %s exceeds %d bytes", locator, limit)
		}
		data, err := readBounded(resp.Body, limit)
		if err != nil {
			return nil, fmt.Errorf("service: read artifact body %s: %w", locator, err)
		}
		return data, nil
	case "s3":
		if r.s3 == nil {
			return nil, fmt.Errorf("service: S3 artifact store is not configured for %s", locator)
		}
		return r.s3.ReadLocatorBounded(ctx, locator, limit)
	default:
		if filepath.IsAbs(locator) {
			// #nosec G304 -- the service reads the persisted local artifact selected by the caller.
			data, readErr := readFileBounded(filepath.Clean(locator), limit)
			if readErr != nil {
				return nil, fmt.Errorf("service: read local artifact %s: %w", locator, readErr)
			}
			return data, nil
		}
		return nil, fmt.Errorf("service: unsupported artifact locator scheme %q", parsed.Scheme)
	}
}

func readFileBounded(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readBounded(file, limit)
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(reader)
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("artifact exceeds %d bytes", limit)
	}
	return data, nil
}
