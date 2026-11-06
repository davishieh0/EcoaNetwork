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
	"BOOTSTRAP_EXTRA_ADDRS",
	"BOLT_DB_PATH",
	"PEER_KEY_PATH",
	"CONN_LOW",
	"CONN_HIGH",
	"MAX_EVENT_SIZE",
	"ADMISSION_RATE",
	"SYNC_MAX_STREAMS",
	"SYNC_MAX_STREAMS_PER_PEER",
	"SYNC_EVENTS_PER_PEER",
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
		Network:       "ecoa-dev",
		ListenAddr:    "/ip4/127.0.0.1/tcp/4001",
		GRPCAddr:      "127.0.0.1:50051",
		BoltDBPath:    "data/bbolt/bolt.db",
		PeerKeyPath:   "data/peer.key",
		ConnLow:       32,
		ConnHigh:      96,
		MaxEventSize:  16384,
		AdmissionRate: 100,

		SyncMaxStreams:        32,
		SyncMaxStreamsPerPeer: 2,
		SyncEventsPerPeer:     4096,
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
			"PEER_KEY_PATH", "x.key",
			func(c Config) any { return c.PeerKeyPath }, "x.key",
		},
		{
			"CONN_LOW", "8",
			func(c Config) any { return c.ConnLow }, 8,
		},
		{
			"CONN_HIGH", "64",
			func(c Config) any { return c.ConnHigh }, 64,
		},
		{
			"MAX_EVENT_SIZE", "4096",
			func(c Config) any { return c.MaxEventSize }, 4096,
		},
		{
			"ADMISSION_RATE", "5",
			func(c Config) any { return c.AdmissionRate }, 5,
		},
		{
			"ADMISSION_RATE", "0",
			func(c Config) any { return c.AdmissionRate }, 0,
		},
		{
			"SYNC_MAX_STREAMS", "5",
			func(c Config) any { return c.SyncMaxStreams }, 5,
		},
		{
			"SYNC_MAX_STREAMS_PER_PEER", "3",
			func(c Config) any { return c.SyncMaxStreamsPerPeer }, 3,
		},
		{
			"SYNC_EVENTS_PER_PEER", "100",
			func(c Config) any { return c.SyncEventsPerPeer }, 100,
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


func TestLoadRefusesANetworkOutsideTheProtocolPattern(t *testing.T) {
	for _, value := range []string{"foo", "ecoa-", "ECOA", "ecoa_dev"} {
		t.Run(value, func(t *testing.T) {
			// Arrange
			clearEnv(t)
			t.Setenv("NETWORK", value)

			// Act
			_, err := Load()

			// Assert
			if err == nil {
				t.Fatalf("Load accepted NETWORK=%s", value)
			}
		})
	}
}


func TestLoadAcceptsTheEdgesOfMaxEventSize(t *testing.T) {
	for value, want := range map[string]int{"1": 1, "16384": 16384} {
		t.Run(value, func(t *testing.T) {
			// Arrange
			clearEnv(t)
			t.Setenv("MAX_EVENT_SIZE", value)

			// Act
			got, err := Load()

			// Assert
			if err != nil || got.MaxEventSize != want {
				t.Fatalf("Load = %d, %v, want %d", got.MaxEventSize, err, want)
			}
		})
	}
}


func TestLoadRefusesAnInvalidMaxEventSize(t *testing.T) {
	for _, value := range []string{"16385", "0", "-1", "big"} {
		t.Run(value, func(t *testing.T) {
			// Arrange
			clearEnv(t)
			t.Setenv("MAX_EVENT_SIZE", value)

			// Act
			_, err := Load()

			// Assert
			if err == nil {
				t.Fatalf("Load accepted MAX_EVENT_SIZE=%s", value)
			}
		})
	}
}


func TestLoadRefusesAnInvalidAdmissionRate(t *testing.T) {
	for _, value := range []string{"-1", "many"} {
		t.Run(value, func(t *testing.T) {
			// Arrange
			clearEnv(t)
			t.Setenv("ADMISSION_RATE", value)

			// Act
			_, err := Load()

			// Assert
			if err == nil {
				t.Fatalf("Load accepted ADMISSION_RATE=%s", value)
			}
		})
	}
}


func TestLoadRefusesAnInvalidSyncLimit(t *testing.T) {
	names := []string{
		"SYNC_MAX_STREAMS",
		"SYNC_MAX_STREAMS_PER_PEER",
		"SYNC_EVENTS_PER_PEER",
	}

	for _, name := range names {
		for _, value := range []string{"0", "-1", "many"} {
			t.Run(name+"="+value, func(t *testing.T) {
				// Arrange
				clearEnv(t)
				t.Setenv(name, value)

				// Act
				_, err := Load()

				// Assert
				if err == nil {
					t.Fatalf("%s=%s should fail", name, value)
				}
			})
		}
	}
}


func TestLoadRefusesInvalidConnLimits(t *testing.T) {
	cases := map[string][2]string{
		"low above high":    {"100", "50"},
		"low below one":     {"0", "50"},
		"low not a number":  {"few", "50"},
		"high not a number": {"8", "many"},
	}

	for name, pair := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			clearEnv(t)
			t.Setenv("CONN_LOW", pair[0])
			t.Setenv("CONN_HIGH", pair[1])

			// Act
			_, err := Load()

			// Assert
			if err == nil {
				t.Fatalf("Load accepted CONN_LOW=%s CONN_HIGH=%s",
				         pair[0],
				         pair[1])
			}
		})
	}
}


func TestLoadBuildsTheBootstrapListFromBothVariables(t *testing.T) {
	cases := map[string]struct {
		addrs string
		extra string
		want  []string
	}{
		"neither":           {"", "", []string{"embedded"}},
		"replaces":          {"a", "", []string{"a"}},
		"adds":              {"", "b", []string{"embedded", "b"}},
		"replaces and adds": {"a", "b", []string{"a", "b"}},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			clearEnv(t)
			previous := defaultBootstrapAddrs
			defaultBootstrapAddrs = []string{"embedded"}
			t.Cleanup(func() { defaultBootstrapAddrs = previous })

			if c.addrs != "" {
				t.Setenv("BOOTSTRAP_ADDRS", c.addrs)
			}

			if c.extra != "" {
				t.Setenv("BOOTSTRAP_EXTRA_ADDRS", c.extra)
			}

			// Act
			got, err := Load()

			// Assert
			if err != nil || !reflect.DeepEqual(got.BootstrapAddrs, c.want) {
				t.Fatalf("BootstrapAddrs = %v, %v, want %v",
				         got.BootstrapAddrs,
				         err,
				         c.want)
			}
		})
	}
}
