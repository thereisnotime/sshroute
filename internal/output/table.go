// Package output — table.go renders data as a terminal table.
package output

import "io"

type tableFormatter struct{}

func (t *tableFormatter) Format(w io.Writer, data any) error { return nil } // implemented by A5
