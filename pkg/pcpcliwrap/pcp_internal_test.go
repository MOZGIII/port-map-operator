package pcpcliwrap

import (
	"context"
	"errors"
	"net"
	"os/exec"

	"github.com/MOZGIII/port-map-operator/pkg/portmap"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PCP", func() {
	var (
		cmd *Command
		req *portmap.Request
		res *portmap.Response
		err error
	)

	JustBeforeEach(func() {
		pcp := New(cmd)
		stopch := make(chan struct{})
		donech := make(chan struct{})

		go func() {
			if runerr := pcp.Run(stopch); runerr != nil {
				Expect(runerr).To(BeNil())
			}
			close(donech)
		}()

		res, err = pcp.Map(context.Background(), req)

		close(stopch)
		<-donech
	})

	Context("with a pcp cli simulator", func() {
		BeforeEach(func() {
			cmd = &Command{
				CommandName: "testdata/pcpsimulator.sh",
				ServerAddr:  "127.0.0.1:5351",
			}
			req = &portmap.Request{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(80),
				Lifetime:    portmap.Lifetime(120),
			}
		})

		It("should produce a correct response", func() {
			Expect(err).To(BeNil())
			Expect(res.Protocol).To(Equal(portmap.ProtocolTCP))
			Expect(res.NodePort).To(Equal(portmap.Port(32100)))
			Expect(res.GatewayPort).To(Equal(portmap.Port(1024)))
			Expect(res.GatewayIP).To(Equal(net.IPv4(1, 2, 3, 4)))
			Expect(res.Lifetime).To(BeNumerically("~", 120, 1))
		})
	})

	Context("with a fail command", func() {
		BeforeEach(func() {
			cmd = &Command{
				CommandName: "testdata/fail.sh",
			}
		})

		It("should produce an expected error", func() {
			Expect(err).To(MatchError("PCP CLI failed: exit status 1: Important message\n"))
			Expect(res).To(BeNil())

			exitErr := &exec.ExitError{}
			isExitErr := errors.As(err, &exitErr)
			Expect(isExitErr).To(BeTrue())

			Expect(exitErr.ProcessState.ExitCode()).To(BeIdenticalTo(1))
			Expect(exitErr.Stderr).To(Equal([]byte("Important message\n")))
		})
	})
})
