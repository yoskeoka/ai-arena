ALTER TABLE game_releases
    ADD COLUMN runtime_args JSONB,
    ADD COLUMN memory_limit_pages INTEGER CHECK (memory_limit_pages >= 0);
