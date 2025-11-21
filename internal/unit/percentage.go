package unit

import (
	"encoding"
	"fmt"
	"strconv"
	"strings"
)

// Percentage represents the number of percents.
type Percentage int

// UnmarshalText implements encoding.TextUnmarshaler.
//
// It is needed for Kong:
// https://pkg.go.dev/github.com/alecthomas/kong#readme-custom-decoders-mappers
func (p *Percentage) UnmarshalText(text []byte) error {
	s, cut := strings.CutSuffix(string(text), "%")
	if !cut {
		return fmt.Errorf("percentage must end with %%: %q", text)
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	*p = Percentage(v)
	return nil
}

// String implements fmt.Stringer.
func (p Percentage) String() string {
	return strconv.Itoa(int(p)) + "%"
}

// check interfaces
var (
	_ encoding.TextUnmarshaler = (*Percentage)(nil)
	_ fmt.Stringer             = Percentage(0)
)
