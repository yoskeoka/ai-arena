package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"github.com/yoskeoka/ai-arena/internal/platform/artifacts"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
)

var (
	// ErrRankingSnapshotNotFound reports that no durable ranking snapshot exists for the requested scope.
	ErrRankingSnapshotNotFound = errors.New("service: ranking snapshot not found")
)

// RankingScope identifies one durable ranking snapshot family.
type RankingScope struct {
	GameID         string `json:"game_id"`
	GameVersion    string `json:"game_version"`
	RulesetVersion string `json:"ruleset_version"`
}

// RankingEntry aggregates one competitor across completed submissions in one scope.
type RankingEntry struct {
	CompetitorRef   string           `json:"competitor_ref"`
	LastPlayerID    string           `json:"last_player_id"`
	MatchesPlayed   int              `json:"matches_played"`
	FirstPlaces     int              `json:"first_places"`
	PlacementCounts map[int]int      `json:"placement_counts,omitempty"`
	LastRunID       string           `json:"last_run_id"`
	LastMatchID     string           `json:"last_match_id"`
	LastStatus      game.MatchStatus `json:"last_status"`
}

// RankingSnapshot is the durable aggregate payload for one ranking scope.
type RankingSnapshot struct {
	Scope              RankingScope   `json:"scope"`
	AppliedRunIDs      []string       `json:"applied_run_ids,omitempty"`
	AppliedMatchIDs    []string       `json:"applied_match_ids,omitempty"`
	LastAppliedRunID   string         `json:"last_applied_run_id,omitempty"`
	LastAppliedMatchID string         `json:"last_applied_match_id,omitempty"`
	CompletedMatches   int            `json:"completed_matches"`
	Entries            []RankingEntry `json:"entries,omitempty"`
}

// StoredRankingSnapshot returns the durable snapshot plus its stable locator.
type StoredRankingSnapshot struct {
	Locator  string          `json:"locator"`
	Snapshot RankingSnapshot `json:"snapshot"`
}

// RankingVerification compares the durable snapshot with a recomputed one.
type RankingVerification struct {
	Scope            RankingScope           `json:"scope"`
	Stored           *StoredRankingSnapshot `json:"stored,omitempty"`
	Recomputed       RankingSnapshot        `json:"recomputed"`
	StoredSnapshotOK bool                   `json:"stored_snapshot_ok"`
}

type rankingUpdate struct {
	Scope      RankingScope
	RunID      string
	MatchID    string
	Status     game.MatchStatus
	Placements []rankingPlacement
}

type rankingPlacement struct {
	CompetitorRef string
	PlayerID      string
	Place         int
}

// RankingSnapshotStore persists ranking snapshots on a durable backend.
type RankingSnapshotStore interface {
	Get(context.Context, RankingScope) (RankingSnapshot, string, error)
	Put(context.Context, RankingSnapshot) (string, error)
}

// RankingUpdater applies one completed submission to the durable ranking snapshot.
type RankingUpdater interface {
	ApplyCompleted(context.Context, MatchSubmission, artifacts.ResultSummary) error
}

// RankingService manages durable ranking snapshots and recomputation helpers.
type RankingService struct {
	store  RankingSnapshotStore
	queue  QueueStore
	reader ArtifactReader
}

// NewRankingService constructs the ranking lifecycle service.
func NewRankingService(store RankingSnapshotStore, queue QueueStore, readers ...ArtifactReader) (*RankingService, error) {
	if store == nil {
		return nil, fmt.Errorf("service: ranking snapshot store is required")
	}
	reader := ArtifactReader(NewDefaultArtifactReader(nil))
	if len(readers) > 0 && readers[0] != nil {
		reader = readers[0]
	}
	return &RankingService{
		store:  store,
		queue:  queue,
		reader: reader,
	}, nil
}

// ApplyCompleted applies one completed submission to the durable snapshot in its scope.
func (s *RankingService) ApplyCompleted(ctx context.Context, submission MatchSubmission, summary artifacts.ResultSummary) error {
	if !submission.Official {
		return nil
	}
	update, err := buildRankingUpdate(submission, summary)
	if err != nil {
		return err
	}
	current, _, err := s.store.Get(ctx, update.Scope)
	switch {
	case err == nil:
	case errors.Is(err, ErrRankingSnapshotNotFound):
		current = newRankingSnapshot(update.Scope)
	default:
		return err
	}
	next, err := applyRankingUpdate(current, update)
	if err != nil {
		return err
	}
	_, err = s.store.Put(ctx, next)
	return err
}

