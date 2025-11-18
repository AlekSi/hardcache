package prog

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AlekSi/hardcache/internal/caches/local"
	"github.com/AlekSi/hardcache/internal/go/cacheprog"
)

// logger returns a [slog.Logger] for the given test.
func logger(t testing.TB) *slog.Logger {
	t.Helper()

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}
	return slog.New(slog.NewTextHandler(t.Output(), opts))
}

func TestProg(t *testing.T) {
	cache, err := local.New(t.TempDir(), nil, nil, logger(t))
	require.NoError(t, err)

	var p *Prog
	var c *client

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	{
		pr, cw := io.Pipe()
		cr, pw := io.Pipe()

		p = New(cache, l, pr, pw)

		go func() {
			assert.NoError(t, p.Run(ctx))
			close(done)
		}()

		c, err = newClient(cr, cw)
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		assert.NoError(t, c.close())

		// TODO check sending command after close

		<-done
		cancel()
	})

	t.Run("GetInvalidActionID", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, c.send(&cacheprog.Request{
			ID:       100,
			Command:  cacheprog.CmdGet,
			ActionID: []byte("missing"),
		}))

		resp, err := c.recv()
		require.NoError(t, err)

		expected := &cacheprog.Response{
			ID:  100,
			Err: "hardcache: get: invalid action ID size 7, expected 32",
		}
		assert.Equal(t, expected, resp)
	})
}
