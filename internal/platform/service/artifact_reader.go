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
	locator = strings.TrimSpace(locator)
	if locator == "" {
		return nil, fmt.Errorf("service: artifact locator is required")
	}

	if isLocalPath(locator) {
		path := localPath(locator)
		// #nosec G304 -- the service reads the persisted local artifact selected by the caller.
		data, err := os.ReadFile(path)
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
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("service: read artifact body %s: %w", locator, err)
		}
		return data, nil
	case "s3":
		if r.s3 == nil {
			return nil, fmt.Errorf("service: S3 artifact store is not configured for %s", locator)
		}
		return r.s3.ReadLocator(ctx, locator)
	default:
		if filepath.IsAbs(locator) {
			// #nosec G304 -- the service reads the persisted local artifact selected by the caller.
			data, readErr := os.ReadFile(filepath.Clean(locator))
			if readErr != nil {
				return nil, fmt.Errorf("service: read local artifact %s: %w", locator, readErr)
			}
			return data, nil
		}
		return nil, fmt.Errorf("service: unsupported artifact locator scheme %q", parsed.Scheme)
	}
}
