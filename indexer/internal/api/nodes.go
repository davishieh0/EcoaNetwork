package api


import (
	"net/http"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/platform/respond"
	"github.com/jackc/pgx/v5/pgxpool"
)


// nodeStatus is what this indexer observed from one followed node's log.
type nodeStatus struct {
	Node           string     `json:"node"`
	LogID          string     `json:"logId"`
	Sequence       int64      `json:"sequence"`
	Events         int64      `json:"events"`
	LastReceivedAt *time.Time `json:"lastReceivedAt,omitempty"`
}


// nodesResponse is the answer of GET /nodes.
type nodesResponse struct {
	Nodes []nodeStatus `json:"nodes"`
}


// nodes answers GET /nodes from Postgres alone (projector_cursors and
// event_holders): reads never query nodes directly.
func nodes(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT c.node,
			       c.log_id,
			       c.sequence,
			       COALESCE(h.events, 0) AS events,
			       h.last_received_at
			FROM projector_cursors c
			LEFT JOIN (
				SELECT node,
				       COUNT(*) AS events,
				       MAX(received_at) AS last_received_at
				FROM event_holders
				GROUP BY node
			) h USING (node)
			ORDER BY c.node
		`)
		if err != nil {
			respond.Failed(w, err)
			return
		}
		defer rows.Close()

		list := make([]nodeStatus, 0)
		for rows.Next() {
			var s nodeStatus
			if err := rows.Scan(&s.Node, &s.LogID, &s.Sequence, &s.Events, &s.LastReceivedAt); err != nil {
				respond.Failed(w, err)
				return
			}
			if s.LastReceivedAt != nil {
				u := s.LastReceivedAt.UTC()
				s.LastReceivedAt = &u
			}
			list = append(list, s)
		}

		if err := rows.Err(); err != nil {
			respond.Failed(w, err)
			return
		}

		respond.JSON(w, http.StatusOK, nodesResponse{Nodes: list})
	}
}
