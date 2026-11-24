-- Verification claims: every one stored, whoever signed it. Which ones count
-- is decided when the API answers: a domain claim by this indexer's own DNS
-- lookup, a human claim by whether it follows the verifier who issued it.

-- +goose Up

-- Deterministic: every verification.claim_created, as signed.
CREATE TABLE claims (
    event_id     text        PRIMARY KEY REFERENCES events (event_id),
    claim_type   text        NOT NULL,

    -- who signed it, and whom it is about: the same for a self-verifiable
    -- claim, which says nothing when they differ
    issuer       text        NOT NULL,
    subject      text        NOT NULL,

    entity       text,
    evidence_ref text,
    expires_at   timestamptz NOT NULL,

    -- its issuer took it back
    revoked      boolean     NOT NULL DEFAULT false
);

CREATE INDEX claims_subject_idx ON claims (subject);

CREATE INDEX claims_entity_idx ON claims (entity) WHERE entity IS NOT NULL;

-- Observed: this indexer's latest DNS lookup of each domain claim. Another
-- indexer, resolving through another resolver, may see another answer.
CREATE TABLE claim_checks (
    event_id       text        PRIMARY KEY REFERENCES claims (event_id),
    verified       boolean     NOT NULL,

    -- the answer was DNSSEC-validated, the stronger evidence
    dnssec         boolean     NOT NULL DEFAULT false,
    checked_at     timestamptz NOT NULL,

    -- since when the claim has verified without a gap: how long someone has
    -- been an entity's representative
    verified_since timestamptz,

    detail         text        NOT NULL DEFAULT ''
);

-- +goose Down

DROP TABLE claim_checks;
DROP TABLE claims;
