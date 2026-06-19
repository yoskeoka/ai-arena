CREATE TABLE account_roles (
    account_id UUID NOT NULL REFERENCES accounts (account_id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (account_id, role)
);
