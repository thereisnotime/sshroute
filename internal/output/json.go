// Package output — json.go renders data as JSON.
package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type jsonFormatter struct{}

// Format marshals data to indented JSON and writes it to w.
func (j *jsonFormatter) Format(w io.Writer, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	_, err = fmt.Fprintf(w, "%s\n", b)
	return err
}
