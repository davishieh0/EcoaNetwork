// Command seed generates the MVP's seeded dataset and publishes it to a
// node: entities from the Receita Federal's public CNPJ data, named by an
// operator's entity.updated proposals, and a few hundred properly signed
// synthetic reviews across them, with edits, votes, responses and
// withdrawals.
//
// The same flags generate the same events, byte for byte, so running it
// again publishes nothing new.
package main


import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/seed"
	"github.com/eduardokairalla1/EcoaNetwork/indexer/internal/seed/sample"
)


// --- GLOBALS ---

// the most entities one run proposes: a node admits 100 events per author
// per hour, and the operator signs every proposal
const mostCompanies = 90


// --- CODE ---

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}


// run reads the flags, generates the events and publishes them.
func run(args []string) error {
	flags := flag.NewFlagSet("seed", flag.ContinueOnError)
	node := flags.String("node", "localhost:50051",
	                     "the node to publish to, host:port of its gRPC API")
	estabelecimentos := flags.String("estabelecimentos", "",
	                                 "a Receita Estabelecimentos file; "+
	                                 "the synthetic sample when empty")
	empresas := flags.String("empresas", "",
	                         "the Receita Empresas file, for legal names")
	companies := flags.Int("companies", 60, "how many entities, at most 90")
	reviews := flags.Int("reviews", 400, "how many reviews")
	reviewers := flags.Int("reviewers", 80, "how many synthetic identities")
	days := flags.Int("days", 90, "the days the reviews spread over")
	until := flags.String("until", "",
	                      "the latest createdAt, RFC 3339; today 00:00 UTC "+
	                      "when empty, so a rerun the same day repeats it")
	seedValue := flags.Int64("seed", 1, "picks every choice and every key")
	operator := flags.String("operator-key", "",
	                         "hex seed of the key signing the entities; "+
	                         "derived from -seed when empty, for "+
	                         "development only")
	network := flags.String("network", "ecoa-dev", "the protocol field")
	dryRun := flags.Bool("dry-run", false,
	                     "generate and count, publish nothing")

	if err := flags.Parse(args); err != nil {
		return err
	}

	if *companies < 1 || *companies > mostCompanies {
		return fmt.Errorf("-companies: %d is not between 1 and %d",
		                  *companies, mostCompanies)
	}

	end := time.Now().UTC().Truncate(24 * time.Hour)
	if *until != "" {
		parsed, err := time.Parse(time.RFC3339, *until)
		if err != nil {
			return fmt.Errorf("-until: %w", err)
		}
		end = parsed
	}

	key, err := operatorKey(*operator, *seedValue)
	if err != nil {
		return err
	}

	list, err := readCompanies(*estabelecimentos, *empresas, *companies)
	if err != nil {
		return err
	}

	generated, err := seed.Generate(seed.Plan{
		Companies: list,
		Reviewers: *reviewers,
		Reviews:   *reviews,
		Until:     end,
		Days:      *days,
		Seed:      *seedValue,
		Operator:  key,
		Network:   *network,
	})
	if err != nil {
		return err
	}

	fmt.Printf("generated %d events for %d entities, operator %s\n",
	           len(generated.Envelopes), len(list), seed.DidOf(key))
	printCounts(generated.Counts)

	if *dryRun {
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	published, err := seed.Publish(ctx, *node, generated.Envelopes)
	printCounts(published.Outcomes)

	for _, reason := range published.Refused {
		fmt.Println("  refused:", reason)
	}

	return err
}


// operatorKey is the key signing the entities' proposals.
func operatorKey(hexSeed string, seedValue int64) (ed25519.PrivateKey, error) {
	if hexSeed == "" {
		return seed.KeyOf(seedValue, "operator", 0), nil
	}

	raw, err := hex.DecodeString(hexSeed)
	if err != nil || len(raw) != ed25519.SeedSize {
		return nil, fmt.Errorf("-operator-key: want %d bytes in hex",
		                       ed25519.SeedSize)
	}

	return ed25519.NewKeyFromSeed(raw), nil
}


// readCompanies reads the Receita files, or the synthetic sample.
func readCompanies(estPath, empPath string, limit int) ([]seed.Company, error) {
	if estPath == "" {
		return seed.ReadReceita(sample.Estabelecimentos(),
		                        sample.Empresas(),
		                        limit)
	}

	est, err := os.Open(estPath)
	if err != nil {
		return nil, err
	}
	defer est.Close()

	var emp io.Reader
	if empPath != "" {
		file, err := os.Open(empPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		emp = file
	}

	return seed.ReadReceita(est, emp, limit)
}


// printCounts prints counts by name, in order.
func printCounts(counts map[string]int) {
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Printf("  %-26s %d\n", name, counts[name])
	}
}
