package projection


import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/fixturetest"
)


// --- CODE ---

func TestChainHeadFixtures(t *testing.T) {
	// Arrange
	file := fixturetest.Load(t, "chain-head")

	// Act
	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// Arrange
		events := eventsOf(c)

		// Act
		head := ResolveHead(rootOf(c), events)

		// Assert
		checkHead(t, c, head)
	})
}


func TestVoteAndLabelBindingFixtures(t *testing.T) {
	// Arrange
	file := fixturetest.Load(t, "vote-and-label-binding")

	// Act
	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// Arrange
		events := eventsOf(c)
		root := rootOf(c)
		head := ResolveHead(root, events)
		chain := Chain(root, events)

		// Act
		score := HeadScore(head.ID, events)

		// Assert
		checkVotes(t, c, events, head, score)
		checkLabels(t, c, LabelsOnChain(chain, events))
		checkResponses(t, c, ResponsesOnChain(chain, events))
	})
}


func TestEntityMetadataFixtures(t *testing.T) {
	// Arrange
	file := fixturetest.Load(t, "entity-metadata")

	// Act
	fixturetest.Run(t, file, func(t *testing.T, c fixturetest.Case) {
		// Arrange
		proposals := unlabelled(eventsOf(c))
		identities := identitiesOf(t, c)

		// Act
		fields := ResolveFields(proposals, identities)

		// Assert
		checkFields(t, c, fields)
	})
}


// eventsOf turns the arrival of c into the events the rules read.
func eventsOf(c fixturetest.Case) []Event {
	events := make([]Event, 0, len(c.Arrival))

	for _, a := range c.Arrival {
		events = append(events, Event{
			ID:        a.ID,
			Author:    a.Author,
			Type:      a.Type,
			CreatedAt: a.At,
			Payload:   a.Payload,
		})
	}

	return events
}


// rootOf returns the id of the review the chain of c starts at.
func rootOf(c fixturetest.Case) string {
	for _, a := range c.Arrival {
		if a.Type == "review.created" {
			return a.ID
		}
	}

	return ""
}


// checkHead compares head with the head, truncated and withdrawn of c.
func checkHead(t *testing.T, c fixturetest.Case, head Head) {
	t.Helper()

	want, _ := c.Expect["head"].(string)
	truncated, _ := c.Expect["truncated"].(bool)
	withdrawn, _ := c.Expect["withdrawn"].(bool)

	if head.ID != want {
		t.Errorf("head = %q, want %q", head.ID, want)
	}

	if head.Truncated != truncated {
		t.Errorf("truncated = %v, want %v", head.Truncated, truncated)
	}

	if head.Withdrawn != withdrawn {
		t.Errorf("withdrawn = %v, want %v", head.Withdrawn, withdrawn)
	}
}


// identitiesOf decodes the identities that c supplies as input.
func identitiesOf(t *testing.T, c fixturetest.Case) map[string]Identity {
	t.Helper()

	identities := map[string]Identity{}

	for name, raw := range c.Identities {
		var identity Identity
		if err := json.Unmarshal(raw, &identity); err != nil {
			t.Fatalf("identity %s: %v", name, err)
		}

		identities[name] = identity
	}

	return identities
}


// unlabelled keeps the entity.updated events no label points at.
func unlabelled(events []Event) []Event {
	labelled := map[string]bool{}

	for _, e := range events {
		if e.Type == "moderation.label_created" {
			labelled[e.Field("target")] = true
		}
	}

	var kept []Event

	for _, e := range events {
		if e.Type == "entity.updated" && !labelled[e.ID] {
			kept = append(kept, e)
		}
	}

	return kept
}


// checkFields compares fields with the canonical values and tier of c.
func checkFields(t *testing.T, c fixturetest.Case, fields map[string]Field) {
	t.Helper()

	canonical, _ := c.Expect["canonical"].(map[string]any)
	tier, hasTier := c.Expect["tier"].(float64)

	if len(fields) != len(canonical) {
		t.Errorf("fields = %+v, want %d of them", fields, len(canonical))
	}

	for name, want := range canonical {
		got := fields[name]
		if got.Value != want {
			t.Errorf("%s = %q, want %q", name, got.Value, want)
		}

		if hasTier && float64(got.Tier) != tier {
			t.Errorf("%s tier = %d, want %v", name, got.Tier, tier)
		}
	}
}


