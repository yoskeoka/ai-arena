-- name: CreateQueueRecord :exec
INSERT INTO service_queue_records (
    submission_id,
    match_id,
    parent_run_id,
    run_kind,
    official,
    game_id,
    game_version,
    ruleset_version,
    players_json,
    output_dir,
    attempt_count,
    state
)
VALUES (
    @submission_id,
    @match_id,
    @parent_run_id,
    @run_kind,
    @official,
    @game_id,
    @game_version,
    @ruleset_version,
    @players_json,
    @output_dir,
    @attempt_count,
    @state
);

-- name: ClaimNextQueueRecord :one
WITH next_record AS (
    SELECT submission_id
    FROM service_queue_records
    WHERE service_queue_records.state = @queued_state
    ORDER BY queue_order
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE service_queue_records AS records
SET state = @leased_state, worker_id = @worker_id, updated_at = NOW()
FROM next_record
WHERE records.submission_id = next_record.submission_id
RETURNING
    records.submission_id,
    records.match_id,
    records.parent_run_id,
    records.run_kind,
    records.official,
    records.game_id,
    records.game_version,
    records.ruleset_version,
    records.players_json,
    records.output_dir,
    records.attempt_count,
    records.state,
    records.worker_id,
    records.terminal_json;

-- name: UpdateQueueRecord :exec
UPDATE service_queue_records
SET
    match_id = @match_id,
    parent_run_id = @parent_run_id,
    run_kind = @run_kind,
    official = @official,
    game_id = @game_id,
    game_version = @game_version,
    ruleset_version = @ruleset_version,
    players_json = @players_json,
    output_dir = @output_dir,
    attempt_count = @attempt_count,
    state = @state,
    worker_id = @worker_id,
    terminal_json = @terminal_json,
    updated_at = NOW()
WHERE submission_id = @submission_id;

-- name: CancelQueueRecord :exec
UPDATE service_queue_records
SET state = @state, worker_id = NULL, terminal_json = NULL, updated_at = NOW()
WHERE submission_id = @submission_id;

-- name: GetQueueRecord :one
SELECT
    submission_id,
    match_id,
    parent_run_id,
    run_kind,
    official,
    game_id,
    game_version,
    ruleset_version,
    players_json,
    output_dir,
    attempt_count,
    state,
    worker_id,
    terminal_json
FROM service_queue_records
WHERE submission_id = @submission_id;

-- name: GetQueueRecordForUpdate :one
SELECT
    submission_id,
    match_id,
    parent_run_id,
    run_kind,
    official,
    game_id,
    game_version,
    ruleset_version,
    players_json,
    output_dir,
    attempt_count,
    state,
    worker_id,
    terminal_json
FROM service_queue_records
WHERE submission_id = @submission_id
FOR UPDATE;

-- name: ListQueueRecords :many
SELECT
    submission_id,
    match_id,
    parent_run_id,
    run_kind,
    official,
    game_id,
    game_version,
    ruleset_version,
    players_json,
    output_dir,
    attempt_count,
    state,
    worker_id,
    terminal_json
FROM service_queue_records
ORDER BY queue_order;
