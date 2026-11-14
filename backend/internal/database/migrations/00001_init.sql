-- +goose Up
CREATE TABLE categories (
    id         BIGSERIAL PRIMARY KEY,
    parent_id  BIGINT REFERENCES categories (id) ON DELETE CASCADE,
    name       TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    multiplier NUMERIC(10, 2) NOT NULL DEFAULT 1.0 CHECK (multiplier > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sibling categories must have unique names; COALESCE makes this hold for top-level ones too.
CREATE UNIQUE INDEX categories_sibling_name_uq
    ON categories (COALESCE(parent_id, 0), LOWER(name));
CREATE INDEX categories_parent_id_idx ON categories (parent_id);

CREATE TABLE activities (
    id              BIGSERIAL PRIMARY KEY,
    category_id     BIGINT NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    name            TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    multiplier      NUMERIC(10, 2) NOT NULL DEFAULT 1.0 CHECK (multiplier > 0),
    minimum_minutes INTEGER NOT NULL DEFAULT 0 CHECK (minimum_minutes >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (category_id, name)
);

CREATE INDEX activities_category_id_idx ON activities (category_id);

CREATE TABLE time_entries (
    id          BIGSERIAL PRIMARY KEY,
    activity_id BIGINT NOT NULL REFERENCES activities (id) ON DELETE CASCADE,
    minutes     INTEGER NOT NULL CHECK (minutes > 0),
    note        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX time_entries_activity_id_idx ON time_entries (activity_id);

-- +goose Down
DROP TABLE time_entries;
DROP TABLE activities;
DROP TABLE categories;
