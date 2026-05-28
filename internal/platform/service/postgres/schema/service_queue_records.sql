CREATE TABLE service_queue_records (
    submission_id TEXT PRIMARY KEY,
    queue_order BIGSERIAL NOT NULL UNIQUE,
    match_id TEXT NOT NULL,
    game_id TEXT NOT NULL,
    game_version TEXT NOT NULL,
    ruleset_version TEXT NOT NULL,
    players_json JSONB NOT NULL,
    output_dir TEXT NOT NULL,
    attempt_count INTEGER NOT NULL,
    state TEXT NOT NULL,
    worker_id TEXT,
    terminal_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
