package main


import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/eduardokairalla1/EcoaNetwork/internal/config"
	"github.com/eduardokairalla1/EcoaNetwork/internal/store"
)


// --- CODE ---

func TestRunRefusesArguments(t *testing.T) {
	// Arrange
	args := []string{"-network", "x"}
	path := filepath.Join(t.TempDir(), "bolt.db")
	t.Setenv("BOLT_DB_PATH", path)

	// Act
	err := run(args)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("run(%q) = %v, want an unexpected arguments error", args, err)
	}

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("store at %s: %v, want it never opened", path, err)
	}
}


func TestRunRefusesANetworkOutsideTheProtocolPattern(t *testing.T) {
	// Arrange
	t.Setenv("NETWORK", "foo")

	// Act
	err := run(nil)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "NETWORK") {
		t.Fatalf("run = %v, want a NETWORK error", err)
	}
}





func TestPeerConfigDialsTheLastSessionAndRemembersThisOne(t *testing.T) {
	// Arrange
	db, err := store.New(filepath.Join(t.TempDir(), "bolt.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer db.Close()

	if err := db.AddPeer("/ip4/1.1.1.1/tcp/1"); err != nil {
		t.Fatalf("AddPeer: %v", err)
	}

	// Act
	peers, err := peerConfig(config.Config{Network: "ecoa-test"}, db)
	if err != nil {
		t.Fatalf("peerConfig: %v", err)
	}

	peers.OnDialed("/ip4/2.2.2.2/tcp/2")

	// Assert
	known, err := db.KnownPeers()
	want := []string{"/ip4/1.1.1.1/tcp/1", "/ip4/2.2.2.2/tcp/2"}
	if err != nil || !reflect.DeepEqual(known, want) {
		t.Fatalf("KnownPeers = %v, %v, want %v", known, err, want)
	}

	if !reflect.DeepEqual(peers.KnownAddrs, want[:1]) {
		t.Fatalf("KnownAddrs = %v, want %v", peers.KnownAddrs, want[:1])
	}
}
