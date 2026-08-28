package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

func TestPostgresBotOwnershipSurvivesRestartAndEnforcesQuota(t *testing.T) {
	ctx := context.Background()
	dsn := postgresTestDSN(t)
	store, err := NewPostgresBotOwnershipStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, err = store.pool.Exec(ctx, `UPDATE ai_bots SET active_submission_id=NULL WHERE scope_id='test-v1-regular'; DELETE FROM ai_submission_revisions WHERE bot_id IN (SELECT bot_id FROM ai_bots WHERE scope_id='test-v1-regular'); DELETE FROM ai_bots WHERE scope_id='test-v1-regular'; INSERT INTO accounts(account_id) VALUES ('00000000-0000-0000-0000-000000000101') ON CONFLICT DO NOTHING; INSERT INTO game_releases(release_id,game_id,game_version,artifact_id) VALUES ('00000000-0000-0000-0000-000000000201','test','1.0.0','digest-test') ON CONFLICT DO NOTHING; INSERT INTO competition_scopes(scope_id,game_id,game_version_major,ruleset_version,active_release_id,player_count,max_active_bots_per_owner) VALUES ('test-v1-regular','test',1,'regular','00000000-0000-0000-0000-000000000201',2,1) ON CONFLICT DO NOTHING`)
	if err != nil {
		t.Fatal(err)
	}
	scope := CompetitionScope{ScopeID: "test-v1-regular"}
	owner := "00000000-0000-0000-0000-000000000101"
	bot, first, err := store.CreateOrRevise(ctx, BotRevisionRequest{Scope: scope, OwnerAccountID: owner, BotName: "Alpha", ArtifactID: "one", RuntimeKind: "wasm-wasi", AIID: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = NewPostgresBotOwnershipStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	updated, second, err := store.CreateOrRevise(ctx, BotRevisionRequest{Scope: scope, OwnerAccountID: owner, BotID: bot.BotID, ArtifactID: "two", RuntimeKind: "wasm-wasi", AIID: "alpha-v2"})
	if err != nil || updated.BotID != bot.BotID || first.AISubmissionID == second.AISubmissionID {
		t.Fatalf("revision = %+v %+v %v", updated, second, err)
	}
	_, _, err = store.CreateOrRevise(ctx, BotRevisionRequest{Scope: scope, OwnerAccountID: owner, BotName: "second", ArtifactID: "three", RuntimeKind: "wasm-wasi", AIID: "second"})
	if !errors.Is(err, ErrBotQuotaExceeded) {
		t.Fatalf("quota = %v", err)
	}
}

func TestPostgresGameRegistrationStorePersistsScopeActivation(t *testing.T) {
	ctx := context.Background()
	dsn := postgresTestDSN(t)
	store, err := NewPostgresGameRegistrationStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	record := RegisteredGame{RegistrationID: "persisted-game-v1-persisted", Game: contract.GameMetadata{GameID: "persisted-game", GameVersion: "1.0.0", RulesetVersion: "persisted"}, ArtifactID: "digest-persisted", PlayerCount: 2, MaxActiveBotsPerOwner: 3}
	if err := store.Save(ctx, record); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = NewPostgresGameRegistrationStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.Get(ctx, record.RegistrationID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ArtifactID != record.ArtifactID || got.PlayerCount != 2 || got.MaxActiveBotsPerOwner != 3 {
		t.Fatalf("scope = %+v", got)
	}
}
