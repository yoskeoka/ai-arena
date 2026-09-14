-- Modify "game_releases" table
ALTER TABLE "public"."game_releases" ALTER COLUMN "runtime_args" SET NOT NULL, ALTER COLUMN "runtime_args" SET DEFAULT '[]', ALTER COLUMN "memory_limit_pages" SET NOT NULL, ALTER COLUMN "memory_limit_pages" SET DEFAULT 0;
-- Modify "service_queue_records" table
ALTER TABLE "public"."service_queue_records" ADD COLUMN "completed_at" timestamptz NULL;
