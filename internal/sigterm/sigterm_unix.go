//go:build unix

package sigterm

import (
	"context"
	"os/signal"

	"golang.org/x/sys/unix"
)

// Ctx returns a copy of the parent context that is marked done
// (its Done channel is closed) when termination signal arrives,
// when the returned stop function is called, or when the parent context's
// Done channel is closed, whichever happens first.
func Ctx(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, unix.SIGTERM, unix.SIGINT)
}