// checkVotes compares the head, its score and the old versions' votes of c.
func checkVotes(
	t *testing.T, c fixturetest.Case, events []Event, head Head, score int,
) {
	t.Helper()

	if want, ok := c.Expect["head"].(string); ok && head.ID != want {
		t.Errorf("head = %q, want %q", head.ID, want)
	}

	if want, ok := c.Expect["headScore"].(map[string]any); ok {
		endorsed := authorsOf(want["endorsed"])
		disputed := authorsOf(want["disputed"])

		checkVoters(t, "head", events, head.ID, endorsed, disputed)

		if score != len(endorsed)-len(disputed) {
			t.Errorf("score = %d, want %d", score, len(endorsed)-len(disputed))
		}
	}

	historical, _ := c.Expect["historical"].(map[string]any)

	for id, versions := range historical {
		want, _ := versions.(map[string]any)

		checkVoters(t, id, events, id, authorsOf(want["endorsed"]), nil)
	}

	checkCountedVote(t, c, events, head)
}


// checkVoters compares who currently endorses and disputes target.
func checkVoters(
	t *testing.T, name string, events []Event, target string,
	endorsed, disputed []string,
) {
	t.Helper()

	var gotEndorsed, gotDisputed []string

	for _, vote := range currentVotes(target, events) {
		if vote.Type == typeEndorsed {
			gotEndorsed = append(gotEndorsed, vote.Author)
		} else {
			gotDisputed = append(gotDisputed, vote.Author)
		}
	}

	if !slices.Equal(gotEndorsed, endorsed) {
		t.Errorf("%s endorsed = %v, want %v", name, gotEndorsed, endorsed)
	}

	if !slices.Equal(gotDisputed, disputed) {
		t.Errorf("%s disputed = %v, want %v", name, gotDisputed, disputed)
	}
}


// checkCountedVote compares the vote that counts for bob, when c names it.
func checkCountedVote(
	t *testing.T, c fixturetest.Case, events []Event, head Head,
) {
	t.Helper()

	want, ok := c.Expect["countedVote"].(string)
	if !ok {
		return
	}

	got, found := CurrentVote("bob", head.ID, events)
	if !found || got.ID != want {
		t.Errorf("current vote = %q, want %q", got.ID, want)
	}
}


// authorsOf returns the strings of a list decoded from an expect.
func authorsOf(v any) []string {
	var authors []string

	list, _ := v.([]any)

	for _, item := range list {
		author, _ := item.(string)
		authors = append(authors, author)
	}

	return authors
}


// checkLabels compares labels with the labelsOnHead of c, when it names them.
func checkLabels(t *testing.T, c fixturetest.Case, labels []Event) {
	t.Helper()

	want, ok := c.Expect["labelsOnHead"].([]any)
	if !ok {
		return
	}

	if len(labels) != len(want) {
		t.Fatalf("labels = %+v, want %d of them", labels, len(want))
	}

	for i, item := range want {
		label, _ := item.(map[string]any)

		if labels[i].ID != label["id"] ||
			labels[i].Field("label") != label["label"] ||
			labels[i].Author != label["by"] {
			t.Errorf("label %d = %+v, want %v", i, labels[i], label)
		}
	}
}


// checkResponses compares responses with the responsesOnHead of c.
func checkResponses(t *testing.T, c fixturetest.Case, responses []Event) {
	t.Helper()

	want, ok := c.Expect["responsesOnHead"].([]any)
	if !ok {
		return
	}

	var got []string

	for _, response := range responses {
		got = append(got, response.ID)
	}

	if !slices.Equal(got, authorsOf(want)) {
		t.Errorf("responses = %v, want %v", got, want)
	}
}
