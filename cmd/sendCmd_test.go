package cmd

import (
	"strings"
	"testing"
)

// --- preprocess tests ---

func TestPreprocessSingleLine(t *testing.T) {
	result := preprocess([]byte("hello\n"))
	if string(result) != "hello\n" {
		t.Fatalf("expected 'hello\\n', got %q", result)
	}
}

func TestPreprocessMultipleLines(t *testing.T) {
	input := "line1\nline2\nline3\n"
	result := preprocess([]byte(input))
	if string(result) != input {
		t.Fatalf("expected %q, got %q", input, result)
	}
}

func TestPreprocessRemovesEmptyLines(t *testing.T) {
	input := "line1\n\nline2\n\n"
	result := preprocess([]byte(input))
	expected := "line1\nline2\n"
	if string(result) != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestPreprocessRemovesWhitespaceOnlyLines(t *testing.T) {
	input := "line1\n   \nline2\n\t\n"
	result := preprocess([]byte(input))
	expected := "line1\nline2\n"
	if string(result) != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestPreprocessTrimsTrailingSpaces(t *testing.T) {
	input := "line1   \nline2\t\n"
	result := preprocess([]byte(input))
	expected := "line1\nline2\n"
	if string(result) != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestPreprocessTrimsTrailingTabs(t *testing.T) {
	input := "hello\t\t\nworld\n"
	result := preprocess([]byte(input))
	expected := "hello\nworld\n"
	if string(result) != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestPreprocessAlwaysEndsWithNewline(t *testing.T) {
	cases := []string{
		"hello",
		"hello\n",
		"hello\n\n",
		"a\nb",
	}
	for _, tc := range cases {
		result := preprocess([]byte(tc))
		if len(result) == 0 || result[len(result)-1] != '\n' {
			t.Fatalf("input %q: expected output to end with newline, got %q", tc, result)
		}
	}
}

func TestPreprocessAllEmptyLines(t *testing.T) {
	input := "\n\n\n"
	result := preprocess([]byte(input))
	// All lines are empty, so output should just be the final newline separator
	if string(result) != "\n" {
		t.Fatalf("expected single newline for all-empty input, got %q", result)
	}
}

func TestPreprocessNoNewlineInInput(t *testing.T) {
	input := "notrailing"
	result := preprocess([]byte(input))
	if string(result) != "notrailing\n" {
		t.Fatalf("expected 'notrailing\\n', got %q", result)
	}
}

func TestPreprocessPreservesInternalContent(t *testing.T) {
	input := "  leading spaces\n"
	result := preprocess([]byte(input))
	// Leading spaces are preserved; only trailing whitespace is trimmed
	if !strings.Contains(string(result), "  leading spaces") {
		t.Fatalf("expected leading spaces preserved, got %q", result)
	}
}

func TestPreprocessMixedContent(t *testing.T) {
	input := "first\n\nsecond   \n   \nthird\t\n"
	result := preprocess([]byte(input))
	expected := "first\nsecond\nthird\n"
	if string(result) != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}
