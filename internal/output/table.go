// Package output — table.go renders data as a terminal table.
package output

import (
	"fmt"
	"io"
	"reflect"

	"github.com/jedib0t/go-pretty/v6/table"
)

type tableFormatter struct{}

// Format renders data as a light-style terminal table.
// data must be a slice of structs. Headers are read from `table:"HEADER"` struct tags.
// Falls back to plain JSON-like output for unsupported types.
func (t *tableFormatter) Format(w io.Writer, data any) error {
	rv := reflect.ValueOf(data)

	// Dereference pointer if needed.
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			_, err := fmt.Fprintln(w, "(nil)")
			return err
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Slice {
		_, err := fmt.Fprintf(w, "%v\n", data)
		return err
	}

	tbl := table.NewWriter()
	tbl.SetOutputMirror(w)
	tbl.SetStyle(table.StyleLight)

	if rv.Len() == 0 {
		tbl.Render()
		return nil
	}

	// Determine element type — support slices of structs or pointers to structs.
	elemType := rv.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		_, err := fmt.Fprintf(w, "%v\n", data)
		return err
	}

	// Build header row from struct field tags.
	headerRow := make(table.Row, 0, elemType.NumField())
	fieldIndices := make([]int, 0, elemType.NumField())
	for i := 0; i < elemType.NumField(); i++ {
		f := elemType.Field(i)
		tag := f.Tag.Get("table")
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = f.Name
		}
		headerRow = append(headerRow, tag)
		fieldIndices = append(fieldIndices, i)
	}
	tbl.AppendHeader(headerRow)

	// Build one row per element.
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		row := make(table.Row, 0, len(fieldIndices))
		for _, idx := range fieldIndices {
			row = append(row, elem.Field(idx).Interface())
		}
		tbl.AppendRow(row)
	}

	tbl.Render()
	return nil
}
