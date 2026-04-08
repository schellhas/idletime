CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    email_verified_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_uq ON users (LOWER(username));
CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_uq ON users (LOWER(email));

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS email_verification_tokens_user_id_idx
    ON email_verification_tokens (user_id);

CREATE TABLE IF NOT EXISTS sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    user_agent TEXT,
    ip_address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id);

CREATE TABLE IF NOT EXISTS categories (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id BIGINT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    multiplier NUMERIC(10, 2) NOT NULL DEFAULT 1.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS parent_id BIGINT NULL REFERENCES categories(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS categories_user_id_idx ON categories (user_id);
CREATE INDEX IF NOT EXISTS categories_parent_id_idx ON categories (parent_id);

UPDATE categories
SET name = 'root',
    parent_id = NULL
WHERE LOWER(name) = 'none'
  AND NOT EXISTS (
      SELECT 1
      FROM categories existing
      WHERE existing.user_id = categories.user_id
        AND LOWER(existing.name) = 'root'
  );

WITH user_roots AS (
    SELECT DISTINCT ON (user_id) user_id, id
    FROM categories
    WHERE LOWER(name) = 'root'
    ORDER BY user_id, id ASC
)
UPDATE categories AS child
SET parent_id = user_roots.id
FROM user_roots
WHERE child.user_id = user_roots.user_id
  AND child.id <> user_roots.id
  AND child.parent_id IS NULL;

CREATE TABLE IF NOT EXISTS activities (
    id BIGSERIAL PRIMARY KEY,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    multiplier NUMERIC(10, 2) NOT NULL DEFAULT 1.0,
    minimum_minutes INTEGER NOT NULL DEFAULT 0 CHECK (minimum_minutes >= 0),
    tracked_minutes INTEGER NOT NULL DEFAULT 0 CHECK (tracked_minutes >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (category_id, name)
);

CREATE INDEX IF NOT EXISTS activities_category_id_idx ON activities (category_id);

CREATE TABLE IF NOT EXISTS time_entries (
    id BIGSERIAL PRIMARY KEY,
    activity_id BIGINT NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    minutes INTEGER NOT NULL CHECK (minutes > 0),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS time_entries_activity_id_idx ON time_entries (activity_id);
