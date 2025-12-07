package unit

import (
	"encoding"
	"fmt"
	"time"

	str2duration "github.com/xhit/go-str2duration/v2"
)

// Duration is a wrapper around time.Duration that supports parsing days.
type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler.
//
// It is needed for Kong:
// https://pkg.go.dev/github.com/alecthomas/kong#readme-custom-decoders-mappers
func (d *Duration) UnmarshalText(text []byte) error {
	dd, err := str2duration.ParseDuration(string(text))
	if err != nil {
		return err
	}

	*d = Duration(dd)
	return nil
}

// String implements fmt.Stringer.
func (d Duration) String() string {
	return str2duration.String(time.Duration(d))
}

// check interfaces
var (
	_ encoding.TextUnmarshaler = (*Duration)(nil)
	_ fmt.Stringer             = Duration(0)
)
