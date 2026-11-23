-- The responses to reviews. A response belongs to the chain of the review it
-- answers, not to one version, and may answer another response.

-- +goose Up

-- Deterministic: every review.response_created on a review or a response.
CREATE TABLE responses (
    event_id          text        PRIMARY KEY REFERENCES events (event_id),

    -- the chain it belongs to, also when it answers another response
    chain_root        text        NOT NULL REFERENCES review_chains (chain_root),

    -- what it answered: a version of the review, or another response
    target            text        NOT NULL,

    -- the version of the review the conversation answered
    responded_version text        NOT NULL REFERENCES reviews (event_id),

    -- 1 for an answer to the review, 2 for an answer to that, and so on
    depth             integer     NOT NULL CHECK (depth >= 1),

    author            text        NOT NULL,
    body              text        NOT NULL,
    lang              text,
    created_at        timestamptz NOT NULL,

    -- the order of the conversation: created_at, but never before what it
    -- answers. createdAt counts whole seconds, so a quick reply shares its
    -- parent's; it then sorts one microsecond after the parent instead.
    sort_at           timestamptz NOT NULL
);

-- a chain's responses, in the order of the conversation
CREATE INDEX responses_chain_idx ON responses (chain_root, sort_at, event_id);

-- +goose Down

DROP TABLE responses;
