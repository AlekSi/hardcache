// Package prog provides GOCACHEPROG protocol wrapper for a given cache.
package prog

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/AlekSi/hardcache/internal/go/cache"
	"github.com/AlekSi/hardcache/internal/go/cacheprog"
)

// Prog provides GOCACHEPROG implementation for the given [cache.Cache]
// (that should safe to use from multiple goroutines and multiple processes).
//
//nolint:vet // for readability
type Prog struct {
	c      cache.Cache
	l      *slog.Logger
	closed bool

	inD *json.Decoder

	outM sync.Mutex
	outC io.Closer
	outW *bufio.Writer
	outE *json.Encoder
}

// New creates a new [Prog].
// It takes over the cache and reader/writer.
func New(c cache.Cache, l *slog.Logger, r io.Reader, w io.WriteCloser) *Prog {
	dec := json.NewDecoder(bufio.NewReader(r))
	dec.DisallowUnknownFields()

	outW := bufio.NewWriter(w)
	outE := json.NewEncoder(outW)

	return &Prog{
		c:    c,
		l:    l,
		inD:  dec,
		outC: w,
		outW: outW,
		outE: outE,
	}
}

// Run runs the Prog until ctx is canceled, close message is received and handled,
// or an error occurs.
// On exit, it closes the cache.
func (p *Prog) Run(ctx context.Context) (err error) {
	defer func() {
		if e := p.c.Close(); err == nil {
			err = e
		}
		if e := p.outC.Close(); err == nil {
			err = e
		}
	}()

	err = p.send(&cacheprog.Response{
		KnownCommands: []cacheprog.Cmd{cacheprog.CmdGet, cacheprog.CmdPut, cacheprog.CmdClose},
	})
	if err != nil {
		return
	}

	for {
		// Go tool does send multiple requests in a pipeline; we should handle that.

		var req cacheprog.Request
		if err = p.inD.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) && p.closed {
				p.l.Debug("Exiting due to EOF after close")
				err = nil
				return
			}

			p.l.Warn("Exiting due to error", slog.Bool("closed", p.closed))
			return
		}

		args := []any{
			slog.Int64("id", req.ID),
			slog.String("command", string(req.Command)),
		}

		if req.ActionID != nil {
			args = append(args, slog.String("actionID", fmt.Sprintf("%x", req.ActionID)))
		}

		if req.OutputID != nil {
			args = append(args, slog.String("outputID", fmt.Sprintf("%x", req.OutputID)))
		}

		args = append(args, slog.Int64("bodySize", req.BodySize))

		p.l.DebugContext(ctx, "Request", args...)

		var resp *cacheprog.Response

		switch req.Command {
		case cacheprog.CmdGet:
			resp, err = p.handleGet(&req)

		case cacheprog.CmdPut:
			resp, err = p.handlePut(&req)

		case cacheprog.CmdClose:
			resp, err = p.handleClose(&req)

		default:
			resp = &cacheprog.Response{
				ID:  req.ID,
				Err: fmt.Sprintf("hardcache: unknown command %q", req.Command),
			}
		}

		if err != nil {
			return
		}

		args = []any{slog.Int64("id", resp.ID)}

		if resp.Err != "" {
			args = append(args, slog.String("error", resp.Err))
		}

		args = append(args, slog.Bool("miss", resp.Miss))

		if resp.OutputID != nil {
			args = append(args, slog.String("outputID", fmt.Sprintf("%x", resp.OutputID)))
		}

		args = append(args, slog.Int64("size", resp.Size))

		if resp.Time != nil {
			args = append(args, slog.String("time", resp.Time.Format(time.RFC3339Nano)))
		}

		args = append(args, slog.String("diskPath", resp.DiskPath))

		p.l.DebugContext(ctx, "Response", args...)

		if err = p.send(resp); err != nil {
			return
		}
	}
}

// handleGet handles the `get` command.
// Returned error is something fatal.
func (p *Prog) handleGet(req *cacheprog.Request) (*cacheprog.Response, error) {
	resp := cacheprog.Response{
		ID: req.ID,
	}

	if len(req.ActionID) != cache.HashSize {
		resp.Err = fmt.Sprintf("hardcache: get: invalid action ID size %d, expected %d", len(req.ActionID), cache.HashSize)
		return &resp, nil
	}

	if req.OutputID != nil {
		resp.Err = "hardcache: get: unexpected output ID"
		return &resp, nil
	}

	entry, err := p.c.Get(cache.ActionID(req.ActionID))
	if err == nil {
		resp.OutputID = entry.OutputID[:]
		resp.Size = entry.Size
		resp.Time = &entry.Time
		resp.DiskPath = p.c.OutputFile(entry.OutputID)

		return &resp, nil
	}

	var notFound *cache.EntryNotFoundError
	if errors.As(err, &notFound) {
		resp.Miss = true
		return &resp, nil
	}

	// it is not entirely clear what errors are possible there and how to handle them
	// (should we send it in resp.Err? treat as fatal?)
	p.l.Warn(fmt.Sprintf("get: %[1]s (%[1]T)", err))
	resp.Miss = true
	return &resp, nil
}

// handlePut handles the `put` command.
// Returned error is something fatal.
func (p *Prog) handlePut(req *cacheprog.Request) (*cacheprog.Response, error) {
	resp := cacheprog.Response{
		ID: req.ID,
	}

	if len(req.ActionID) != cache.HashSize {
		resp.Err = fmt.Sprintf("hardcache: put: invalid action ID size %d, expected %d", len(req.ActionID), cache.HashSize)
		return &resp, nil
	}

	if len(req.OutputID) != cache.HashSize {
		resp.Err = fmt.Sprintf("hardcache: put: invalid output ID size %d, expected %d", len(req.OutputID), cache.HashSize)
		return &resp, nil
	}

	var b []byte
	if req.BodySize > 0 {
		// currently, there is no way to make JSON decoder use preallocated slice capacity
		if err := p.inD.Decode(&b); err != nil {
			return nil, err
		}

		if l := len(b); int(req.BodySize) != l {
			return nil, fmt.Errorf("put: expected body size %d, got %d", req.BodySize, l)
		}
	}

	outputID, size, err := p.c.Put(cache.ActionID(req.ActionID), bytes.NewReader(b))
	if err != nil {
		resp.Err = err.Error()
		return &resp, nil
	}

	if id := cache.OutputID(req.OutputID); id != outputID {
		resp.Err = fmt.Sprintf("hardcache: put: expected output ID %x, got %x", id, outputID)
		return &resp, nil
	}

	if req.BodySize != size {
		resp.Err = fmt.Sprintf("hardcache: put: expected size %d, got %d", req.BodySize, size)
		return &resp, nil
	}

	resp.DiskPath = p.c.OutputFile(outputID)
	return &resp, nil
}

// handleClose handles the `close` command.
// Returned error is something fatal.
func (p *Prog) handleClose(req *cacheprog.Request) (*cacheprog.Response, error) {
	p.closed = true
	return &cacheprog.Response{ID: req.ID}, nil
}

// send sends the given response.
// Returned error is something fatal.
func (p *Prog) send(resp *cacheprog.Response) error {
	p.outM.Lock()
	defer p.outM.Unlock()

	if err := p.outE.Encode(resp); err != nil {
		return err
	}

	return p.outW.Flush()
}
