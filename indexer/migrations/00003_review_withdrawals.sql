-- The withdrawals of reviews, kept so a chain can be resolved again with
-- every event that decides it (projection.ResolveHead takes them as input).

-- +goose Up

-- Deterministic: every review.withdrawn, with the chain of the version it named.
CREATE TABLE review_withdrawals (
    event_id   text PRIMARY KEY REFERENCES events (event_id),
    chain_root text NOT NULL REFERENCES review_chains (chain_root),

    -- the version it named; any version withdraws the whole chain
    target     text NOT NULL REFERENCES reviews (event_id)
);

CREATE INDEX review_withdrawals_chain_root_idx
    ON review_withdrawals (chain_root);

-- +goose Down

DROP TABLE review_withdrawals;
