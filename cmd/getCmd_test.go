package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azr4e1/gorepl/internals"
)

// --- formatConnections tests ---

func TestFormatConnectionsEmpty(t *testing.T) {
	result := formatConnections([]Connection{})
	if result != "There are no active connections" {
		t.Fatalf("expected 'There are no active connections', got %q", result)
	}
}

func TestFormatConnectionsSingle(t *testing.T) {
	conns := []Connection{{Type: UDSType, Address: "/tmp/test.sock"}}
	result := formatConnections(conns)
	if !strings.Contains(result, "There is 1 active connection") {
		t.Fatalf("expected singular form, got %q", result)
	}
	if !strings.Contains(result, "/tmp/test.sock") {
		t.Fatalf("expected address in output, got %q", result)
	}
	if !strings.Contains(result, UDSType) {
		t.Fatalf("expected type in output, got %q", result)
	}
}

func TestFormatConnectionsMultiple(t *testing.T) {
	conns := []Connection{
		{Type: UDSType, Address: "/tmp/a.sock"},
		{Type: TCPType, Address: "localhost:4501"},
	}
	result := formatConnections(conns)
	if !strings.Contains(result, "There are 2 active connections") {
		t.Fatalf("expected plural form, got %q", result)
	}
	if !strings.Contains(result, "/tmp/a.sock") {
		t.Fatalf("expected first address in output, got %q", result)
	}
	if !strings.Contains(result, "localhost:4501") {
		t.Fatalf("expected second address in output, got %q", result)
	}
}

func TestFormatConnectionsThree(t *testing.T) {
	conns := []Connection{
		{Type: UDSType, Address: "/tmp/a.sock"},
		{Type: TCPType, Address: "localhost:4501"},
		{Type: NamedPipeType, Address: "/tmp/pipe"},
	}
	result := formatConnections(conns)
	if !strings.Contains(result, "There are 3 active connections") {
		t.Fatalf("expected '3 active connections', got %q", result)
	}
}

func TestFormatConnectionsContainsConnectionBlock(t *testing.T) {
	conns := []Connection{{Type: TCPType, Address: "localhost:9000"}}
	result := formatConnections(conns)
	if !strings.Contains(result, "Connection:") {
		t.Fatalf("expected 'Connection:' block, got %q", result)
	}
	if !strings.Contains(result, "Type:") {
		t.Fatalf("expected 'Type:' field, got %q", result)
	}
	if !strings.Contains(result, "Address:") {
		t.Fatalf("expected 'Address:' field, got %q", result)
	}
}

// --- marshallConnectionsJSON tests ---

