// Package output provides multi-format output rendering for sshroute CLI commands.
package output

import "io"

// Format constants.
const (
	FormatTable = "table"
	FormatJSON  = "json"
	FormatYAML  = "yaml"
)

// Formatter renders structured data to a writer.
type Formatter interface {
	Format(w io.Writer, data any) error
}

// New returns a Formatter for the given format string.
// Defaults to table if format is unrecognised.
func New(format string) Formatter {
	switch format {
	case FormatJSON:
		return &jsonFormatter{}
	case FormatYAML:
		return &yamlFormatter{}
	default:
		return &tableFormatter{}
	}
}
