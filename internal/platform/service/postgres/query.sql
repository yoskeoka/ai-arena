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
SET state = @leased_state,
    worker_id = @worker_id,
    lease_deadline = @lease_deadline,
    last_heartbeat_at = @last_heartbeat_at,
    updated_at = NOW()
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
    records.lease_deadline,
    records.last_heartbeat_at,
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
    lease_deadline = @lease_deadline,
    last_heartbeat_at = @last_heartbeat_at,
    terminal_json = @terminal_json,
    updated_at = NOW()
WHERE submission_id = @submission_id;

-- name: CancelQueueRecord :exec
UPDATE service_queue_records
SET state = @state, worker_id = NULL, lease_deadline = NULL, last_heartbeat_at = NULL, terminal_json = NULL, updated_at = NOW()
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
    lease_deadline,
    last_heartbeat_at,
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
    lease_deadline,
    last_heartbeat_at,
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
    lease_deadline,
    last_heartbeat_at,
    terminal_json
FROM service_queue_records
ORDER BY queue_order;

-- name: HeartbeatQueueRecord :execrows
UPDATE service_queue_records
SET lease_deadline = @lease_deadline, last_heartbeat_at = @last_heartbeat_at, updated_at = NOW()
WHERE submission_id = @submission_id AND worker_id = @worker_id
  AND state IN ('leased', 'running', 'persisting');

-- name: RecoverExpiredQueueRecords :execrows
UPDATE service_queue_records
SET state = @queued_state, worker_id = NULL, lease_deadline = NULL, last_heartbeat_at = NULL, updated_at = NOW()
WHERE state IN ('leased', 'running', 'persisting') AND lease_deadline <= @now;
-- name: ListOwnedBots :many
SELECT bot_id, owner_account_id, scope_id, bot_name, normalized_bot_name, lifecycle_state, active_submission_id, created_at, retired_at
FROM ai_bots
WHERE owner_account_id = $1 AND scope_id = $2
ORDER BY created_at;

-- name: ListSubmissionRevisionsForBot :many
SELECT ai_submission_id, bot_id, artifact_id, runtime_kind, ai_id, validation_state, created_at
FROM ai_submission_revisions
WHERE bot_id = $1
ORDER BY created_at;
