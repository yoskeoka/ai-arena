// Command arena-service validates and queues one match submission via the service skeleton.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
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
	return runWithFactory(args, stdout, stderr, newCLIApp)
}

type cliAppFactory func(baseDir string, matchTimeout time.Duration, postgresDSN string, artifactRuntime artifactRuntimeConfig) (*cliApp, error)

func runWithFactory(args []string, stdout, stderr io.Writer, factory cliAppFactory) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s", usageFor(""))
	}

	subcommand := args[0]
	if subcommand != "submit" && subcommand != "run-once" && subcommand != "submit-cancel" && subcommand != "list" && subcommand != "get" && subcommand != "read" && subcommand != "serve" {
		return fmt.Errorf("usage: %s", usageFor(""))
	}

	fs := flag.NewFlagSet("arena-service "+subcommand, flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		submissionPath string
		submissionID   string
		artifactKind   string
		baseDir        string
		workerID       string
		listenAddr     string
		presetConfig   string
		pollInterval   time.Duration
		matchTimeout   time.Duration
		postgresDSN    string
	)
	fs.StringVar(&submissionPath, "submission", "", "submission JSON path or - for stdin")
	fs.StringVar(&submissionID, "submission-id", "", "submission id for get/read")
	fs.StringVar(&artifactKind, "artifact", "", "artifact selector for read")
	fs.StringVar(&baseDir, "base-dir", "", "base directory for resolving local artifact refs and output_dir")
	fs.StringVar(&workerID, "worker-id", "cli-worker", "worker identifier for run-once")
	fs.StringVar(&listenAddr, "listen-addr", ":8080", "listen address for serve")
	fs.StringVar(&presetConfig, "preset-config", "", "preset config JSON path for serve")
	fs.DurationVar(&pollInterval, "worker-poll-interval", 2*time.Second, "poll interval for serve worker loop")
	fs.DurationVar(&matchTimeout, "match-timeout", 0, "match timeout for run-once")
	fs.StringVar(&postgresDSN, "postgres-dsn", "", "PostgreSQL DSN for durable queue storage")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if baseDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		baseDir = cwd
	}
	baseDir = filepath.Clean(baseDir)
	if postgresDSN == "" {
		postgresDSN = strings.TrimSpace(os.Getenv("ARENA_SERVICE_POSTGRES_DSN"))
	}
	artifactRuntime, err := loadArtifactRuntimeFromEnv()
	if err != nil {
		return err
	}

	app, err := factory(baseDir, matchTimeout, postgresDSN, artifactRuntime)
	if err != nil {
		return err
	}
	defer app.close()

	if subcommand == "serve" {
		if presetConfig == "" {
			presetConfig = strings.TrimSpace(os.Getenv("ARENA_SERVICE_PRESET_CONFIG"))
		}
		serveCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return app.serve(serveCtx, listenAddr, presetConfig, workerID, pollInterval, stderr)
	}

	switch subcommand {
	case "list":
		return app.list(context.Background(), stdout)
	case "get":
		if submissionID == "" {
			return fmt.Errorf("--submission-id is required")
		}
		return app.get(context.Background(), submissionID, stdout)
	case "read":
		if submissionID == "" {
			return fmt.Errorf("--submission-id is required")
		}
		if artifactKind == "" {
			return fmt.Errorf("--artifact is required")
		}
		return app.read(context.Background(), submissionID, artifactKind, stdout)
	}

	if submissionPath == "" {
		return fmt.Errorf("--submission is required")
	}
	submission, err := loadSubmission(submissionPath, os.Stdin)
	if err != nil {
		return err
	}
	resolveOutputDir(baseDir, artifactRuntime.usesOpaqueOutputDir(), &submission)

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
		return "arena-service run-once --submission <path-or-> [--base-dir <dir>] [--worker-id <id>] [--match-timeout <duration>] [--postgres-dsn <dsn>]"
	case "submit-cancel":
		return "arena-service submit-cancel --submission <path-or-> [--base-dir <dir>] [--postgres-dsn <dsn>]"
	case "submit":
		return "arena-service submit --submission <path-or-> [--base-dir <dir>] [--postgres-dsn <dsn>]"
	case "list":
		return "arena-service list [--base-dir <dir>] [--postgres-dsn <dsn>]"
	case "get":
		return "arena-service get --submission-id <id> [--base-dir <dir>] [--postgres-dsn <dsn>]"
	case "read":
		return "arena-service read --submission-id <id> --artifact <result-summary|record|snapshot|history|exported-snapshot|stderr:<player-id>> [--base-dir <dir>] [--postgres-dsn <dsn>]"
	case "serve":
		return "arena-service serve [--listen-addr <addr>] [--preset-config <path>] [--worker-id <id>] [--worker-poll-interval <duration>] [--base-dir <dir>] [--match-timeout <duration>] [--postgres-dsn <dsn>]"
	default:
		return "arena-service <submit|run-once|submit-cancel|list|get|read|serve> ..."
	}
}

