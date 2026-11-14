-- +goose Up
-- Activities and time entries get an owner so that the database itself guarantees that
-- they can only reference rows of the same user (composite foreign keys), like categories.
-- Both tables have never been writable through the API, so they are empty.

ALTER TABLE activities ADD COLUMN user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE activities ADD CONSTRAINT activities_id_user_uq UNIQUE (id, user_id);
ALTER TABLE activities DROP CONSTRAINT activities_category_id_fkey;
ALTER TABLE activities ADD CONSTRAINT activities_category_fkey
    FOREIGN KEY (category_id, user_id) REFERENCES categories (id, user_id) ON DELETE CASCADE;
ALTER TABLE activities DROP CONSTRAINT activities_category_id_name_key;
CREATE UNIQUE INDEX activities_category_name_uq ON activities (category_id, LOWER(name));
ALTER TABLE activities ADD CONSTRAINT activities_minimum_minutes_max CHECK (minimum_minutes <= 1440);
CREATE INDEX activities_user_id_idx ON activities (user_id);

ALTER TABLE time_entries ADD COLUMN user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE;
-- logged_at is when the activity happened (editable); created_at is when the row was written.
ALTER TABLE time_entries ADD COLUMN logged_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE time_entries DROP CONSTRAINT time_entries_activity_id_fkey;
ALTER TABLE time_entries ADD CONSTRAINT time_entries_activity_fkey
    FOREIGN KEY (activity_id, user_id) REFERENCES activities (id, user_id) ON DELETE CASCADE;
ALTER TABLE time_entries ADD CONSTRAINT time_entries_minutes_max CHECK (minutes <= 1440);
ALTER TABLE time_entries ADD CONSTRAINT time_entries_note_length CHECK (note IS NULL OR char_length(note) <= 500);
CREATE INDEX time_entries_user_logged_at_idx ON time_entries (user_id, logged_at);

-- At most one timer runs per user; it lives on the server so it survives reloads and
-- can be stopped from another device.
CREATE TABLE running_timers (
    user_id     BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    activity_id BIGINT NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (activity_id, user_id) REFERENCES activities (id, user_id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE running_timers;

DROP INDEX time_entries_user_logged_at_idx;
ALTER TABLE time_entries DROP CONSTRAINT time_entries_note_length;
ALTER TABLE time_entries DROP CONSTRAINT time_entries_minutes_max;
ALTER TABLE time_entries DROP CONSTRAINT time_entries_activity_fkey;
ALTER TABLE time_entries DROP COLUMN logged_at;
ALTER TABLE time_entries DROP COLUMN user_id;
ALTER TABLE time_entries ADD CONSTRAINT time_entries_activity_id_fkey
    FOREIGN KEY (activity_id) REFERENCES activities (id) ON DELETE CASCADE;

DROP INDEX activities_user_id_idx;
ALTER TABLE activities DROP CONSTRAINT activities_minimum_minutes_max;
DROP INDEX activities_category_name_uq;
ALTER TABLE activities ADD CONSTRAINT activities_category_id_name_key UNIQUE (category_id, name);
ALTER TABLE activities DROP CONSTRAINT activities_category_fkey;
ALTER TABLE activities ADD CONSTRAINT activities_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE CASCADE;
ALTER TABLE activities DROP CONSTRAINT activities_id_user_uq;
ALTER TABLE activities DROP COLUMN user_id;
