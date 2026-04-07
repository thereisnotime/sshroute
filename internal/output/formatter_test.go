package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type testRow struct {
	Name  string `json:"name"  yaml:"name"  table:"NAME"`
	Value int    `json:"value" yaml:"value" table:"VALUE"`
}

func TestNew(t *testing.T) {
	if _, ok := New(FormatJSON).(*jsonFormatter); !ok {
		t.Error("New(json) should return *jsonFormatter")
	}
	if _, ok := New(FormatYAML).(*yamlFormatter); !ok {
		t.Error("New(yaml) should return *yamlFormatter")
	}
	if _, ok := New(FormatTable).(*tableFormatter); !ok {
		t.Error("New(table) should return *tableFormatter")
	}
	if _, ok := New("unknown").(*tableFormatter); !ok {
		t.Error("New(unknown) should default to *tableFormatter")
	}
}

func TestJSONFormatter(t *testing.T) {
	rows := []testRow{{"foo", 1}, {"bar", 2}}
	var buf bytes.Buffer
	if err := New(FormatJSON).Format(&buf, rows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out []testRow
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(out) != 2 || out[0].Name != "foo" || out[1].Value != 2 {
		t.Errorf("unexpected decoded output: %+v", out)
	}
}

func TestYAMLFormatter(t *testing.T) {
	rows := []testRow{{"foo", 1}}
	var buf bytes.Buffer
	if err := New(FormatYAML).Format(&buf, rows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "foo") || !strings.Contains(out, "name") {
		t.Errorf("YAML output missing expected fields: %s", out)
	}
}

func TestTableFormatter(t *testing.T) {
	rows := []testRow{{"hello", 42}}
	var buf bytes.Buffer
	if err := New(FormatTable).Format(&buf, rows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Errorf("table missing NAME header: %s", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("table missing row value: %s", out)
	}
}

func TestTableFormatter_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	if err := New(FormatTable).Format(&buf, []testRow{}); err != nil {
		t.Fatalf("unexpected error on empty slice: %v", err)
	}
}

func TestTableFormatter_NonSlice(t *testing.T) {
	var buf bytes.Buffer
	// Should not panic, just print the value
	if err := New(FormatTable).Format(&buf, "plain string"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONFormatter_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	if err := New(FormatJSON).Format(&buf, []testRow{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "[]" {
		t.Errorf("expected [], got %q", buf.String())
	}
}

func TestYAMLFormatter_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	if err := New(FormatYAML).Format(&buf, []testRow{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTableFormatter_MultipleRows(t *testing.T) {
	rows := []testRow{{"a", 1}, {"b", 2}, {"c", 3}}
	var buf bytes.Buffer
	if err := New(FormatTable).Format(&buf, rows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, name := range []string{"a", "b", "c"} {
		if !strings.Contains(out, name) {
			t.Errorf("table missing row %q", name)
		}
	}
}

// TestTableFormatter_NilPointer covers the nil-pointer dereference guard (lines 21-27):
// when data is a non-nil pointer that points to nil, Format should print "(nil)".
func TestTableFormatter_NilPointer(t *testing.T) {
	var buf bytes.Buffer
	var p *[]testRow // typed nil pointer
	if err := New(FormatTable).Format(&buf, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "(nil)") {
		t.Errorf("expected \"(nil)\" output for nil pointer, got %q", buf.String())
	}
}

// testRowWithSkip has a field tagged table:"-" to exercise the skip branch (lines 59-61).
type testRowWithSkip struct {
	Name   string `table:"NAME"`
	Secret string `table:"-"`
	Value  int    `table:"VALUE"`
}

// TestTableFormatter_SkipTag covers the table:"-" field-skip branch (lines 59-61).
func TestTableFormatter_SkipTag(t *testing.T) {
	rows := []testRowWithSkip{{"alice", "hidden", 7}}
	var buf bytes.Buffer
	if err := New(FormatTable).Format(&buf, rows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Errorf("table should not contain the skipped field value, got %q", out)
	}
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "alice") {
		t.Errorf("table missing expected NAME/alice content, got %q", out)
	}
}

// TestTableFormatter_SliceOfPointers covers the slice-of-pointer-to-struct branch (lines 45-47)
// and the nil-pointer element branch (lines 73-78).
func TestTableFormatter_SliceOfPointers(t *testing.T) {
	a := &testRow{"x", 10}
	b := &testRow{"y", 20}
	rows := []*testRow{a, nil, b} // nil in the middle exercises the per-element nil guard
	var buf bytes.Buffer
	if err := New(FormatTable).Format(&buf, rows); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "x") || !strings.Contains(out, "y") {
		t.Errorf("table missing expected row values, got %q", out)
	}
}

// TestTableFormatter_SliceOfNonStruct covers the slice-of-non-struct branch (lines 48-51).
func TestTableFormatter_SliceOfNonStruct(t *testing.T) {
	var buf bytes.Buffer
	if err := New(FormatTable).Format(&buf, []string{"foo", "bar"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The fallback path just prints the whole slice via fmt.Fprintf; output should be non-empty.
	if buf.Len() == 0 {
		t.Error("expected non-empty output for slice of non-struct")
	}
}
