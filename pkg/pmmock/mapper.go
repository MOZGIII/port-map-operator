package pmmock

import (
	"context"

	"github.com/MOZGIII/port-map-operator/pkg/portmap"
)

type RequestWrap struct {
	Context context.Context
	Request *portmap.Request
}

type ResponseWrap struct {
	Response *portmap.Response
	Error    error
}

type Control struct {
	RequestCh  <-chan RequestWrap
	ResponseCh chan<- ResponseWrap
}

type MockMapper struct {
	req chan<- RequestWrap
	res <-chan ResponseWrap
}

var _ portmap.Mapper = (*MockMapper)(nil)

func New() (*MockMapper, *Control) {
	req := make(chan RequestWrap)
	res := make(chan ResponseWrap)
	mapper := &MockMapper{
		req: req,
		res: res,
	}
	control := &Control{
		RequestCh:  req,
		ResponseCh: res,
	}
	return mapper, control
}

func (m *MockMapper) Map(ctx context.Context, req *portmap.Request) (*portmap.Response, error) {
	m.req <- RequestWrap{Context: ctx, Request: req}
	res := <-m.res
	return res.Response, res.Error
}
