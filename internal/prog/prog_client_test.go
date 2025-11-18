package prog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/AlekSi/hardcache/internal/go/cacheprog"
	"github.com/AlekSi/lazyerrors"
)

// client is a simple GOCACHEPROG client for use in tests.
//
//nolint:vet // for readability
type client struct {
	inM sync.Mutex
	inD *json.Decoder

	outM sync.Mutex
	outC io.Closer
	outE *json.Encoder
}

// newClient creates a new client connected to the given reader and writer.
func newClient(r io.Reader, w io.WriteCloser) (*client, error) {
	c := &client{
		inD:  json.NewDecoder(r),
		outC: w,
		outE: json.NewEncoder(w),
	}

	resp, err := c.recv()
	if err != nil {
		_ = w.Close()
		return nil, lazyerrors.Error(err)
	}

	expected := &cacheprog.Response{
		KnownCommands: []cacheprog.Cmd{cacheprog.CmdGet, cacheprog.CmdPut, cacheprog.CmdClose},
	}
	if !reflect.DeepEqual(resp, expected) {
		_ = w.Close()
		return nil, fmt.Errorf("client.newClient: expected initial response %+v, got %+v", expected, resp)
	}

	return c, nil
}

// close correctly closes the client connection.
func (c *client) close() (err error) {
	defer func() {
		if e := c.outC.Close(); e != nil && err == nil {
			err = lazyerrors.Error(e)
		}

		if _, e := c.recv(); !errors.Is(e, io.EOF) && err == nil {
			err = lazyerrors.Error(e)
		}
	}()

	err = c.send(&cacheprog.Request{
		ID:      100500,
		Command: cacheprog.CmdClose,
	})
	if err != nil {
		err = lazyerrors.Error(err)
		return
	}

	var resp *cacheprog.Response
	if resp, err = c.recv(); err != nil {
		err = lazyerrors.Error(err)
		return
	}

	expected := &cacheprog.Response{
		ID: 100500,
	}
	if !reflect.DeepEqual(resp, expected) {
		err = fmt.Errorf("client.close: expected empty response, got %+v", resp)
		return
	}

	return
}

// send sends a request.
func (c *client) send(req *cacheprog.Request) error {
	c.outM.Lock()
	defer c.outM.Unlock()

	return c.outE.Encode(req)
}

// recv receives a response.
func (c *client) recv() (*cacheprog.Response, error) {
	c.inM.Lock()
	defer c.inM.Unlock()

	var resp cacheprog.Response
	if err := c.inD.Decode(&resp); err != nil {
		return nil, lazyerrors.Error(err)
	}

	return &resp, nil
}
