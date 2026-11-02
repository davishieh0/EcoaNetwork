package config


import (
	"os"
	"reflect"
	"testing"
)


// --- GLOBALS ---

// names lists every variable Load reads.
var names = []string{
	"NETWORK",
	"LISTEN_ADDR",
	"GRPC_ADDR",
	"BOOTSTRAP_ADDRS",
	"BOLT_DB_PATH",
}


// --- CODE ---

// clearEnv unsets every variable Load reads for the length of the test.
func clearEnv(t *testing.T) {
	t.Helper()

	for _, name := range names {
		t.Setenv(name, "")

		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}
}


func TestLoadAppliesTheDefaultsToAnEmptyEnvironment(t *testing.T) {
	// Arrange
	clearEnv(t)

	want := Config{
		Network:    "ecoa-dev",
		ListenAddr: "/ip4/127.0.0.1/tcp/4001",
		GRPCAddr:   "127.0.0.1:50051",
		BoltDBPath: "data/bbolt/bolt.db",
	}

	// Act
	got, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}
}


func TestLoadTreatsAnEmptyVariableAsUnset(t *testing.T) {
	// Arrange
	clearEnv(t)
	t.Setenv("NETWORK", "")

	// Act
	got, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Network != "ecoa-dev" {
		t.Fatalf("Network = %q, want the default", got.Network)
	}
}


func TestLoadReadsEachVariable(t *testing.T) {
	cases := []struct {
		name  string
		value string
		read  func(Config) any
		want  any
	}{
		{
			"NETWORK", "ecoa-x",
			func(c Config) any { return c.Network }, "ecoa-x",
		},
		{
			"LISTEN_ADDR", "/ip4/0.0.0.0/tcp/1",
			func(c Config) any { return c.ListenAddr },
			"/ip4/0.0.0.0/tcp/1",
		},
		{
			"GRPC_ADDR", "0.0.0.0:1",
			func(c Config) any { return c.GRPCAddr }, "0.0.0.0:1",
		},
		{
			"BOLT_DB_PATH", "x.db",
			func(c Config) any { return c.BoltDBPath }, "x.db",
		},
		{
			"BOOTSTRAP_ADDRS", "/ip4/1.1.1.1/tcp/1,/ip4/2.2.2.2/tcp/2",
			func(c Config) any { return c.BootstrapAddrs },
			[]string{"/ip4/1.1.1.1/tcp/1", "/ip4/2.2.2.2/tcp/2"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Arrange
			clearEnv(t)
			t.Setenv(c.name, c.value)

			// Act
			got, err := Load()

			// Assert
			if err != nil {
				t.Fatalf("Load: %v", err)
			}

			if !reflect.DeepEqual(c.read(got), c.want) {
				t.Fatalf("%s = %v, want %v", c.name, c.read(got), c.want)
			}
		})
	}
}
