-- The entities and the metadata proposed for them. An entity needs no event
-- to exist: the first review or proposal that names it is enough.

-- +goose Up

-- Deterministic: every entity a review or a proposal named.
CREATE TABLE entities (
    entity     text        PRIMARY KEY,

    -- the earliest createdAt of an event that named it
    first_seen timestamptz NOT NULL
);

-- the browse and search order, newest first
CREATE INDEX entities_first_seen_idx ON entities (first_seen DESC, entity DESC);

-- Deterministic: every entity.updated. A proposal, not a fact: which one sets
-- each field is worked out when the API answers (projection.ResolveFields).
CREATE TABLE entity_proposals (
    event_id    text        PRIMARY KEY REFERENCES events (event_id),
    entity      text        NOT NULL REFERENCES entities (entity),
    author      text        NOT NULL,
    created_at  timestamptz NOT NULL,

    -- each may be absent: a proposal can fix one field alone
    name        text,
    category    text,
    description text,

    -- what an entity: identifier is the hash of, when the proposal carries it
    descriptor  jsonb,

    -- name, lowercased and without accents, for search by part of it
    search_name text
);

CREATE INDEX entity_proposals_entity_idx ON entity_proposals (entity);

-- A label can name a proposal too: a labeled proposal leaves the contest.
ALTER TABLE labels ADD COLUMN proposal text REFERENCES entity_proposals (event_id);
ALTER TABLE labels DROP CONSTRAINT labels_check;
ALTER TABLE labels ADD CONSTRAINT labels_one_target
    CHECK (num_nonnulls(chain_root, response, proposal) = 1);

CREATE INDEX labels_proposal_idx ON labels (proposal) WHERE proposal IS NOT NULL;

-- +goose Down

DROP INDEX labels_proposal_idx;
ALTER TABLE labels DROP CONSTRAINT labels_one_target;
ALTER TABLE labels DROP COLUMN proposal;
ALTER TABLE labels ADD CONSTRAINT labels_check
    CHECK ((chain_root IS NULL) <> (response IS NULL));
DROP TABLE entity_proposals;
DROP TABLE entities;