// RefreshCompletedRun recomputes and stores the snapshot for one completed run scope.
func (s *RankingService) RefreshCompletedRun(ctx context.Context, record QueueRecord) error {
	if record.Terminal == nil {
		return fmt.Errorf("service: terminal artifacts are required to refresh ranking")
	}
	summary, err := readResultSummary(ctx, s.reader, record.Terminal.ResultSummaryPath)
	if err != nil {
		return err
	}
	if summary == nil {
		return fmt.Errorf("service: result summary is required to refresh ranking")
	}
	scope := scopeFromSummary(*summary)
	recomputed, err := s.Recompute(ctx, scope)
	if err != nil {
		return err
	}
	_, err = s.store.Put(ctx, recomputed)
	return err
}

// Get returns the current durable snapshot for one scope.
func (s *RankingService) Get(ctx context.Context, scope RankingScope) (StoredRankingSnapshot, error) {
	scope = normalizeRankingScope(scope)
	if err := validateRankingScope(scope); err != nil {
		return StoredRankingSnapshot{}, err
	}
	snapshot, locator, err := s.store.Get(ctx, scope)
	if err != nil {
		return StoredRankingSnapshot{}, err
	}
	return StoredRankingSnapshot{Locator: locator, Snapshot: normalizeRankingSnapshot(snapshot)}, nil
}

// Recompute rebuilds the ranking snapshot for one scope from completed queue records.
func (s *RankingService) Recompute(ctx context.Context, scope RankingScope) (RankingSnapshot, error) {
	scope = normalizeRankingScope(scope)
	if err := validateRankingScope(scope); err != nil {
		return RankingSnapshot{}, err
	}
	if s.queue == nil {
		return RankingSnapshot{}, fmt.Errorf("service: queue store is required for ranking recompute")
	}
	records, err := s.queue.List(ctx)
	if err != nil {
		return RankingSnapshot{}, err
	}
	snapshot := newRankingSnapshot(scope)
	for _, record := range records {
		if record.State != StateCompleted || !record.Submission.Official || record.Terminal == nil || strings.TrimSpace(record.Terminal.ResultSummaryPath) == "" {
			continue
		}
		summary, err := readResultSummary(ctx, s.reader, record.Terminal.ResultSummaryPath)
		if err != nil {
			return RankingSnapshot{}, err
		}
		if summary == nil {
			continue
		}
		if normalizeRankingScope(scopeFromSummary(*summary)) != scope {
			continue
		}
		update, err := buildRankingUpdate(record.Submission, *summary)
		if err != nil {
			return RankingSnapshot{}, err
		}
		snapshot, err = applyRankingUpdate(snapshot, update)
		if err != nil {
			return RankingSnapshot{}, err
		}
	}
	return normalizeRankingSnapshot(snapshot), nil
}

// Verify compares the durable snapshot and a recomputed snapshot for one scope.
func (s *RankingService) Verify(ctx context.Context, scope RankingScope) (RankingVerification, error) {
	scope = normalizeRankingScope(scope)
	recomputed, err := s.Recompute(ctx, scope)
	if err != nil {
		return RankingVerification{}, err
	}
	verification := RankingVerification{
		Scope:      scope,
		Recomputed: recomputed,
	}
	stored, err := s.Get(ctx, scope)
	switch {
	case err == nil:
		verification.Stored = &stored
		verification.StoredSnapshotOK = rankingSnapshotsEqual(stored.Snapshot, recomputed)
		return verification, nil
	case errors.Is(err, ErrRankingSnapshotNotFound):
		return verification, nil
	default:
		return RankingVerification{}, err
	}
}

// LocalRankingSnapshotStore persists ranking snapshots under one local base directory.
type LocalRankingSnapshotStore struct {
	baseDir string
}

// NewLocalRankingSnapshotStore constructs a filesystem-backed ranking snapshot store.
func NewLocalRankingSnapshotStore(baseDir string) (*LocalRankingSnapshotStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("service: base_dir is required")
	}
	return &LocalRankingSnapshotStore{baseDir: filepath.Clean(baseDir)}, nil
}

// Get loads one local ranking snapshot.
func (s *LocalRankingSnapshotStore) Get(_ context.Context, scope RankingScope) (RankingSnapshot, string, error) {
	scope = normalizeRankingScope(scope)
	if err := validateRankingScope(scope); err != nil {
		return RankingSnapshot{}, "", err
	}
	locator := s.locator(scope)
	data, err := os.ReadFile(filepath.Clean(locator))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RankingSnapshot{}, "", ErrRankingSnapshotNotFound
		}
		return RankingSnapshot{}, "", fmt.Errorf("service: read ranking snapshot %s: %w", locator, err)
	}
	snapshot, err := decodeRankingSnapshot(data)
	if err != nil {
		return RankingSnapshot{}, "", err
	}
	return normalizeRankingSnapshot(snapshot), locator, nil
}

