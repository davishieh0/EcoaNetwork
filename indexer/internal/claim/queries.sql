-- The queries of the claims. `make sqlc` turns each into a function of the
-- generated package, which projector.go and repository.go use.

-- name: SaveClaim :exec
-- Stores one claim.
INSERT INTO claims (event_id, claim_type, issuer, subject, entity,
                    evidence_ref, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: RevokeClaim :exec
-- Takes a claim back, only by its issuer: the same identity, whichever of
-- its keys signed each.
UPDATE claims SET revoked = true
WHERE event_id = sqlc.arg(claim)
  AND COALESCE((SELECT i.identity FROM identities i WHERE i.key = issuer),
               issuer)
    = COALESCE((SELECT i.identity FROM identities i
                WHERE i.key = sqlc.arg(issuer)),
               sqlc.arg(issuer)::text);
