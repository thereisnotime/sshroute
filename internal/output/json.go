// Package output — json.go renders data as JSON.
package output

import "io"

type jsonFormatter struct{}

func (j *jsonFormatter) Format(w io.Writer, data any) error { return nil } // implemented by A5
