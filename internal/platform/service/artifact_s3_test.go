package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestS3ArtifactStorePutReadAndPresign(t *testing.T) {
	store, shutdown := newTestS3ArtifactStore(t)
	defer shutdown()

	ctx := context.Background()
	locator, err := store.PutBytes(ctx, "matches/match-1/result-summary.json", []byte(`{"match_id":"match-1"}`), "application/json")
	if err != nil {
		t.Fatalf("PutBytes() error = %v", err)
	}
	if locator != "s3://ai-arena-local/matches/match-1/result-summary.json" {
		t.Fatalf("locator = %q, want stable s3 locator", locator)
	}

	data, err := store.ReadLocator(ctx, locator)
	if err != nil {
		t.Fatalf("ReadLocator() error = %v", err)
	}
	if string(data) != `{"match_id":"match-1"}` {
		t.Fatalf("ReadLocator() = %s, want stored object", string(data))
	}

	detail := MatchDetail{ResultListItem: ResultListItem{ResultSummaryPath: locator}}
	issuer := NewS3ArtifactAccessIssuer(store)
	access, err := issuer.Issue(ctx, detail)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	entry := access["result-summary"]
	if entry.Status != "delegated" {
		t.Fatalf("entry.Status = %q, want delegated", entry.Status)
	}
	if entry.DownloadURL == "" || !strings.Contains(entry.DownloadURL, "X-Amz-Signature") {
		t.Fatalf("entry.DownloadURL = %q, want presigned URL", entry.DownloadURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.DownloadURL, nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(presigned) error = %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if string(body) != `{"match_id":"match-1"}` {
		t.Fatalf("presigned GET body = %s, want stored object", string(body))
	}
}

func TestDefaultArtifactReaderSupportsS3Locator(t *testing.T) {
	store, shutdown := newTestS3ArtifactStore(t)
	defer shutdown()

	ctx := context.Background()
	locator, err := store.PutBytes(ctx, "matches/match-2/result-summary.json", []byte(`{"match_id":"match-2"}`), "application/json")
	if err != nil {
		t.Fatalf("PutBytes() error = %v", err)
	}

	reader := NewDefaultArtifactReader(store)
	data, err := reader.Read(ctx, locator)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	var summary struct {
		MatchID string `json:"match_id"`
	}
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if summary.MatchID != "match-2" {
		t.Fatalf("summary.MatchID = %q, want match-2", summary.MatchID)
	}
}

func newTestS3ArtifactStore(t *testing.T) (*S3ArtifactStore, func()) {
	t.Helper()

	var (
		mu      sync.Mutex
		objects = map[string][]byte{}
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/")
		switch r.Method {
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("io.ReadAll(PUT) error = %v", err)
			}
			mu.Lock()
			objects[key] = body
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			mu.Lock()
			body, ok := objects[key]
			mu.Unlock()
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	store, err := NewS3ArtifactStore(context.Background(), S3ArtifactConfig{
		Bucket:          "ai-arena-local",
		Endpoint:        server.URL,
		AccessKeyID:     "admin",
		SecretAccessKey: "secret",
	})
	if err != nil {
		server.Close()
		t.Fatalf("NewS3ArtifactStore() error = %v", err)
	}
	return store, server.Close
}
