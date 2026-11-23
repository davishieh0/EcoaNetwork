package projector


import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/label"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/profile"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/envelope"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/projection"
	"github.com/jackc/pgx/v5/pgxpool"
)


// --- CODE ---

// profilesAPI answers path through the profile routes, following policy.
func profilesAPI(t *testing.T,
                 pool *pgxpool.Pool,
                 policy label.Policy,
                 path string) *httptest.ResponseRecorder {
	t.Helper()

	h := profile.NewHandler(profile.NewRepository(pool),
	                        label.NewModeration(policy, pool))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /profiles", h.List)
	mux.HandleFunc("GET /profiles/{did}", h.Get)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

	return rec
}


// profileOf is a profile.updated by the key of seed, written at.
func profileOf(t *testing.T, seed byte, at, payload string) ([]byte, string) {
	t.Helper()

	return signAs(t, seed, "profile.updated", at, payload)
}


// displayNames decodes a profiles answer, one item or a list, into each
// item's identity and displayName.
func displayNames(t *testing.T, rec *httptest.ResponseRecorder) ([]string, []string) {
	t.Helper()

	type item struct {
		Envelope string `json:"envelope"`
		Derived  struct {
			Identity string `json:"identity"`
		} `json:"derived"`
	}

	var a struct{ Items []item }
	var one item

	if strings.Contains(rec.Body.String(), `"items"`) {
		json.Unmarshal(rec.Body.Bytes(), &a)
	} else {
		json.Unmarshal(rec.Body.Bytes(), &one)
		a.Items = []item{one}
	}

	var dids, names []string

	for _, it := range a.Items {
		var e struct {
			Payload struct{ DisplayName string } `json:"payload"`
		}

		json.Unmarshal([]byte(it.Envelope), &e)
		dids = append(dids, it.Derived.Identity)
		names = append(names, e.Payload.DisplayName)
	}

	return dids, names
}


func TestTheLatestProfileHoldsWhateverOrderTheyArrived(t *testing.T) {
	// Arrange: the newer profile arrives first, the older one later
	pool := migrated(t)

	newer, _ := profileOf(t, 'b', "2026-11-02T10:00:00Z", `{"displayName":"new"}`)
	older, _ := profileOf(t, 'b', "2026-11-01T10:00:00Z", `{"displayName":"old"}`)

	// Act
	feed(t, pool, "node-a", newer, older)

	// Assert
	_, names := displayNames(t, profilesAPI(t, pool, label.Policy{},
		"/profiles/"+didOf('b')))

	if !slices.Equal(names, []string{"new"}) {
		t.Fatalf("profile = %v, want new", names)
	}
}


func TestATieOnCreatedAtBreaksAsTheProtocolSays(t *testing.T) {
	// Arrange: two profiles in the same second
	pool := migrated(t)

	a, _ := profileOf(t, 'b', "2026-11-01T10:00:00Z", `{"displayName":"a"}`)
	b, _ := profileOf(t, 'b', "2026-11-01T10:00:00Z", `{"displayName":"b"}`)

	envA, _ := envelope.Decode(a)
	envB, _ := envelope.Decode(b)

	want := "a"
	if eventOf(envB).Outranks(eventOf(envA)) {
		want = "b"
	}

	// Act: in both orders
	feed(t, pool, "node-a", a, b)
	_, first := displayNames(t, profilesAPI(t, pool, label.Policy{},
		"/profiles/"+didOf('b')))

	other := migrated(t)
	feed(t, other, "node-a", b, a)
	_, second := displayNames(t, profilesAPI(t, other, label.Policy{},
		"/profiles/"+didOf('b')))

	// Assert
	if first[0] != want || second[0] != want {
		t.Fatalf("holds %q and %q, the rule says %q", first[0], second[0], want)
	}
}


func TestANewProfileReplacesEveryField(t *testing.T) {
	// Arrange: a bio, then a profile without one
	pool := migrated(t)

	first, _ := profileOf(t, 'b', "2026-11-01T10:00:00Z",
		`{"displayName":"bia","bio":"padeira"}`)
	second, _ := profileOf(t, 'b', "2026-11-02T10:00:00Z", `{"displayName":"bia"}`)

	// Act
	feed(t, pool, "node-a", first, second)

	// Assert
	var bio *string

	pool.QueryRow(context.Background(),
		"SELECT bio FROM profiles WHERE identity = $1", didOf('b')).Scan(&bio)

	if bio != nil {
		t.Fatalf("bio = %q, want it cleared", *bio)
	}
}


