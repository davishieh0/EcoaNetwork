-- The queries of the votes. `make sqlc` turns each into a function of the
-- generated package, which projector.go uses.

-- name: IsReview :one
-- Whether an event is a version of a review: the only votes counted.
SELECT EXISTS (SELECT 1 FROM reviews WHERE event_id = $1);

-- name: SaveVote :exec
-- Stores one vote.
INSERT INTO votes (event_id, target, voter, kind, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: VoteEnvelopes :many
-- Every vote on one version, as signed, with the identity of its voter: the
-- key's, or the key itself when it never rotated.
SELECT e.envelope, COALESCE(i.identity, v.voter)::text AS identity
FROM votes v
JOIN events e ON e.event_id = v.event_id
LEFT JOIN identities i ON i.key = v.voter
WHERE v.target = $1;

-- name: VotedBy :many
-- The versions some keys voted on.
SELECT DISTINCT target FROM votes WHERE voter = ANY (sqlc.arg(keys)::text[]);

-- name: SaveTally :exec
-- Records the votes that count on one version.
INSERT INTO version_votes (version, endorsements, disputes)
VALUES ($1, $2, $3)
ON CONFLICT (version) DO UPDATE
SET endorsements = EXCLUDED.endorsements, disputes = EXCLUDED.disputes;
