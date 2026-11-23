-- The reviews: one row per version, and one per chain of versions with the
-- head the protocol's rules resolve to (projection.ResolveHead).

-- +goose Up

-- Deterministic: every version of every review, the original and each edit.
CREATE TABLE reviews (
    event_id   text        PRIMARY KEY REFERENCES events (event_id),

    -- the original of the chain; equal to event_id for the original itself
    chain_root text        NOT NULL,

    -- the version this one edits; null for the original
    replaces   text        REFERENCES reviews (event_id),

    author     text        NOT NULL,

    -- always the root's: review.replaced carries no entity
    entity     text        NOT NULL,

    rating     smallint    NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title      text,
    body       text        NOT NULL,
    lang       text,
    created_at timestamptz NOT NULL
);

CREATE INDEX reviews_chain_root_idx ON reviews (chain_root);

-- Deterministic: one row per chain, with what the rules resolved.
CREATE TABLE review_chains (
    chain_root text        PRIMARY KEY REFERENCES reviews (event_id),
    entity     text        NOT NULL,

    -- the current version; null when the chain is truncated
    head       text        REFERENCES reviews (event_id),

    -- a withdrawal named any version of the chain
    withdrawn  boolean     NOT NULL DEFAULT false,

    -- the chain is longer than the 64 links a walk visits
    truncated  boolean     NOT NULL DEFAULT false,

    -- the original's createdAt: the listing's order, which an edit never moves
    created_at timestamptz NOT NULL
);

-- the entity's listing: its visible chains, newest first
CREATE INDEX review_chains_listing_idx
    ON review_chains (entity, created_at DESC, chain_root DESC)
    WHERE NOT withdrawn AND NOT truncated;

-- +goose Down

DROP TABLE review_chains;
DROP TABLE reviews;
