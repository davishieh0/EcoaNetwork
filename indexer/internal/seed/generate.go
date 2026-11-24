package seed


import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/identity"
	"github.com/eduardokairalla1/EcoaNetwork/protocol/go/testevent"
)


// --- GLOBALS ---

// how often each follow-up happens, per review
const (
	editShare     = 0.08
	withdrawShare = 0.02
	respondShare  = 0.14
	replyShare    = 0.35
	titleShare    = 0.45
	endorseShare  = 0.8
	mostVotes     = 5
)

var firstNames = []string{
	"Ana", "Bruno", "Carla", "Diego", "Eduarda", "Felipe", "Gabriela",
	"Henrique", "Isabela", "João", "Karina", "Lucas", "Mariana", "Nicolas",
	"Olívia", "Pedro", "Quésia", "Rafael", "Sofia", "Thiago", "Úrsula",
	"Vitor", "Yasmin", "Zeca",
}

var surnames = []string{
	"Almeida", "Barbosa", "Costa", "Dias", "Esteves", "Ferreira", "Gomes",
	"Lima", "Martins", "Nunes", "Oliveira", "Pereira", "Rocha", "Souza",
}

// sentences by rating, combined two at a time into a body
var sentences = map[int][]string{
	1: {
		"Fui muito mal atendido e ninguém resolveu o problema.",
		"Cobraram um valor diferente do combinado.",
		"Esperei mais de uma hora e desisti.",
		"O produto veio com defeito e a troca foi negada.",
		"Não recomendo, foi uma experiência ruim do começo ao fim.",
	},
	2: {
		"O atendimento foi lento e pouco atencioso.",
		"Achei caro para o que entregam.",
		"Resolveram, mas só depois de muita insistência.",
		"A qualidade caiu bastante desde a última vez.",
	},
	3: {
		"Nada de especial, cumpre o básico.",
		"Atendimento ok, mas o lugar estava cheio.",
		"O preço é justo, a qualidade é mediana.",
		"Algumas coisas boas, outras nem tanto.",
	},
	4: {
		"Gostei bastante, atendimento cordial.",
		"Bom custo-benefício, voltaria.",
		"Resolveram rápido o que eu precisava.",
		"Ambiente agradável e equipe atenciosa.",
	},
	5: {
		"Excelente, superou o que eu esperava.",
		"Atendimento impecável, todos muito educados.",
		"Recomendo de olhos fechados.",
		"Voltarei com certeza, foi ótimo.",
		"Resolveram um problema que ninguém mais resolvia.",
	},
}

var titles = map[int][]string{
	1: {"Experiência péssima", "Não voltaria", "Problema sem solução"},
	2: {"Deixou a desejar", "Esperava mais"},
	3: {"Razoável", "Na média"},
	4: {"Boa experiência", "Gostei"},
	5: {"Excelente", "Recomendo muito", "Atendimento nota 10"},
}

var replies = []string{
	"Passei pela mesma situação na semana passada.",
	"Comigo foi diferente, fui bem atendido.",
	"Obrigado pelo relato, ajudou na minha decisão.",
	"Você chegou a reclamar com a gerência?",
	"Concordo totalmente.",
	"Será que mudaram de dono? Antes era melhor.",
}


// --- CODE ---

// Plan is what to generate.
type Plan struct {
	// Companies become entities, each named by an entity.updated proposal.
	Companies []Company

	// Reviewers is how many synthetic identities write the reviews.
	Reviewers int

	// Reviews is how many reviews they write, across the companies.
	Reviews int

	// Until is the latest createdAt; everything happens in the Days before.
	Until time.Time
	Days  int

	// Seed picks every random choice and every reviewer's key: the same
	// plan generates the same events, byte for byte.
	Seed int64

	// Operator signs the entities' proposals, as anyone may.
	Operator ed25519.PrivateKey

	// Network is the protocol field, such as ecoa-dev.
	Network string
}


// Generated is what a plan gives: envelopes in the order to publish them,
// so no reference arrives before its target.
type Generated struct {
	Envelopes [][]byte
	Counts    map[string]int
}


// event is one envelope waiting to be signed, at its time.
type event struct {
	key     ed25519.PrivateKey
	kind    string
	at      time.Time
	payload map[string]any
	// the round it publishes in: references come in a later round
	round int
}