// Put writes one local ranking snapshot.
func (s *LocalRankingSnapshotStore) Put(_ context.Context, snapshot RankingSnapshot) (string, error) {
	snapshot = normalizeRankingSnapshot(snapshot)
	if err := validateRankingScope(snapshot.Scope); err != nil {
		return "", err
	}
	locator := s.locator(snapshot.Scope)
	// #nosec G703 -- locator is derived from validated scope fields via escapeRankingPathSegment under baseDir.
	if err := os.MkdirAll(filepath.Dir(locator), 0o750); err != nil {
		return "", fmt.Errorf("service: create ranking snapshot dir %s: %w", filepath.Dir(locator), err)
	}
	data, err := encodeRankingSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	// #nosec G703 -- locator is derived from validated scope fields via escapeRankingPathSegment under baseDir.
	if err := os.WriteFile(locator, data, 0o600); err != nil {
		return "", fmt.Errorf("service: write ranking snapshot %s: %w", locator, err)
	}
	return locator, nil
}

func (s *LocalRankingSnapshotStore) locator(scope RankingScope) string {
	return filepath.Join(
		s.baseDir,
		".arena-service",
		"rankings",
		escapeRankingPathSegment(scope.GameID),
		escapeRankingPathSegment(scope.GameVersion),
		escapeRankingPathSegment(scope.RulesetVersion),
		"snapshot.json",
	)
}

// S3RankingSnapshotStore persists ranking snapshots under the shared artifact bucket.
type S3RankingSnapshotStore struct {
	store *S3ArtifactStore
}

// NewS3RankingSnapshotStore constructs an object-storage-backed ranking snapshot store.
func NewS3RankingSnapshotStore(store *S3ArtifactStore) (*S3RankingSnapshotStore, error) {
	if store == nil {
		return nil, fmt.Errorf("service: S3 artifact store is required")
	}
	return &S3RankingSnapshotStore{store: store}, nil
}

// Get loads one ranking snapshot from the shared object store.
func (s *S3RankingSnapshotStore) Get(ctx context.Context, scope RankingScope) (RankingSnapshot, string, error) {
	scope = normalizeRankingScope(scope)
	if err := validateRankingScope(scope); err != nil {
		return RankingSnapshot{}, "", err
	}
	key := rankingSnapshotObjectKey(scope)
	locator := s.store.ObjectLocator(key)
	resp, err := s.store.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.store.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey" {
			return RankingSnapshot{}, "", ErrRankingSnapshotNotFound
		}
		return RankingSnapshot{}, "", fmt.Errorf("service: get ranking snapshot %s: %w", locator, err)
	}
	defer resp.Body.Close()
	data := new(bytes.Buffer)
	if _, err := data.ReadFrom(resp.Body); err != nil {
		return RankingSnapshot{}, "", fmt.Errorf("service: read ranking snapshot %s: %w", locator, err)
	}
	snapshot, err := decodeRankingSnapshot(data.Bytes())
	if err != nil {
		return RankingSnapshot{}, "", err
	}
	return normalizeRankingSnapshot(snapshot), locator, nil
}

// Put writes one ranking snapshot into the shared object store.
func (s *S3RankingSnapshotStore) Put(ctx context.Context, snapshot RankingSnapshot) (string, error) {
	snapshot = normalizeRankingSnapshot(snapshot)
	if err := validateRankingScope(snapshot.Scope); err != nil {
		return "", err
	}
	data, err := encodeRankingSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	return s.store.PutBytes(ctx, rankingSnapshotObjectKey(snapshot.Scope), data, "application/json")
}

