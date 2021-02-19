package pcpcliwrap

import (
	"context"
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
			Expect(res.Lifetime).To(BeNumerically("<=", 1)) // TODO: teach the simulator to adjust time properly
		})
	})

	Context("with a fail command", func() {
		BeforeEach(func() {
			cmd = &Command{
				CommandName: "testdata/fail.sh",
			}
		})

		It("should produce an expected error", func() {
			Expect(err).To(BeAssignableToTypeOf(&exec.ExitError{}))
			Expect(res).To(BeNil())

			exitErr, _ := err.(*exec.ExitError)
			Expect(exitErr.ProcessState.ExitCode()).To(BeIdenticalTo(1))
			Expect(exitErr.Stderr).To(Equal([]byte("Important message\n")))
		})
	})
})