// review is a review chain as the generator tracks it.
type review struct {
	author   int
	versions []string
	at       time.Time
	rating   int
}


// KeyOf returns the key of reviewer i of seed, derived so it is the same on
// every run.
func KeyOf(seed int64, who string, i int) ed25519.PrivateKey {
	digest := sha256.Sum256([]byte(fmt.Sprintf("ecoa-seed/%d/%s/%d",
	                                          seed, who, i)))

	return ed25519.NewKeyFromSeed(digest[:])
}


// Generate builds and signs every event of p.
func Generate(p Plan) (Generated, error) {
	random := rand.New(rand.NewSource(p.Seed))
	start := p.Until.Add(-time.Duration(p.Days) * 24 * time.Hour)
	span := p.Until.Sub(start)

	// a moment after t, before Until
	after := func(t time.Time) time.Time {
		left := p.Until.Sub(t)
		if left <= time.Minute {
			return p.Until
		}

		return t.Add(time.Duration(random.Int63n(int64(left))))
	}

	var events []event

	// the entities, named before anyone reviews them
	for i, c := range p.Companies {
		payload := map[string]any{
			"entity":   "cnpj:" + c.CNPJ,
			"name":     c.Name,
			"category": CategoryOf(c.CNAE),
		}

		if c.Since != "" || c.District != "" {
			payload["description"] = describe(c)
		}

		events = append(events, event{
			key:     p.Operator,
			kind:    "entity.updated",
			at:      start.Add(time.Duration(i) * time.Second),
			payload: payload,
		})
	}

	// the reviewers' profiles
	keys := make([]ed25519.PrivateKey, p.Reviewers)
	for i := range keys {
		keys[i] = KeyOf(p.Seed, "reviewer", i)
		name := fmt.Sprintf("%s %s.",
		                    firstNames[random.Intn(len(firstNames))],
		                    surnames[random.Intn(len(surnames))][:1])

		events = append(events, event{
			key:     keys[i],
			kind:    "profile.updated",
			at:      start.Add(time.Duration(i) * time.Second),
			payload: map[string]any{"displayName": name, "lang": "pt-BR"},
		})
	}

	// some companies draw many reviews, most draw a few
	weights := make([]float64, len(p.Companies))
	order := random.Perm(len(p.Companies))
	total := 0.0
	for rank, i := range order {
		weights[i] = 1 / math.Pow(float64(rank+1), 0.8)
		total += weights[i]
	}

	pick := func() int {
		x := random.Float64() * total
		for i, w := range weights {
			if x -= w; x <= 0 {
				return i
			}
		}

		return len(weights) - 1
	}

	// the reviews, then what happens to them
	var chains []*review
	for range p.Reviews {
		company := p.Companies[pick()]
		rating := 1 + random.Intn(5)
		author := random.Intn(p.Reviewers)
		at := start.Add(time.Duration(random.Int63n(int64(span))))

		payload := text(random, rating)
		payload["entity"] = "cnpj:" + company.CNPJ

		events = append(events, event{
			key: keys[author], kind: "review.created", at: at,
			payload: payload, round: 1,
		})
		chains = append(chains, &review{author: author, at: at, rating: rating})
	}

	// sign the reviews now: what follows names their eventIds
	signed, err := sign(p, events)
	if err != nil {
		return Generated{}, err
	}

	for i, ch := range chains {
		ch.versions = []string{signed.ids[len(p.Companies)+p.Reviewers+i]}
	}

	var later []event

	for _, ch := range chains {
		// a vote on the original, before any edit: it stays there
		votes := random.Intn(mostVotes + 1)
		voted := map[int]bool{ch.author: true}

		for range votes {
			voter := random.Intn(p.Reviewers)
			if voted[voter] {
				continue
			}
			voted[voter] = true

			kind := "review.endorsed"
			if random.Float64() > endorseShare {
				kind = "review.disputed"
			}

			later = append(later, event{
				key: keys[voter], kind: kind, at: after(ch.at),
				payload: map[string]any{"target": ch.versions[0]}, round: 2,
			})
		}

		if random.Float64() < respondShare {
			who := random.Intn(p.Reviewers)
			later = append(later, event{
				key:  keys[who],
				kind: "review.response_created",
				at:   after(ch.at),
				payload: map[string]any{
					"target": ch.versions[0],
					"body":   replies[random.Intn(len(replies))],
					"lang":   "pt-BR",
				},
				round: 2,
			})
		}

		switch x := random.Float64(); {
		case x < withdrawShare:
			later = append(later, event{
				key: keys[ch.author], kind: "review.withdrawn",
				at: after(ch.at), payload: map[string]any{
					"target": ch.versions[0], "reason": "resolved",
				},
				round: 3,
			})
		case x < withdrawShare+editShare:
			rating := min(5, ch.rating+1+random.Intn(2))
			payload := text(random, rating)
			payload["replaces"] = ch.versions[0]

			later = append(later, event{
				key: keys[ch.author], kind: "review.replaced",
				at: after(ch.at), payload: payload, round: 3,
			})
		}
	}

	second, err := sign(p, later)
	if err != nil {
		return Generated{}, err
	}

	// responses to responses, a round after the responses they answer
	var replyEvents []event
	for i, e := range later {
		if e.kind != "review.response_created" ||
			random.Float64() >= replyShare {
			continue
		}

		who := random.Intn(p.Reviewers)
		replyEvents = append(replyEvents, event{
			key: keys[who], kind: "review.response_created", at: after(e.at),
			payload: map[string]any{
				"target": second.ids[i],
				"body":   replies[random.Intn(len(replies))],
				"lang":   "pt-BR",
			},
			round: 4,
		})
	}

	third, err := sign(p, replyEvents)
	if err != nil {
		return Generated{}, err
	}

	all := append(append(signed.items, second.items...), third.items...)

	// each round in its own time order, rounds one after the other
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].round != all[j].round {
			return all[i].round < all[j].round
		}

		return all[i].at.Before(all[j].at)
	})

	out := Generated{Counts: map[string]int{}}
	for _, s := range all {
		out.Envelopes = append(out.Envelopes, s.envelope)
		out.Counts[s.kind]++
	}

	return out, nil
}


