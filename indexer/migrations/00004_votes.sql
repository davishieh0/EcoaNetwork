-- The votes on reviews, each bound to the exact version it targeted, and the
-- tally of each version under the protocol's rule (projection.CurrentVote).

-- +goose Up

-- Deterministic: every review.endorsed and review.disputed on a review.
CREATE TABLE votes (
    event_id   text        PRIMARY KEY REFERENCES events (event_id),

    -- the exact version voted on: a vote never follows an edit
    target     text        NOT NULL REFERENCES reviews (event_id),

    voter      text        NOT NULL,
    kind       text        NOT NULL CHECK (kind IN ('endorsed', 'disputed')),
    created_at timestamptz NOT NULL
);

CREATE INDEX votes_target_idx ON votes (target);

-- Deterministic: per version, the votes that count, one per voter, the latest
-- by the author's ordering.
CREATE TABLE version_votes (
    version      text    PRIMARY KEY REFERENCES reviews (event_id),
    endorsements integer NOT NULL CHECK (endorsements >= 0),
    disputes     integer NOT NULL CHECK (disputes >= 0)
);

-- +goose Down

DROP TABLE version_votes;
DROP TABLE votes;