func buildRankingUpdate(submission MatchSubmission, summary artifacts.ResultSummary) (rankingUpdate, error) {
	scope := scopeFromSummary(summary)
	if err := validateRankingScope(scope); err != nil {
		return rankingUpdate{}, err
	}
	if summary.Status != game.StatusCompleted {
		return rankingUpdate{}, fmt.Errorf("service: ranking summary status must be completed")
	}
	if strings.TrimSpace(submission.RunID) == "" {
		return rankingUpdate{}, fmt.Errorf("service: run_id is required for ranking update")
	}
	if strings.TrimSpace(summary.MatchID) == "" {
		return rankingUpdate{}, fmt.Errorf("service: match_id is required for ranking update")
	}
	if len(summary.Placements) == 0 {
		return rankingUpdate{}, fmt.Errorf("service: ranking placements are required")
	}
	playerByID := make(map[string]SubmittedPlayer, len(submission.Players))
	for _, player := range submission.Players {
		playerByID[player.PlayerID] = player
	}
	seenPlayers := make(map[string]struct{}, len(summary.Placements))
	placements := make([]rankingPlacement, 0, len(summary.Placements))
	for _, placement := range summary.Placements {
		if placement.Place <= 0 {
			return rankingUpdate{}, fmt.Errorf("service: placement place must be positive for player %q", placement.PlayerID)
		}
		player, ok := playerByID[placement.PlayerID]
		if !ok {
			return rankingUpdate{}, fmt.Errorf("service: ranking placement player %q is not present in submission", placement.PlayerID)
		}
		if _, ok := seenPlayers[placement.PlayerID]; ok {
			return rankingUpdate{}, fmt.Errorf("service: duplicate ranking placement for player %q", placement.PlayerID)
		}
		if strings.TrimSpace(player.ArtifactRef) == "" {
			return rankingUpdate{}, fmt.Errorf("service: artifact_ref is required for ranking competitor %q", placement.PlayerID)
		}
		seenPlayers[placement.PlayerID] = struct{}{}
		placements = append(placements, rankingPlacement{
			CompetitorRef: player.ArtifactRef,
			PlayerID:      placement.PlayerID,
			Place:         placement.Place,
		})
	}
	return rankingUpdate{
		Scope:      scope,
		RunID:      submission.RunID,
		MatchID:    summary.MatchID,
		Status:     summary.Status,
		Placements: placements,
	}, nil
}

func applyRankingUpdate(snapshot RankingSnapshot, update rankingUpdate) (RankingSnapshot, error) {
	snapshot = normalizeRankingSnapshot(snapshot)
	if snapshot.Scope != normalizeRankingScope(update.Scope) {
		return RankingSnapshot{}, fmt.Errorf("service: ranking scope mismatch")
	}
	if slices.Contains(snapshot.AppliedRunIDs, update.RunID) || slices.Contains(snapshot.AppliedMatchIDs, update.MatchID) {
		return snapshot, nil
	}

	entries := make(map[string]RankingEntry, len(snapshot.Entries))
	for _, entry := range snapshot.Entries {
		entry = normalizeRankingEntry(entry)
		entries[entry.CompetitorRef] = entry
	}
	for _, placement := range update.Placements {
		entry := normalizeRankingEntry(entries[placement.CompetitorRef])
		entry.CompetitorRef = placement.CompetitorRef
		entry.LastPlayerID = placement.PlayerID
		entry.MatchesPlayed++
		entry.PlacementCounts[placement.Place]++
		if placement.Place == 1 {
			entry.FirstPlaces++
		}
		entry.LastRunID = update.RunID
		entry.LastMatchID = update.MatchID
		entry.LastStatus = update.Status
		entries[placement.CompetitorRef] = entry
	}

	snapshot.AppliedRunIDs = append(snapshot.AppliedRunIDs, update.RunID)
	snapshot.AppliedMatchIDs = append(snapshot.AppliedMatchIDs, update.MatchID)
	snapshot.LastAppliedRunID = update.RunID
	snapshot.LastAppliedMatchID = update.MatchID
	snapshot.CompletedMatches = len(snapshot.AppliedMatchIDs)
	snapshot.Entries = make([]RankingEntry, 0, len(entries))
	for _, entry := range entries {
		snapshot.Entries = append(snapshot.Entries, normalizeRankingEntry(entry))
	}
	sort.Slice(snapshot.Entries, func(i, j int) bool {
		return snapshot.Entries[i].CompetitorRef < snapshot.Entries[j].CompetitorRef
	})
	return snapshot, nil
}

func newRankingSnapshot(scope RankingScope) RankingSnapshot {
	return RankingSnapshot{Scope: normalizeRankingScope(scope)}
}

func scopeFromSummary(summary artifacts.ResultSummary) RankingScope {
	return normalizeRankingScope(RankingScope{
		GameID:         summary.GameID,
		GameVersion:    summary.GameVersion,
		RulesetVersion: summary.RulesetVersion,
	})
}

func normalizeRankingScope(scope RankingScope) RankingScope {
	scope.GameID = strings.TrimSpace(scope.GameID)
	scope.GameVersion = strings.TrimSpace(scope.GameVersion)
	scope.RulesetVersion = strings.TrimSpace(scope.RulesetVersion)
	return scope
}

func validateRankingScope(scope RankingScope) error {
	if scope.GameID == "" {
		return fmt.Errorf("service: game_id is required")
	}
	if scope.GameVersion == "" {
		return fmt.Errorf("service: game_version is required")
	}
	if scope.RulesetVersion == "" {
		return fmt.Errorf("service: ruleset_version is required")
	}
	return nil
}

