CREATE TABLE game_releases (
    release_id UUID PRIMARY KEY,
    game_id TEXT NOT NULL,
    game_version TEXT NOT NULL,
    artifact_id TEXT NOT NULL UNIQUE,
    build_mode TEXT NOT NULL DEFAULT '',
    builder_id TEXT NOT NULL DEFAULT '',
    supported_rulesets JSONB NOT NULL DEFAULT '[]',
    source TEXT NOT NULL DEFAULT 'manual',
    source_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (game_id, game_version)
);

CREATE TABLE competition_scopes (
    scope_id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    game_version_major INTEGER NOT NULL CHECK (game_version_major > 0),
    ruleset_version TEXT NOT NULL,
    active_release_id UUID NOT NULL REFERENCES game_releases (release_id),
    player_count INTEGER NOT NULL CHECK (player_count > 0),
    max_active_bots_per_owner INTEGER NOT NULL CHECK (max_active_bots_per_owner > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (game_id, game_version_major, ruleset_version)
);

CREATE TABLE ai_bots (
    bot_id UUID PRIMARY KEY,
    owner_account_id UUID NOT NULL REFERENCES accounts (account_id),
    scope_id TEXT NOT NULL REFERENCES competition_scopes (scope_id),
    bot_name TEXT NOT NULL,
    normalized_bot_name TEXT NOT NULL,
    lifecycle_state TEXT NOT NULL CHECK (lifecycle_state IN ('active', 'retired')),
    active_submission_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    retired_at TIMESTAMPTZ,
    UNIQUE (owner_account_id, scope_id, normalized_bot_name)
);

CREATE TABLE ai_submission_revisions (
    ai_submission_id UUID PRIMARY KEY,
    bot_id UUID NOT NULL REFERENCES ai_bots (bot_id),
    artifact_id TEXT NOT NULL,
    runtime_kind TEXT NOT NULL,
    ai_id TEXT NOT NULL,
    validation_state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE ai_bots ADD CONSTRAINT ai_bots_active_submission_fk FOREIGN KEY (active_submission_id) REFERENCES ai_submission_revisions (ai_submission_id);
CREATE INDEX ai_bots_active_quota_idx ON ai_bots (owner_account_id, scope_id) WHERE lifecycle_state = 'active';
