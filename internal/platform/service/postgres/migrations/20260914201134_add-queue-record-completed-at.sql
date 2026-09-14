-- Modify "service_queue_records" table
ALTER TABLE "public"."service_queue_records" ADD COLUMN "completed_at" timestamptz NULL;
