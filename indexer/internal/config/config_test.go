package config


import (
	"testing"
)


// --- CODE ---

func TestLoadAppliesTheDefaultToAnEmptyEnvironment(t *testing.T) {
	// Arrange
	t.Setenv("HTTP_ADDR", "")

	// Act
	got, err := Load()

	// Assert
	if err != nil || got.HTTPAddr != "127.0.0.1:8081" {
		t.Fatalf("Load = %+v, %v, want the default address", got, err)
	}
}


func TestLoadReadsHTTPAddr(t *testing.T) {
	// Arrange
	t.Setenv("HTTP_ADDR", "0.0.0.0:9000")

	// Act
	got, err := Load()

	// Assert
	if err != nil || got.HTTPAddr != "0.0.0.0:9000" {
		t.Fatalf("Load = %+v, %v, want 0.0.0.0:9000", got, err)
	}
}