func TestMarshallConnectionsJSONEmpty(t *testing.T) {
	result, err := marshallConnectionsJSON([]Connection{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty slice serializes as "[]"
	if strings.TrimSpace(result) != "[]" {
		t.Fatalf("expected '[]', got %q", result)
	}
}

func TestMarshallConnectionsJSONSingle(t *testing.T) {
	conns := []Connection{{Type: TCPType, Address: "localhost:4501"}}
	result, err := marshallConnectionsJSON(conns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var decoded []Connection
	if err := json.Unmarshal([]byte(result), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(decoded))
	}
	if decoded[0].Type != TCPType {
		t.Fatalf("expected type %q, got %q", TCPType, decoded[0].Type)
	}
	if decoded[0].Address != "localhost:4501" {
		t.Fatalf("expected address 'localhost:4501', got %q", decoded[0].Address)
	}
}

func TestMarshallConnectionsJSONMultiple(t *testing.T) {
	conns := []Connection{
		{Type: UDSType, Address: "/tmp/a.sock"},
		{Type: NamedPipeType, Address: "/tmp/a.pipe"},
		{Type: TCPType, Address: "localhost:5000"},
	}
	result, err := marshallConnectionsJSON(conns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var decoded []Connection
	if err := json.Unmarshal([]byte(result), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(decoded) != 3 {
		t.Fatalf("expected 3 connections, got %d", len(decoded))
	}
}

func TestMarshallConnectionsJSONIsIndented(t *testing.T) {
	conns := []Connection{{Type: TCPType, Address: "localhost:1234"}}
	result, err := marshallConnectionsJSON(conns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// MarshalIndent produces newlines
	if !strings.Contains(result, "\n") {
		t.Fatalf("expected indented JSON, got %q", result)
	}
}

// --- getAllConnections tests ---

func TestGetAllConnectionsUnknownType(t *testing.T) {
	_, err := getAllConnections("unknown")
	if err == nil {
		t.Fatal("expected error for unknown connection type")
	}
}

func TestGetAllConnectionsUDSNoFiles(t *testing.T) {
	// Should return empty slice, not an error, when no matching files exist.
	// (There may or may not be real sockets in /tmp from other processes — just
	//  ensure we get a non-error result.)
	conns, err := getAllConnections("uds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = conns // may be non-empty on a busy system; just verify no error
}

func TestGetAllConnectionsUDSFindsFile(t *testing.T) {
	// Create a fake UDS socket file with a name matching the expected pattern.
	// Pattern: gorepl_[a-zA-Z0-9]{78}_unix_socket
	fakeHash := strings.Repeat("a", 78)
	fakeName := fmt.Sprintf("gorepl_%s_%s", fakeHash, internals.UDSName)
	fakePath := filepath.Join(os.TempDir(), fakeName)

	f, err := os.Create(fakePath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(fakePath)

	conns, err := getAllConnections("uds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, c := range conns {
		if c.Address == fakePath {
			found = true
			if c.Type != UDSType {
				t.Fatalf("expected type %q, got %q", UDSType, c.Type)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected to find fake UDS connection at %q", fakePath)
	}
}

func TestGetAllConnectionsNamedPipeFindsFile(t *testing.T) {
	fakeHash := strings.Repeat("b", 78)
	fakeName := fmt.Sprintf("gorepl_%s_%s", fakeHash, internals.NPipeName)
	fakePath := filepath.Join(os.TempDir(), fakeName)

	f, err := os.Create(fakePath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(fakePath)

	conns, err := getAllConnections("namedpipe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, c := range conns {
		if c.Address == fakePath {
			found = true
			if c.Type != NamedPipeType {
				t.Fatalf("expected type %q, got %q", NamedPipeType, c.Type)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected to find fake named pipe connection at %q", fakePath)
	}
}

func TestGetAllConnectionsTCPFindsFile(t *testing.T) {
	fakeHash := strings.Repeat("c", 78)
	fakeName := fmt.Sprintf("gorepl_%s_%s", fakeHash, internals.TCPName)
	fakePath := filepath.Join(os.TempDir(), fakeName)

	expectedAddr := "localhost:9988"
	if err := os.WriteFile(fakePath, []byte(expectedAddr), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fakePath)

	conns, err := getAllConnections("tcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, c := range conns {
		if c.Address == expectedAddr {
			found = true
			if c.Type != TCPType {
				t.Fatalf("expected type %q, got %q", TCPType, c.Type)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected to find fake TCP connection with address %q", expectedAddr)
	}
}

// --- getCurConnections tests ---

func TestGetCurConnectionsUnknownType(t *testing.T) {
	_, err := getCurConnections("unknown")
	if err == nil {
		t.Fatal("expected error for unknown connection type")
	}
}

func TestGetCurConnectionsUDSNotExist(t *testing.T) {
	// Ensure the UDS file doesn't exist
	udsPath, err := internals.GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(udsPath)

	conns, err := getCurConnections("uds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Fatalf("expected empty connections, got %v", conns)
	}
}

func TestGetCurConnectionsUDSExists(t *testing.T) {
	udsPath, err := internals.GenerateUDSPath()
	if err != nil {
		t.Fatal(err)
	}
	// Create the socket file as a placeholder
	f, err := os.Create(udsPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(udsPath)

	conns, err := getCurConnections("uds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].Type != UDSType {
		t.Fatalf("expected type %q, got %q", UDSType, conns[0].Type)
	}
	if conns[0].Address != udsPath {
		t.Fatalf("expected address %q, got %q", udsPath, conns[0].Address)
	}
}

func TestGetCurConnectionsNamedPipeNotExist(t *testing.T) {
	npipePath, err := internals.GenerateNPipePath()
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(npipePath)

	conns, err := getCurConnections("namedpipe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Fatalf("expected empty connections, got %v", conns)
	}
}

func TestGetCurConnectionsNamedPipeExists(t *testing.T) {
	npipePath, err := internals.GenerateNPipePath()
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(npipePath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(npipePath)

	conns, err := getCurConnections("namedpipe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].Type != NamedPipeType {
		t.Fatalf("expected type %q, got %q", NamedPipeType, conns[0].Type)
	}
}

func TestGetCurConnectionsTCPNotExist(t *testing.T) {
	tcpPath, err := internals.GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(tcpPath)

	// When TCP trace file doesn't exist, getCurConnections returns empty slice
	// (it ignores the error from RetrieveTCPAddress).
	conns, err := getCurConnections("tcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Fatalf("expected empty connections when no TCP trace file, got %v", conns)
	}
}

func TestGetCurConnectionsTCPExists(t *testing.T) {
	tcpPath, err := internals.GenerateTCPPath()
	if err != nil {
		t.Fatal(err)
	}
	expectedAddr := "localhost:7777"
	if err := os.WriteFile(tcpPath, []byte(expectedAddr), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tcpPath)

	conns, err := getCurConnections("tcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].Type != TCPType {
		t.Fatalf("expected type %q, got %q", TCPType, conns[0].Type)
	}
	if conns[0].Address != expectedAddr {
		t.Fatalf("expected address %q, got %q", expectedAddr, conns[0].Address)
	}
}
