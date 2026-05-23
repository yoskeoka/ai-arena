package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunResolvesRelativeOutputDirAgainstBaseDir(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	baseDir := filepath.Clean(filepath.Join(cwd, "..", ".."))

	input := `{
  "submission_id": "sub-1",
  "match_id": "match-1",
  "game": {
    "game_id": "janken",
    "game_version": "2.1.0",
    "ruleset_version": "regular"
  },
  "players": [
    {
      "player_id": "p1",
      "artifact_ref": "./testdata/ai/janken/janken-rock-ai"
    }
  ],
  "output_dir": "arena-service-output",
  "attempt_count": 1
}`

	submissionPath := filepath.Join(t.TempDir(), "submission.json")
	if err := os.WriteFile(submissionPath, []byte(input), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"submit", "--submission", submissionPath, "--base-dir", baseDir}, &stdout, &stderr); err != nil {
		t.Fatalf("run() error = %v, stderr = %s", err, stderr.String())
	}

	var record struct {
		Submission struct {
			OutputDir string `json:"output_dir"`
		} `json:"submission"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &record); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	want := filepath.Join(baseDir, "arena-service-output")
	if record.Submission.OutputDir != want {
		t.Fatalf("output_dir = %q, want %q", record.Submission.OutputDir, want)
	}
}
