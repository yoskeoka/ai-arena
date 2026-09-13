CREATE TABLE public_match_states (
    match_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    version BIGINT NOT NULL,
    exported_snapshot_json JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (match_id, run_id)
);
