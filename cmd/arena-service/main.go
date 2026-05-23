// Command arena-service validates and queues one match submission via the service skeleton.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/yoskeoka/ai-arena/internal/platform/service"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "submit" {
		return fmt.Errorf("usage: arena-service submit --submission <path-or-> [--base-dir <dir>]")
	}

	fs := flag.NewFlagSet("arena-service submit", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		submissionPath string
		baseDir        string
	)
	fs.StringVar(&submissionPath, "submission", "", "submission JSON path or - for stdin")
	fs.StringVar(&baseDir, "base-dir", "", "base directory for resolving local artifact refs and output_dir")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if submissionPath == "" {
		return fmt.Errorf("--submission is required")
	}
	if baseDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		baseDir = cwd
	}
	baseDir = filepath.Clean(baseDir)

	submission, err := loadSubmission(submissionPath, os.Stdin)
	if err != nil {
		return err
	}
	resolveOutputDir(baseDir, &submission)

	dryRun, err := service.NewLocalDryRunChecker(baseDir)
	if err != nil {
		return err
	}
	validator, err := service.NewDefaultAdmissionValidator(nil, dryRun)
	if err != nil {
		return err
	}
	commands, err := service.NewCommandService(service.NewInMemoryQueueStore(), validator)
	if err != nil {
		return err
	}
	record, err := commands.Submit(context.Background(), submission)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(record)
}

func loadSubmission(path string, stdin io.Reader) (service.MatchSubmission, error) {
	var reader io.Reader
	switch path {
	case "-":
		reader = stdin
	default:
		// #nosec G304 -- the operator explicitly chooses the local submission file.
		file, err := os.Open(path)
		if err != nil {
			return service.MatchSubmission{}, err
		}
		defer file.Close()
		reader = file
	}

	var submission service.MatchSubmission
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&submission); err != nil {
		return service.MatchSubmission{}, fmt.Errorf("decode submission: %w", err)
	}
	return submission, nil
}

func resolveOutputDir(baseDir string, submission *service.MatchSubmission) {
	if submission == nil || submission.OutputDir == "" {
		return
	}
	parsed, err := url.Parse(submission.OutputDir)
	if err == nil && parsed.Scheme != "" {
		return
	}
	if filepath.IsAbs(submission.OutputDir) {
		submission.OutputDir = filepath.Clean(submission.OutputDir)
		return
	}
	submission.OutputDir = filepath.Join(baseDir, filepath.Clean(submission.OutputDir))
}
