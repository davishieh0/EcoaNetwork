-- Identities through their keys: every rotation and revocation any node
-- accepted, and which keys are one person (identities.Resolve).

-- +goose Up

-- Deterministic: every identity.key_rotated.
CREATE TABLE key_rotations (
    event_id   text        PRIMARY KEY REFERENCES events (event_id),
    key        text        NOT NULL,
    successor  text        NOT NULL,

    -- the signed text, which the rule orders rotations by
    created_at text        NOT NULL
);

-- Deterministic: every identity.key_revoked.
CREATE TABLE key_revocations (
    event_id   text        PRIMARY KEY REFERENCES events (event_id),
    key        text        NOT NULL,
    reason     text,
    created_at timestamptz NOT NULL
);

CREATE INDEX key_revocations_key_idx ON key_revocations (key);

-- Which identity each key that rotated, or was rotated to, belongs to. A key
-- that is absent is its own identity. Worked out again from key_rotations
-- whenever a rotation arrives.
CREATE TABLE identities (
    key       text    PRIMARY KEY,

    -- the first key of the chain, which names the identity
    identity  text    NOT NULL,
    position  integer NOT NULL CHECK (position >= 0),

    -- a key of the identity has two successors
    contested boolean NOT NULL
);

CREATE INDEX identities_identity_idx ON identities (identity);

-- +goose Down

DROP TABLE identities;
DROP TABLE key_revocations;
DROP TABLE key_rotations;
