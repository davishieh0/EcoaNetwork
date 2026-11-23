-- The moderation labels. Every label is stored, whoever signed it: which ones
-- count is who this indexer follows, decided when it answers.

-- +goose Up

-- Deterministic: every moderation.label_created on a review or a response.
CREATE TABLE labels (
    event_id   text    PRIMARY KEY REFERENCES events (event_id),

    -- the event labeled
    target     text    NOT NULL,

    -- what the label applies to: a review's whole chain, so an edit does not
    -- escape it, or one response
    chain_root text    REFERENCES review_chains (chain_root),
    response   text    REFERENCES responses (event_id),

    moderator  text    NOT NULL,
    label      text    NOT NULL,
    note       text,

    -- its moderator took it back
    revoked    boolean NOT NULL DEFAULT false,

    CHECK ((chain_root IS NULL) <> (response IS NULL))
);

CREATE INDEX labels_chain_root_idx ON labels (chain_root)
    WHERE chain_root IS NOT NULL;

CREATE INDEX labels_response_idx ON labels (response)
    WHERE response IS NOT NULL;

-- +goose Down

DROP TABLE labels;
