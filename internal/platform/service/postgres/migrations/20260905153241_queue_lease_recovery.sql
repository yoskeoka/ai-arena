-- Modify "service_queue_records" table
ALTER TABLE "public"."service_queue_records" ADD COLUMN "lease_deadline" timestamptz NULL, ADD COLUMN "last_heartbeat_at" timestamptz NULL;
