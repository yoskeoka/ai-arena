CREATE TABLE signup_invites (
    invite_id UUID PRIMARY KEY,
    invite_token_hash TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    claimed_account_id UUID REFERENCES accounts (account_id) ON DELETE SET NULL,
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
