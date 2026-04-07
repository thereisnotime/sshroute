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
