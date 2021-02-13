package portmap

import (
	"context"
	"net"
)

type Mapper interface {
	Map(ctx context.Context, req *Request) (*Response, error)
}

type Request struct {
	Protocol    Protocol
	NodePort    Port
	GatewayPort Port

	// Pass `LifetimeDelete` to request mapping deletion.
	Lifetime Lifetime
}

type Response struct {
	Protocol    Protocol
	NodePort    Port
	GatewayPort Port
	GatewayIP   net.IP
	Lifetime    Lifetime
}
