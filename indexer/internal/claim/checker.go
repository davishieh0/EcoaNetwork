package claim


import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/claim/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CONSTANTS ---

const (
	// Recheck is how long a passing lookup holds before it is done again: a
	// domain changes hands, and the claim is live state, not a fact.
	Recheck = 30 * 24 * time.Hour

	// retry is how long a failing lookup waits before it is tried again.
	retry = 30 * time.Second

	// every is how often the checker looks for lookups due.
	every = 10 * time.Second

	// batch is the most lookups one round does.
	batch = 50

	// lookupTimeout bounds one lookup.
	lookupTimeout = 5 * time.Second

	// recordPrefix names the TXT record a domain vouches through, and what
	// its value starts with.
	recordPrefix = "_ecoa-verify."
	valuePrefix  = "ecoa-identity="
)


// --- CODE ---

// Answer is what a TXT lookup found.
type Answer struct {
	Records []string

	// DNSSEC reports an answer the resolver validated.
	DNSSEC bool
}


// Lookup resolves the TXT records of a name.
type Lookup interface {
	TXT(ctx context.Context, name string) (Answer, error)
}


// DoH resolves over DNS-over-HTTPS, through the JSON API every major public
// resolver answers: the operator picks the resolver, rather than trusting
// whatever the host's default happens to be.
type DoH struct {
	URL    string
	Client *http.Client
}


// TXT asks the resolver for name's TXT records.
func (d DoH) TXT(ctx context.Context, name string) (Answer, error) {
	u, err := url.Parse(d.URL)
	if err != nil {
		return Answer{}, err
	}

	q := u.Query()
	q.Set("name", name)
	q.Set("type", "TXT")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Answer{}, err
	}

	req.Header.Set("Accept", "application/dns-json")

	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}

	res, err := client.Do(req)
	if err != nil {
		return Answer{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return Answer{}, fmt.Errorf("resolver answered %d", res.StatusCode)
	}

	var body struct {
		Status int  `json:"Status"`
		AD     bool `json:"AD"`
		Answer []struct {
			Type int    `json:"type"`
			Data string `json:"data"`
		} `json:"Answer"`
	}

	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return Answer{}, err
	}

	// NXDOMAIN and the like: the name has no record, not a failure to ask
	answer := Answer{DNSSEC: body.AD}

	for _, a := range body.Answer {
		if a.Type == 16 {
			answer.Records = append(answer.Records, unquote(a.Data))
		}
	}

	return answer, nil
}


// Plain resolves over ordinary DNS through one server: for a network that
// runs its own, as infra/local does. Unauthenticated, so never DNSSEC.
type Plain struct {
	Server string
}


// TXT asks the server for name's TXT records.
func (p Plain) TXT(ctx context.Context, name string) (Answer, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, p.Server)
		},
	}

	records, err := resolver.LookupTXT(ctx, name)

	var notFound *net.DNSError
	if errors.As(err, &notFound) && notFound.IsNotFound {
		return Answer{}, nil
	}

	return Answer{Records: records}, err
}


// unquote joins the quoted strings a DoH answer writes a TXT record as:
// `"part one" "part two"` reads "part onepart two".
func unquote(data string) string {
	if !strings.HasPrefix(data, `"`) {
		return data
	}

	var b strings.Builder
	in := false
	escaped := false

	for _, r := range data {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\' && in:
			escaped = true
		case r == '"':
			in = !in
		case in:
			b.WriteRune(r)
		}
	}

	return b.String()
}


// Verify looks up domain's record and reports whether it vouches for
// subject: one of its TXT values is exactly ecoa-identity=<subject>.
func Verify(ctx context.Context,
            lookup Lookup,
            domain string,
            subject string) (bool, bool, string) {
	name := recordPrefix + domain

	answer, err := lookup.TXT(ctx, name)
	if err != nil {
		return false, false, "lookup failed: " + err.Error()
	}

	if slices.Contains(answer.Records, valuePrefix+subject) {
		return true, answer.DNSSEC, "found at " + name
	}

	if len(answer.Records) == 0 {
		return false, answer.DNSSEC, "no TXT record at " + name
	}

	return false, answer.DNSSEC, "no record at " + name + " names this identity"
}


// Checker looks up the domain claims due, and records what it found.
type Checker struct {
	queries *sqlc.Queries
	lookup  Lookup
	now     func() time.Time
}


// NewChecker returns a checker writing to pool, resolving through lookup.
func NewChecker(pool *pgxpool.Pool, lookup Lookup) *Checker {
	return &Checker{queries: sqlc.New(pool), lookup: lookup, now: time.Now}
}


// Run checks the claims due, round after round, until ctx ends. A round
// that fails is logged and the next one tries again.
func (c *Checker) Run(ctx context.Context) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		if err := c.Round(ctx); err != nil && ctx.Err() == nil {
			slog.Warn("claim checks", "err", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}


// Round looks up every claim due now, at most a batch of them.
func (c *Checker) Round(ctx context.Context) error {
	now := c.now()

	due, err := c.queries.DueChecks(ctx, sqlc.DueChecksParams{
		Now:           now,
		RecheckBefore: now.Add(-Recheck),
		RetryBefore:   now.Add(-retry),
		Most:          batch,
	})
	if err != nil {
		return err
	}

	for _, d := range due {
		domain := strings.TrimPrefix(d.Entity, "domain:")

		call, cancel := context.WithTimeout(ctx, lookupTimeout)
		verified, dnssec, detail := Verify(call, c.lookup, domain, d.Subject)
		cancel()

		err := c.queries.SaveCheck(ctx, sqlc.SaveCheckParams{
			EventID:   d.EventID,
			Verified:  verified,
			Dnssec:    dnssec,
			CheckedAt: c.now(),
			Detail:    detail,
		})
		if err != nil {
			return err
		}

		slog.Info("claim checked",
		          "eventId", d.EventID,
		          "domain", domain,
		          "verified", verified,
		          "detail", detail)
	}

	return nil
}
