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
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/service"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s", usageFor(""))
	}

	subcommand := args[0]
	if subcommand != "submit" && subcommand != "run-once" && subcommand != "submit-cancel" {
		return fmt.Errorf("usage: %s", usageFor(""))
	}

	fs := flag.NewFlagSet("arena-service "+subcommand, flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		submissionPath string
		baseDir        string
		workerID       string
		matchTimeout   time.Duration
	)
	fs.StringVar(&submissionPath, "submission", "", "submission JSON path or - for stdin")
	fs.StringVar(&baseDir, "base-dir", "", "base directory for resolving local artifact refs and output_dir")
	fs.StringVar(&workerID, "worker-id", "cli-worker", "worker identifier for run-once")
	fs.DurationVar(&matchTimeout, "match-timeout", 0, "match timeout for run-once")
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

	app, err := newCLIApp(baseDir, matchTimeout)
	if err != nil {
		return err
	}

	var record service.QueueRecord
	switch subcommand {
	case "submit":
		record, err = app.commands.Submit(context.Background(), submission)
	case "run-once":
		record, err = app.runOnce(context.Background(), submission, workerID)
	case "submit-cancel":
		record, err = app.submitCancel(context.Background(), submission)
	default:
		return fmt.Errorf("unsupported subcommand %q", subcommand)
	}
	if err != nil {
		if subcommand == "run-once" && record.Submission.SubmissionID != "" {
			if encodeErr := encodeRecord(stdout, record); encodeErr != nil {
				return encodeErr
			}
		}
		return err
	}
	return encodeRecord(stdout, record)
}

func usageFor(subcommand string) string {
	switch subcommand {
	case "run-once":
		return "arena-service run-once --submission <path-or-> [--base-dir <dir>] [--worker-id <id>] [--match-timeout <duration>]"
	case "submit-cancel":
		return "arena-service submit-cancel --submission <path-or-> [--base-dir <dir>]"
	case "submit":
		return "arena-service submit --submission <path-or-> [--base-dir <dir>]"
	default:
		return "arena-service <submit|run-once|submit-cancel> --submission <path-or-> [--base-dir <dir>]"
	}
}

type cliApp struct {
	commands *service.CommandService
	queue    service.QueueStore
	baseDir  string
	timeout  time.Duration
}

func newCLIApp(baseDir string, matchTimeout time.Duration) (*cliApp, error) {
	dryRun, err := service.NewLocalDryRunChecker(baseDir)
	if err != nil {
		return nil, err
	}
	validator, err := service.NewDefaultAdmissionValidator(nil, dryRun)
	if err != nil {
		return nil, err
	}
	store := service.NewInMemoryQueueStore()
	commands, err := service.NewCommandService(store, validator)
	if err != nil {
		return nil, err
	}
	return &cliApp{
		commands: commands,
		queue:    store,
		baseDir:  baseDir,
		timeout:  matchTimeout,
	}, nil
}

func (a *cliApp) runOnce(ctx context.Context, submission service.MatchSubmission, workerID string) (service.QueueRecord, error) {
	if _, err := a.commands.Submit(ctx, submission); err != nil {
		return service.QueueRecord{}, err
	}
	worker, err := a.newWorker()
	if err != nil {
		return service.QueueRecord{}, err
	}
	record, err := worker.ProcessNext(ctx, workerID)
	if err != nil {
		return record, err
	}
	return record, nil
}

func (a *cliApp) submitCancel(ctx context.Context, submission service.MatchSubmission) (service.QueueRecord, error) {
	record, err := a.commands.Submit(ctx, submission)
	if err != nil {
		return service.QueueRecord{}, err
	}
	return a.commands.Cancel(ctx, record.Submission.SubmissionID)
}

func (a *cliApp) newWorker() (*service.Worker, error) {
	invoker, err := service.NewLocalRunnerInvoker(a.baseDir, nil, a.timeout)
	if err != nil {
		return nil, err
	}
	return service.NewWorker(a.queue, invoker, service.LocalTerminalPersister{})
}

func encodeRecord(stdout io.Writer, record service.QueueRecord) error {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(record)
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
