package projection


import (
	"cmp"
	"maps"
	"slices"
)


// --- CONSTANTS ---

// typeEndorsed is the event type of a vote for a review.
const typeEndorsed = "review.endorsed"


// typeDisputed is the event type of a vote against a review.
const typeDisputed = "review.disputed"


// --- CODE ---

// CurrentVote is the latest vote voter cast on target, by the author's order.
func CurrentVote(voter, target string, votes []Event) (Event, bool) {
	vote, found := latestVotes(target, votes)[voter]

	return vote, found
}


// HeadScore is endorsements minus disputes among the current votes on head.
func HeadScore(head string, votes []Event) int {
	score := 0

	for _, vote := range currentVotes(head, votes) {
		if vote.Type == typeEndorsed {
			score++
		} else {
			score--
		}
	}

	return score
}


// currentVotes returns the current vote of each voter on target, by author.
func currentVotes(target string, votes []Event) []Event {
	latest := latestVotes(target, votes)

	return slices.SortedFunc(maps.Values(latest), func(a, b Event) int {
		return cmp.Compare(a.Author, b.Author)
	})
}


// latestVotes maps each voter to the latest vote they cast on target.
func latestVotes(target string, votes []Event) map[string]Event {
	latest := map[string]Event{}

	for _, vote := range votes {
		if vote.Type != typeEndorsed && vote.Type != typeDisputed {
			continue
		}

		if vote.Field("target") != target {
			continue
		}

		held, found := latest[vote.Author]
		if !found || vote.Outranks(held) {
			latest[vote.Author] = vote
		}
	}

	return latest
}
