package pcpcliwrap

import (
	"context"
	"errors"

	"github.com/MOZGIII/port-map-operator/pkg/portmap"
)

var (
	ErrResponseChannelClosed = errors.New("response channel closed")
)

type PCP struct {
	control chan opReq
	cmd     *Command
}

var _ portmap.Mapper = (*PCP)(nil)

func New(cmd *Command) *PCP {
	return &PCP{
		control: make(chan opReq),
		cmd:     cmd,
	}
}

type opRes struct {
	Response *portmap.Response
	Error    error
}

type opReq struct {
	Request    *portmap.Request
	Context    context.Context
	ResponseCh chan<- *opRes
}

func (p *PCP) Run(stopch <-chan struct{}) error {
	for {
		select {
		case <-stopch:
			close(p.control)
			return nil
		case req := <-p.control:
			res, err := p.cmd.Exec(req.Context, req.Request)
			if req.Context.Err() != nil {
				// Context has expired, this means we are no longer interested
				// in the response, and the response channel should've been
				// closed.
				continue
			}
			req.ResponseCh <- &opRes{Response: res, Error: err}
		}
	}
}

func (p *PCP) Map(ctx context.Context, req *portmap.Request) (*portmap.Response, error) {
	resCh := make(chan *opRes)
	defer close(resCh)

	p.control <- opReq{
		Context:    ctx,
		Request:    req,
		ResponseCh: resCh,
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resCh:
		if res == nil {
			// Should not happen, but in case channel was closed without
			// sending a value - return an error.
			return nil, ErrResponseChannelClosed
		}
		return res.Response, res.Error
	}
}
