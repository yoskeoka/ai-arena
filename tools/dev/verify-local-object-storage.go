package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type detailResponse struct {
	SubmissionID      string `json:"submission_id"`
	ResultSummaryPath string `json:"result_summary_path"`
	ArtifactAccess    map[string]struct {
		Locator     string `json:"locator"`
		DownloadURL string `json:"download_url"`
		Status      string `json:"status"`
	} `json:"artifact_access"`
}

type resultSummary struct {
	MatchID string `json:"match_id"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	baseURL := strings.TrimRight(os.Getenv("ARENA_SERVICE_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:10000"
	}

	submissionID := fmt.Sprintf("sub-local-object-storage-%d", time.Now().UnixNano())
	matchID := fmt.Sprintf("match-local-object-storage-%d", time.Now().UnixNano())
	body := map[string]string{
		"preset_id":     "echo-reference",
		"submission_id": submissionID,
		"match_id":      matchID,
		"output_dir":    "arena-service-output",
	}
	if err := postJSON(ctx, baseURL+"/api/v1/preset-matches", body); err != nil {
		exitf("enqueue preset match: %v", err)
	}

	for ctx.Err() == nil {
		detail, err := fetchDetail(ctx, baseURL, submissionID)
		if err == nil && detail.SubmissionID == submissionID {
			access, ok := detail.ArtifactAccess["result-summary"]
			if detail.ResultSummaryPath == "" || !ok || access.DownloadURL == "" || access.Status != "delegated" {
				time.Sleep(1 * time.Second)
				continue
			}
			if !strings.HasPrefix(detail.ResultSummaryPath, "s3://") {
				exitf("result_summary_path = %q, want s3:// locator", detail.ResultSummaryPath)
			}
			summary, fetchErr := fetchResultSummary(ctx, access.DownloadURL)
			if fetchErr != nil {
				exitf("fetch delegated result-summary: %v", fetchErr)
			}
			if summary.MatchID != matchID {
				exitf("delegated result-summary match_id = %q, want %q", summary.MatchID, matchID)
			}
			fmt.Printf("verified local object storage lane: submission_id=%s locator=%s\n", submissionID, detail.ResultSummaryPath)
			return
		}
		time.Sleep(1 * time.Second)
	}

	exitf("timed out waiting for completed submission %s", submissionID)
}

func postJSON(ctx context.Context, endpoint string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	// #nosec G107,G704 -- local verification posts only to the repo-owned arena-service endpoint.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// #nosec G107,G704 -- local verification posts only to the repo-owned arena-service endpoint.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(payload)))
	}
	return nil
}

func fetchDetail(ctx context.Context, baseURL, submissionID string) (detailResponse, error) {
	// #nosec G107,G704 -- local verification reads only from the repo-owned arena-service endpoint.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/matches/"+submissionID, nil)
	if err != nil {
		return detailResponse{}, err
	}
	// #nosec G107,G704 -- local verification reads only from the repo-owned arena-service endpoint.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return detailResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return detailResponse{}, fmt.Errorf("submission not ready")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return detailResponse{}, fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(payload)))
	}
	var detail detailResponse
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return detailResponse{}, err
	}
	return detail, nil
}

func fetchResultSummary(ctx context.Context, downloadURL string) (resultSummary, error) {
	// #nosec G107,G704 -- delegated URLs come from the local arena-service under test.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return resultSummary{}, err
	}
	// #nosec G107,G704 -- delegated URLs come from the local arena-service under test.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resultSummary{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return resultSummary{}, fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(payload)))
	}
	var summary resultSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return resultSummary{}, err
	}
	return summary, nil
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