// signedSet is a batch of events signed, with their eventIds in order.
type signedSet struct {
	items []signedItem
	ids   []string
}


// signedItem is one event signed, with what sorts it.
type signedItem struct {
	envelope []byte
	kind     string
	at       time.Time
	round    int
}


// sign signs every event of events, in order.
func sign(p Plan, events []event) (signedSet, error) {
	set := signedSet{}

	for _, e := range events {
		payload, err := json.Marshal(e.payload)
		if err != nil {
			return signedSet{}, err
		}

		raw, err := testevent.Sign(e.key, testevent.Fields{
			Protocol:  p.Network,
			Version:   1,
			EventType: e.kind,
			CreatedAt: e.at.UTC().Truncate(time.Second).Format(time.RFC3339),
			Payload:   string(payload),
		})
		if err != nil {
			return signedSet{}, fmt.Errorf("sign %s: %w", e.kind, err)
		}

		var head struct {
			EventID string `json:"eventId"`
		}
		if err := json.Unmarshal(raw, &head); err != nil {
			return signedSet{}, err
		}

		set.items = append(set.items, signedItem{
			envelope: raw, kind: e.kind, at: e.at, round: e.round,
		})
		set.ids = append(set.ids, head.EventID)
	}

	return set, nil
}


// text is the rating, body and sometimes title of a review of rating.
func text(random *rand.Rand, rating int) map[string]any {
	pool := sentences[rating]
	first := random.Intn(len(pool))
	second := (first + 1 + random.Intn(len(pool)-1)) % len(pool)

	payload := map[string]any{
		"rating": rating,
		"body":   pool[first] + " " + pool[second],
		"lang":   "pt-BR",
	}

	if random.Float64() < titleShare {
		options := titles[rating]
		payload["title"] = options[random.Intn(len(options))]
	}

	return payload
}


// describe is an entity's description from what the Receita says of it.
func describe(c Company) string {
	where := c.District
	if c.State != "" {
		if where != "" {
			where += ", "
		}
		where += c.State
	}

	switch {
	case c.Since != "" && where != "":
		return "Aberta em " + c.Since + " · " + where
	case c.Since != "":
		return "Aberta em " + c.Since
	}

	return where
}


// DidOf is the did:key a key signs as.
func DidOf(key ed25519.PrivateKey) string {
	public, _ := key.Public().(ed25519.PublicKey)

	return identity.FromKey(public).String()
}
