-- +goose Up
CREATE TABLE users (
    id                BIGSERIAL PRIMARY KEY,
    username          TEXT NOT NULL CHECK (char_length(username) BETWEEN 3 AND 32),
    email             TEXT NOT NULL CHECK (char_length(email) <= 254),
    password_hash     TEXT NOT NULL,
    email_verified_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_username_lower_uq ON users (LOWER(username));
CREATE UNIQUE INDEX users_email_lower_uq ON users (LOWER(email));

-- One-time tokens sent by email. Only hashes are stored.
CREATE TABLE user_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    TEXT NOT NULL CHECK (purpose IN ('email_verification', 'password_reset')),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX user_tokens_user_purpose_idx ON user_tokens (user_id, purpose);

CREATE TABLE sessions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    user_agent TEXT,
    ip_address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- Categories become private to their owner. Rows created before accounts existed have
-- no owner and are removed.
DELETE FROM categories;

ALTER TABLE categories DROP CONSTRAINT categories_parent_id_fkey;
DROP INDEX categories_sibling_name_uq;

ALTER TABLE categories ADD COLUMN user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE categories ADD CONSTRAINT categories_id_user_uq UNIQUE (id, user_id);
-- The composite key guarantees that a category can only be nested below one of the same owner.
ALTER TABLE categories ADD CONSTRAINT categories_parent_fkey
    FOREIGN KEY (parent_id, user_id) REFERENCES categories (id, user_id) ON DELETE CASCADE;

CREATE UNIQUE INDEX categories_sibling_name_uq
    ON categories (user_id, COALESCE(parent_id, 0), LOWER(name));
CREATE INDEX categories_user_id_idx ON categories (user_id);

-- +goose Down
DROP INDEX categories_user_id_idx;
DROP INDEX categories_sibling_name_uq;
ALTER TABLE categories DROP CONSTRAINT categories_parent_fkey;
ALTER TABLE categories DROP CONSTRAINT categories_id_user_uq;
ALTER TABLE categories DROP COLUMN user_id;
ALTER TABLE categories ADD CONSTRAINT categories_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES categories (id) ON DELETE CASCADE;
CREATE UNIQUE INDEX categories_sibling_name_uq ON categories (COALESCE(parent_id, 0), LOWER(name));

DROP TABLE sessions;
DROP TABLE user_tokens;
DROP TABLE users;
