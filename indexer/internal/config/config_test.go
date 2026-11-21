package config


import (
	"os"
	"reflect"
	"testing"
)


// --- GLOBALS ---

// names lists every variable Load reads.
var names = []string{
	"DATABASE_URL",
	"HTTP_ADDR",
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
		DatabaseURL: defaultDatabaseURL,
		HTTPAddr:    "127.0.0.1:8081",
	}

	// Act
	got, err := Load()

	// Assert
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Load = %+v, %v, want %+v", got, err, want)
	}
}


func TestLoadReadsEachVariable(t *testing.T) {
	cases := []struct {
		name  string
		value string
		read  func(Config) any
	}{
		{
			"DATABASE_URL", "postgresql://u:p@db:5432/x",
			func(c Config) any { return c.DatabaseURL },
		},
		{
			"HTTP_ADDR", "0.0.0.0:9000",
			func(c Config) any { return c.HTTPAddr },
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
			if err != nil || c.read(got) != c.value {
				t.Fatalf("%s = %v, %v, want %v",
				         c.name, c.read(got), err, c.value)
			}
		})
	}
}


func TestLoadRefusesADatabaseURLThatIsNotPostgres(t *testing.T) {
	// Arrange
	clearEnv(t)
	t.Setenv("DATABASE_URL", "mysql://u:p@db/x")

	// Act
	_, err := Load()

	// Assert
	if err == nil {
		t.Fatal("a mysql:// DATABASE_URL should fail")
	}
}
