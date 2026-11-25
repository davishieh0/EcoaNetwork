-- Which nodes hold each event: the inspection view shows it. Observed, as
-- event_observations is: another indexer, following other nodes, sees others.

-- +goose Up

-- Observed: each node that reported an event, where in its log, and when it
-- had it. A node reporting it again, after its log started over, moves it.
CREATE TABLE event_holders (
    event_id    text        NOT NULL REFERENCES events (event_id),

    -- the node's address, as configured, as projector_cursors names it
    node        text        NOT NULL,
    sequence    bigint      NOT NULL CHECK (sequence >= 0),
    received_at timestamptz,

    PRIMARY KEY (event_id, node)
);

-- +goose Down

DROP TABLE event_holders;
