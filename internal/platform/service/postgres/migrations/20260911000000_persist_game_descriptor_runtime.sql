ALTER TABLE game_releases
    ADD COLUMN runtime_args JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN memory_limit_pages INTEGER NOT NULL DEFAULT 0 CHECK (memory_limit_pages >= 0);
