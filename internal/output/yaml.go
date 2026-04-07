// Package output — yaml.go renders data as YAML.
package output

import "io"

type yamlFormatter struct{}

func (y *yamlFormatter) Format(w io.Writer, data any) error { return nil } // implemented by A5
