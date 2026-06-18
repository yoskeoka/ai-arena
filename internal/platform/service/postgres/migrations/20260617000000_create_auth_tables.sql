CREATE TABLE accounts (
    account_id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE account_identities (
    identity_id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts (account_id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_subject TEXT NOT NULL,
    provider_login TEXT NOT NULL,
    provider_email TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_subject)
);

CREATE TABLE account_roles (
    account_id UUID NOT NULL REFERENCES accounts (account_id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (account_id, role)
);

CREATE TABLE account_sessions (
    session_id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts (account_id) ON DELETE CASCADE,
    session_token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE signup_invites (
    invite_id UUID PRIMARY KEY,
    invite_token_hash TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    claimed_account_id UUID REFERENCES accounts (account_id) ON DELETE SET NULL,
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
