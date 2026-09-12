-- Create "public_match_states" table
CREATE TABLE "public"."public_match_states" ("match_id" text NOT NULL, "run_id" text NOT NULL, "version" bigint NOT NULL, "exported_snapshot_json" jsonb NOT NULL, "updated_at" timestamptz NOT NULL DEFAULT now(), PRIMARY KEY ("match_id", "run_id"));