func TestTheProfileRouteAnswers(t *testing.T) {
	// Arrange
	pool := migrated(t)
	raw, _ := profileOf(t, 'b', "2026-11-01T10:00:00Z", `{"displayName":"bia"}`)

	feed(t, pool, "node-a", raw)
	feed(t, pool, "node-b", raw)

	// Act
	found := profilesAPI(t, pool, label.Policy{}, "/profiles/"+didOf('b'))
	none := profilesAPI(t, pool, label.Policy{}, "/profiles/"+didOf('c'))
	bad := profilesAPI(t, pool, label.Policy{}, "/profiles/someone")

	// Assert
	var item struct{ Envelope string }
	json.Unmarshal(found.Body.Bytes(), &item)

	if found.Code != http.StatusOK || !bytes.Equal([]byte(item.Envelope), raw) {
		t.Fatalf("found = %d, envelope %q", found.Code, item.Envelope)
	}

	if none.Code != http.StatusNotFound || bad.Code != http.StatusBadRequest {
		t.Fatalf("none = %d, bad = %d", none.Code, bad.Code)
	}

	if count(t, pool, "profiles") != 1 {
		t.Fatalf("profiles = %d, want one from two nodes", count(t, pool, "profiles"))
	}
}


func TestManyProfilesComeInOneCallInTheOrderAsked(t *testing.T) {
	// Arrange: b and d have profiles, c has none
	pool := migrated(t)

	b, _ := profileOf(t, 'b', "2026-11-01T10:00:00Z", `{"displayName":"bia"}`)
	d, _ := profileOf(t, 'd', "2026-11-01T10:00:00Z", `{"displayName":"davi"}`)
	feed(t, pool, "node-a", b, d)

	// Act
	rec := profilesAPI(t, pool, label.Policy{},
		"/profiles?ids="+didOf('d')+","+didOf('c')+","+didOf('b'))

	// Assert
	dids, names := displayNames(t, rec)

	if rec.Code != http.StatusOK ||
	   !slices.Equal(dids, []string{didOf('d'), didOf('b')}) ||
	   !slices.Equal(names, []string{"davi", "bia"}) {
		t.Fatalf("GET = %d %v %v", rec.Code, dids, names)
	}

	tooMany := strings.TrimSuffix(strings.Repeat(didOf('b')+",", 101), ",")

	for _, query := range []string{"", "ids=", "ids=someone", "ids=" + tooMany} {
		if rec := profilesAPI(t, pool, label.Policy{}, "/profiles?"+query); rec.Code != http.StatusBadRequest {
			t.Errorf("%.40s = %d, want 400", query, rec.Code)
		}
	}
}


func TestLabelsOnAProfileStayWithThatProfile(t *testing.T) {
	// Arrange: b's profile labeled impersonation, c's labeled illegal
	pool := migrated(t)

	b, bID := profileOf(t, 'b', "2026-11-01T10:00:00Z", `{"displayName":"Banco Oficial"}`)
	c, cID := profileOf(t, 'c', "2026-11-01T10:00:00Z", `{"displayName":"bad"}`)
	l1, _ := labelOf(t, 'm', bID, "impersonation")
	l2, _ := labelOf(t, 'm', cID, "illegal")

	feed(t, pool, "node-a", b, c, l1, l2)

	following := label.Follow(didOf('m'))

	// Act and Assert: marked
	var marked struct {
		Observed struct{ Labels []label.Applied } `json:"observed"`
	}

	json.Unmarshal(profilesAPI(t, pool, following, "/profiles/"+didOf('b')).Body.Bytes(),
	               &marked)

	if len(marked.Observed.Labels) != 1 || marked.Observed.Labels[0].Label != "impersonation" {
		t.Fatalf("labels = %+v, want impersonation", marked.Observed.Labels)
	}

	// removed: 451 alone, left out of a list
	if rec := profilesAPI(t, pool, following, "/profiles/"+didOf('c')); rec.Code != http.StatusUnavailableForLegalReasons {
		t.Fatalf("removed profile = %d, want 451", rec.Code)
	}

	dids, _ := displayNames(t, profilesAPI(t, pool, following,
		"/profiles?ids="+didOf('b')+","+didOf('c')))

	if !slices.Equal(dids, []string{didOf('b')}) {
		t.Fatalf("list = %v, want only b", dids)
	}

	// a new profile is a new statement: it does not inherit the label
	again, _ := profileOf(t, 'c', "2026-11-02T10:00:00Z", `{"displayName":"fixed"}`)
	feed(t, pool, "node-b", again)

	if rec := profilesAPI(t, pool, following, "/profiles/"+didOf('c')); rec.Code != http.StatusOK {
		t.Fatalf("new profile = %d, want 200", rec.Code)
	}
}


// eventOf is env as the protocol's rules read it.
func eventOf(env *envelope.Envelope) projection.Event {
	return projection.Event{ID: env.EventID, Author: env.Author,
	                        Type: env.EventType, CreatedAt: env.CreatedAt,
	                        Payload: env.Payload}
}