func normalizeRankingSnapshot(snapshot RankingSnapshot) RankingSnapshot {
	snapshot.Scope = normalizeRankingScope(snapshot.Scope)
	if snapshot.AppliedRunIDs == nil {
		snapshot.AppliedRunIDs = []string{}
	}
	if snapshot.AppliedMatchIDs == nil {
		snapshot.AppliedMatchIDs = []string{}
	}
	snapshot.CompletedMatches = len(snapshot.AppliedMatchIDs)
	if snapshot.Entries == nil {
		snapshot.Entries = []RankingEntry{}
	}
	for index := range snapshot.Entries {
		snapshot.Entries[index] = normalizeRankingEntry(snapshot.Entries[index])
	}
	sort.Slice(snapshot.Entries, func(i, j int) bool {
		return snapshot.Entries[i].CompetitorRef < snapshot.Entries[j].CompetitorRef
	})
	return snapshot
}

func normalizeRankingEntry(entry RankingEntry) RankingEntry {
	entry.CompetitorRef = strings.TrimSpace(entry.CompetitorRef)
	entry.LastPlayerID = strings.TrimSpace(entry.LastPlayerID)
	entry.LastRunID = strings.TrimSpace(entry.LastRunID)
	entry.LastMatchID = strings.TrimSpace(entry.LastMatchID)
	if entry.PlacementCounts == nil {
		entry.PlacementCounts = map[int]int{}
	}
	return entry
}

func rankingSnapshotsEqual(left, right RankingSnapshot) bool {
	left = normalizeRankingSnapshot(left)
	right = normalizeRankingSnapshot(right)
	return slices.Equal(left.AppliedRunIDs, right.AppliedRunIDs) &&
		slices.Equal(left.AppliedMatchIDs, right.AppliedMatchIDs) &&
		left.Scope == right.Scope &&
		left.LastAppliedRunID == right.LastAppliedRunID &&
		left.LastAppliedMatchID == right.LastAppliedMatchID &&
		left.CompletedMatches == right.CompletedMatches &&
		slices.EqualFunc(left.Entries, right.Entries, rankingEntriesEqual)
}

func rankingEntriesEqual(left, right RankingEntry) bool {
	left = normalizeRankingEntry(left)
	right = normalizeRankingEntry(right)
	return left.CompetitorRef == right.CompetitorRef &&
		left.LastPlayerID == right.LastPlayerID &&
		left.MatchesPlayed == right.MatchesPlayed &&
		left.FirstPlaces == right.FirstPlaces &&
		left.LastRunID == right.LastRunID &&
		left.LastMatchID == right.LastMatchID &&
		left.LastStatus == right.LastStatus &&
		equalPlacementCounts(left.PlacementCounts, right.PlacementCounts)
}

func equalPlacementCounts(left, right map[int]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func encodeRankingSnapshot(snapshot RankingSnapshot) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(normalizeRankingSnapshot(snapshot)); err != nil {
		return nil, fmt.Errorf("service: encode ranking snapshot: %w", err)
	}
	return buf.Bytes(), nil
}

func decodeRankingSnapshot(data []byte) (RankingSnapshot, error) {
	var snapshot RankingSnapshot
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&snapshot); err != nil {
		return RankingSnapshot{}, fmt.Errorf("service: decode ranking snapshot: %w", err)
	}
	return normalizeRankingSnapshot(snapshot), nil
}

func rankingSnapshotObjectKey(scope RankingScope) string {
	return path.Join(
		"rankings",
		escapeRankingPathSegment(scope.GameID),
		escapeRankingPathSegment(scope.GameVersion),
		escapeRankingPathSegment(scope.RulesetVersion),
		"snapshot.json",
	)
}

func escapeRankingPathSegment(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	return replacer.Replace(strings.TrimSpace(value))
}

func summaryFromRecord(result ExecutionResult) artifacts.ResultSummary {
	summary := artifacts.ResultSummary{
		MatchID:        result.Record.MatchID,
		GameID:         result.Record.Game.GameID,
		GameVersion:    result.Record.Game.GameVersion,
		RulesetVersion: result.Record.Game.RulesetVersion,
		Status:         result.Record.Status,
		Turn:           result.Record.Snapshot.Turn,
		Placements:     append([]game.Placement(nil), result.Record.Result.Placements...),
	}
	if errMsg := artifacts.TerminalError(result.Record); errMsg != "" {
		summary.Error = errMsg
	}
	return summary
}
