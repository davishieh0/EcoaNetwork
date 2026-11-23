-- The profiles: per identity, its latest profile.updated by the author's own
-- ordering. A profile replaces the one before entirely.

-- +goose Up

-- Deterministic: the current profile of each identity that published one.
CREATE TABLE profiles (
    identity     text        PRIMARY KEY,

    -- the profile.updated that holds now
    event_id     text        NOT NULL REFERENCES events (event_id),

    -- each absent where that profile left it out: a profile is total, an
    -- omitted field is cleared
    display_name text,
    bio          text,
    lang         text,

    created_at   timestamptz NOT NULL
);

-- A label can name one profile.updated: it stays with that profile, and a new
-- one, a new statement, does not inherit it.
ALTER TABLE labels ADD COLUMN profile text REFERENCES events (event_id);
ALTER TABLE labels DROP CONSTRAINT labels_one_target;
ALTER TABLE labels ADD CONSTRAINT labels_one_target
    CHECK (num_nonnulls(chain_root, response, proposal, profile) = 1);

CREATE INDEX labels_profile_idx ON labels (profile) WHERE profile IS NOT NULL;

-- +goose Down

DROP INDEX labels_profile_idx;
ALTER TABLE labels DROP CONSTRAINT labels_one_target;
ALTER TABLE labels DROP COLUMN profile;
ALTER TABLE labels ADD CONSTRAINT labels_one_target
    CHECK (num_nonnulls(chain_root, response, proposal) = 1);
DROP TABLE profiles;
