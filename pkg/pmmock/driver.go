package pmmock

import (
	"time"

	"github.com/MOZGIII/port-map-operator/pkg/portmap"
	. "github.com/onsi/gomega"
)

func (ctl *Control) Expect(req *portmap.Request, timeout time.Duration) {
	select {
	case ctlreq := <-ctl.RequestCh:
		ExpectWithOffset(1, ctlreq.Request).To(Equal(req))
		ExpectWithOffset(1, ctlreq.Context).NotTo(BeNil())
	case <-time.After(timeout):
		panic("timeout while waiting for a request")
	}
}

func (ctl *Control) Inject(res *portmap.Response, timeout time.Duration) {
	ctlres := ResponseWrap{
		Response: res,
		Error:    nil,
	}
	select {
	case ctl.ResponseCh <- ctlres:
		// fine, noop
	case <-time.After(timeout):
		panic("timeout while injecting response")
	}
}