type cliApp struct {
	commands       *service.CommandService
	queries        *service.QueryService
	queue          service.QueueStore
	reader         service.ArtifactReader
	artifactAccess service.ArtifactAccessIssuer
	persister      service.TerminalPersister
	baseDir        string
	timeout        time.Duration
	closeFn        func()
}

type artifactRuntimeConfig struct {
	backend         string
	bucket          string
	endpoint        string
	accessKeyID     string
	secretAccessKey string
}

func (c artifactRuntimeConfig) usesOpaqueOutputDir() bool {
	return c.backend == "r2"
}

func newCLIApp(baseDir string, matchTimeout time.Duration, postgresDSN string, artifactRuntime artifactRuntimeConfig) (*cliApp, error) {
	dryRun, err := service.NewLocalDryRunChecker(baseDir)
	if err != nil {
		return nil, err
	}
	validator, err := service.NewDefaultAdmissionValidator(nil, dryRun)
	if err != nil {
		return nil, err
	}
	store, closeFn, err := newQueueStore(postgresDSN)
	if err != nil {
		return nil, err
	}
	commands, err := service.NewCommandService(store, validator)
	if err != nil {
		closeFn()
		return nil, err
	}
	reader, artifactAccess, persister, err := newArtifactRuntime(context.Background(), artifactRuntime)
	if err != nil {
		closeFn()
		return nil, err
	}
	queries, err := service.NewQueryService(store, reader)
	if err != nil {
		closeFn()
		return nil, err
	}
	return &cliApp{
		commands:       commands,
		queries:        queries,
		queue:          store,
		reader:         reader,
		artifactAccess: artifactAccess,
		persister:      persister,
		baseDir:        baseDir,
		timeout:        matchTimeout,
		closeFn:        closeFn,
	}, nil
}

func newQueueStore(postgresDSN string) (service.QueueStore, func(), error) {
	if strings.TrimSpace(postgresDSN) == "" {
		return service.NewInMemoryQueueStore(), func() {}, nil
	}

	store, err := service.NewPostgresQueueStore(context.Background(), postgresDSN)
	if err != nil {
		return nil, nil, err
	}
	return store, store.Close, nil
}

