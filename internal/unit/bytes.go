// Package unit provides utilities for parsing and handling various units of measurement.
package unit

import (
	"encoding"
	"fmt"

	"github.com/alecthomas/units"
)

// Bytes represents the size in bytes.
type Bytes int64

// UnmarshalText implements encoding.TextUnmarshaler.
//
// It is needed for Kong:
// https://pkg.go.dev/github.com/alecthomas/kong#readme-custom-decoders-mappers
func (b *Bytes) UnmarshalText(text []byte) error {
	bb, err := units.ParseStrictBytes(string(text))
	if err != nil {
		return err
	}

	*b = Bytes(bb)
	return nil
}

// String implements fmt.Stringer.
func (b Bytes) String() string {
	return units.MetricBytes(b).Floor().String()
}

// check interfaces
var (
	_ encoding.TextUnmarshaler = (*Bytes)(nil)
	_ fmt.Stringer             = Bytes(0)
)
