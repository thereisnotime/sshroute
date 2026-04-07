// Package output — yaml.go renders data as YAML.
package output

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type yamlFormatter struct{}

// Format marshals data to YAML and writes it to w.
func (y *yamlFormatter) Format(w io.Writer, data any) error {
	b, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("yaml marshal: %w", err)
	}
	_, err = fmt.Fprint(w, string(b))
	return err
}