func (a *cliApp) close() {
	if a == nil || a.closeFn == nil {
		return
	}
	a.closeFn()
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

func (a *cliApp) list(ctx context.Context, stdout io.Writer) error {
	items, err := a.queries.List(ctx)
	if err != nil {
		return err
	}
	return encodeJSON(stdout, items)
}

func (a *cliApp) get(ctx context.Context, submissionID string, stdout io.Writer) error {
	detail, err := a.queries.Get(ctx, submissionID)
	if err != nil {
		return err
	}
	return encodeJSON(stdout, detail)
}

func (a *cliApp) read(ctx context.Context, submissionID string, artifactKind string, stdout io.Writer) error {
	detail, err := a.queries.Get(ctx, submissionID)
	if err != nil {
		return err
	}
	path, err := selectArtifactPath(detail, artifactKind)
	if err != nil {
		return err
	}
	data, err := a.reader.Read(ctx, path)
	if err != nil {
		return err
	}
	_, err = stdout.Write(data)
	return err
}

func (a *cliApp) newWorker() (*service.Worker, error) {
	invoker, err := service.NewLocalRunnerInvoker(a.baseDir, nil, a.timeout)
	if err != nil {
		return nil, err
	}
	return service.NewWorker(a.queue, invoker, a.persister)
}

func (a *cliApp) serve(ctx context.Context, listenAddr string, presetConfig string, workerID string, pollInterval time.Duration, stderr io.Writer) error {
	presets, err := service.LoadPresetCatalog(resolveBaseDirPath(a.baseDir, presetConfig))
	if err != nil {
		return err
	}
	worker, err := a.newWorker()
	if err != nil {
		return err
	}
	api, err := service.NewOperatorAPI(a.commands, a.queries, resolvingPresetCatalog{
		baseDir: a.baseDir,
		opaque:  isOpaqueArtifactBackend(a.persister),
		next:    presets,
	}, a.artifactAccess)
	if err != nil {
		return err
	}
	logger := log.New(stderr, "arena-service: ", log.LstdFlags)
	loop, err := service.NewWorkerLoop(worker, workerID, pollInterval, func(err error) {
		logger.Printf("worker loop error: %v", err)
	})
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              listenAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	loopErrCh := make(chan error, 1)
	go func() {
		loopErrCh <- loop.Run(ctx)
	}()

	serverErrCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	select {
	case err := <-loopErrCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		return err
	case err := <-serverErrCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return <-loopErrCh
}

type resolvingPresetCatalog struct {
	baseDir string
	opaque  bool
	next    service.PresetCatalog
}

func (c resolvingPresetCatalog) Build(ctx context.Context, req service.PresetMatchRequest) (service.MatchSubmission, error) {
	submission, err := c.next.Build(ctx, req)
	if err != nil {
		return service.MatchSubmission{}, err
	}
	resolveOutputDir(c.baseDir, c.opaque, &submission)
	return submission, nil
}

func encodeRecord(stdout io.Writer, record service.QueueRecord) error {
	return encodeJSON(stdout, record)
}

func encodeJSON(stdout io.Writer, value any) error {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func resolveOutputDir(baseDir string, opaque bool, submission *service.MatchSubmission) {
	if submission == nil || submission.OutputDir == "" {
		return
	}
	if opaque {
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

func resolveBaseDirPath(baseDir, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme != "" {
		return value
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Join(baseDir, filepath.Clean(value))
}

func loadArtifactRuntimeFromEnv() (artifactRuntimeConfig, error) {
	cfg := artifactRuntimeConfig{
		backend:         strings.TrimSpace(os.Getenv("ARENA_SERVICE_ARTIFACT_BACKEND")),
		bucket:          strings.TrimSpace(os.Getenv("ARENA_SERVICE_ARTIFACT_R2_BUCKET")),
		endpoint:        strings.TrimSpace(os.Getenv("ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT")),
		accessKeyID:     strings.TrimSpace(os.Getenv("ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID")),
		secretAccessKey: strings.TrimSpace(os.Getenv("ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY")),
	}
	if cfg.backend == "" {
		cfg.backend = "filesystem"
	}
	switch cfg.backend {
	case "filesystem":
		return cfg, nil
	case "r2":
		if cfg.bucket == "" || cfg.endpoint == "" || cfg.accessKeyID == "" || cfg.secretAccessKey == "" {
			return artifactRuntimeConfig{}, fmt.Errorf("ARENA_SERVICE_ARTIFACT_R2_BUCKET, ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT, ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID, and ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY are required when ARENA_SERVICE_ARTIFACT_BACKEND=r2")
		}
		return cfg, nil
	default:
		return artifactRuntimeConfig{}, fmt.Errorf("unsupported ARENA_SERVICE_ARTIFACT_BACKEND %q", cfg.backend)
	}
}

func newArtifactRuntime(ctx context.Context, cfg artifactRuntimeConfig) (service.ArtifactReader, service.ArtifactAccessIssuer, service.TerminalPersister, error) {
	switch cfg.backend {
	case "", "filesystem":
		reader := service.NewDefaultArtifactReader(nil)
		return reader, service.DirectArtifactAccessIssuer{}, service.LocalTerminalPersister{}, nil
	case "r2":
		store, err := service.NewS3ArtifactStore(ctx, service.S3ArtifactConfig{
			Bucket:          cfg.bucket,
			Endpoint:        cfg.endpoint,
			AccessKeyID:     cfg.accessKeyID,
			SecretAccessKey: cfg.secretAccessKey,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		persister, err := service.NewS3TerminalPersister(store)
		if err != nil {
			return nil, nil, nil, err
		}
		reader := service.NewDefaultArtifactReader(store)
		return reader, service.NewS3ArtifactAccessIssuer(store), persister, nil
	default:
		return nil, nil, nil, fmt.Errorf("unsupported artifact backend %q", cfg.backend)
	}
}

func isOpaqueArtifactBackend(persister service.TerminalPersister) bool {
	_, ok := persister.(*service.S3TerminalPersister)
	return ok
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

func selectArtifactPath(detail service.MatchDetail, artifactKind string) (string, error) {
	switch {
	case artifactKind == "result-summary":
		if detail.ResultSummaryPath == "" {
			return "", fmt.Errorf("artifact result-summary is not available")
		}
		return detail.ResultSummaryPath, nil
	case artifactKind == "record":
		if detail.RecordPath == "" {
			return "", fmt.Errorf("artifact record is not available")
		}
		return detail.RecordPath, nil
	case artifactKind == "snapshot":
		if detail.ReplayInputs == nil || detail.ReplayInputs.SnapshotPath == "" {
			return "", fmt.Errorf("artifact snapshot is not available")
		}
		return detail.ReplayInputs.SnapshotPath, nil
	case artifactKind == "history":
		if detail.ReplayInputs == nil || detail.ReplayInputs.HistoryPath == "" {
			return "", fmt.Errorf("artifact history is not available")
		}
		return detail.ReplayInputs.HistoryPath, nil
	case artifactKind == "exported-snapshot":
		if detail.ReplayInputs == nil || detail.ReplayInputs.ExportedSnapshotPath == "" {
			return "", fmt.Errorf("artifact exported-snapshot is not available")
		}
		return detail.ReplayInputs.ExportedSnapshotPath, nil
	case strings.HasPrefix(artifactKind, "stderr:"):
		playerID := strings.TrimPrefix(artifactKind, "stderr:")
		if playerID == "" {
			return "", fmt.Errorf("artifact stderr requires a player id")
		}
		path := detail.PlayerStderrPaths[playerID]
		if path == "" {
			return "", fmt.Errorf("artifact stderr for player %q is not available", playerID)
		}
		return path, nil
	default:
		return "", fmt.Errorf("unsupported artifact %q", artifactKind)
	}
}
