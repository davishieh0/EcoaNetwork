-- The tables every event goes through, whatever its type. The tables of each
-- subject (reviews, votes, entities...) come with the step that projects it.

-- +goose Up

-- Deterministic: every event the indexer holds, as signed. A projection of
-- the nodes' logs, not a second log: dropping it and reading the logs again
-- brings it back.
CREATE TABLE events (
    event_id   text        PRIMARY KEY,
    event_type text        NOT NULL,
    author     text        NOT NULL,
    created_at timestamptz NOT NULL,

    -- bytea, never jsonb: jsonb reorders keys and rewrites numbers, and the
    -- client checks the signature over these exact bytes
    envelope   bytea       NOT NULL
);

-- Observed: when the network had each event, the earliest any node reported.
-- Apart from events, so a rebuild can compare the deterministic tables alone.
CREATE TABLE event_observations (
    event_id    text        PRIMARY KEY REFERENCES events (event_id),
    received_at timestamptz NOT NULL
);

-- How far the indexer has read each node's log. Saved in the same
-- transaction as what the events changed.
CREATE TABLE projector_cursors (
    -- the node's address, as configured
    node     text   PRIMARY KEY,

    -- the log `sequence` counts in; when the node's changes, it starts over
    log_id   text   NOT NULL,
    sequence bigint NOT NULL CHECK (sequence >= 0)
);

-- +goose Down

DROP TABLE projector_cursors;
DROP TABLE event_observations;
DROP TABLE events;
