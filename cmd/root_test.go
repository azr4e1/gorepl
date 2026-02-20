package cmd

import (
	"testing"
)

func TestConnectionFlagString(t *testing.T) {
	f := connectionFlag("uds")
	if f.String() != "uds" {
		t.Fatalf("expected 'uds', got %q", f.String())
	}
}

func TestConnectionFlagType(t *testing.T) {
	f := connectionFlag("tcp")
	if f.Type() != "connection" {
		t.Fatalf("expected 'connection', got %q", f.Type())
	}
}

func TestConnectionFlagSetValidValues(t *testing.T) {
	valid := []string{"uds", "tcp", "namedpipe"}
	for _, v := range valid {
		var f connectionFlag
		if err := f.Set(v); err != nil {
			t.Fatalf("expected no error for %q, got %v", v, err)
		}
		if f.String() != v {
			t.Fatalf("expected value %q after Set, got %q", v, f.String())
		}
	}
}

func TestConnectionFlagSetInvalidValue(t *testing.T) {
	var f connectionFlag
	err := f.Set("invalid")
	if err == nil {
		t.Fatal("expected error for invalid connection type")
	}
}

func TestConnectionFlagSetEmptyString(t *testing.T) {
	var f connectionFlag
	err := f.Set("")
	if err == nil {
		t.Fatal("expected error for empty connection type")
	}
}

func TestConnectionFlagSetCaseSensitive(t *testing.T) {
	var f connectionFlag
	// Values are case-sensitive: "UDS" should not be accepted
	err := f.Set("UDS")
	if err == nil {
		t.Fatal("expected error: connection flag is case-sensitive")
	}
}
