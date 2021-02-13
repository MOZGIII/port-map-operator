package pcpcliwrap

import (
	"io"
	"net"
	"time"

	"github.com/MOZGIII/port-map-operator/pkg/portmap"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const SAMPLE_OK = `
Flow signaling succeeded.
PCP Server IP        Prot Int. IP               port   Dst. IP               port   Ext. IP               port Res State Ends
::ffff:192.168.0.1   TCP  ::ffff:192.168.0.2   32100   ::                       0   ::ffff:1.2.3.4 32100   0  succ Sat Feb 13 19:41:56 2021
fe80::abcd:abcd:abcd:abcd TCP  fe80::abcd:abcd:abcd:ffff 32100   ::                       0   fe80::abcd:abcd:abcd:ffff 32100   0  succ Sat Feb 13 19:41:56 2021
`

const SAMPLE_OK_LOCAL_FIRST = `
Flow signaling succeeded.
PCP Server IP        Prot Int. IP               port   Dst. IP               port   Ext. IP               port Res State Ends
fe80::abcd:abcd:abcd:abcd TCP  fe80::abcd:abcd:abcd:ffff 32100   ::                       0   fe80::abcd:abcd:abcd:ffff 32100   0  succ Sat Feb 13 19:41:56 2021
::ffff:192.168.0.1   TCP  ::ffff:192.168.0.2   32100   ::                       0   ::ffff:1.2.3.4 32100   0  succ Sat Feb 13 19:41:56 2021
`

const SAMPLE_INVALID_NO_HEADER = `
Flow signaling succeeded.
`

const SAMPLE_INVALID_NO_LINES = `
Flow signaling succeeded.
PCP Server IP        Prot Int. IP               port   Dst. IP               port   Ext. IP               port Res State Ends
`

const SAMPLE_INVALID_LINES = `
Flow signaling succeeded.
PCP Server IP        Prot Int. IP               port   Dst. IP               port   Ext. IP               port Res State Ends
qwerty
`

const SAMPLE_INVALID_TIMEOUT = `
Flow signaling timed out.
PCP Server IP        Prot Int. IP               port   Dst. IP               port   Ext. IP               port Res State Ends
::ffff:192.168.0.1   TCP  ::ffff:192.168.0.2   32100   ::                       0   ::                   32100   0  proc  -
fe80::abcd:abcd:abcd:abcd TCP  fe80::abcd:abcd:abcd:ffff 32100   ::                       0   ::                   32100   0  proc  -
`

var _ = Describe("parseOutput", func() {
	var (
		sampleOutput string
		res          *portmap.Response
		err          error
	)

	JustBeforeEach(func() {
		timeNow = mockTime
		res, err = parseOutput([]byte(sampleOutput))
		timeNow = time.Now
	})

	Context("with successful sample output", func() {
		BeforeEach(func() {
			sampleOutput = SAMPLE_OK
		})

		It("should produce a correct response", func() {
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(32100),
				GatewayIP:   net.IPv4(1, 2, 3, 4),
				Lifetime:    portmap.Lifetime(120),
			}))
		})
	})

	Context("with successful sample output with local IPv6 address first", func() {
		BeforeEach(func() {
			sampleOutput = SAMPLE_OK_LOCAL_FIRST
		})

		It("should produce a correct response", func() {
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(32100),
				GatewayIP:   net.IPv4(1, 2, 3, 4),
				Lifetime:    portmap.Lifetime(120),
			}))
		})
	})

	Context("with empty sample output", func() {
		BeforeEach(func() {
			sampleOutput = ``
		})

		It("should produce an expected error", func() {
			Expect(err).To(MatchError(ErrParseNotFound))
			Expect(res).To(BeNil())
		})
	})

	Context("with sample output with no header", func() {
		BeforeEach(func() {
			sampleOutput = SAMPLE_INVALID_NO_HEADER
		})

		It("should produce an expected error", func() {
			Expect(err).To(MatchError(ErrParseNotFound))
			Expect(res).To(BeNil())
		})
	})

	Context("with sample output with no lines", func() {
		BeforeEach(func() {
			sampleOutput = SAMPLE_INVALID_NO_LINES
		})

		It("should produce an expected error", func() {
			Expect(err).To(MatchError(ErrParseNotFound))
			Expect(res).To(BeNil())
		})
	})

	Context("with sample output with no lines", func() {
		BeforeEach(func() {
			sampleOutput = SAMPLE_INVALID_LINES
		})

		It("should produce an expected error", func() {
			Expect(err).To(MatchError("unable to parse response line: EOF"))
			Expect(err).To(MatchError(io.EOF)) // should also be an EOF error
			Expect(res).To(BeNil())
		})
	})

	Context("with sample output with timeout", func() {
		BeforeEach(func() {
			sampleOutput = SAMPLE_INVALID_TIMEOUT
		})

		It("should produce an expected error", func() {
			Expect(err).To(MatchError(ErrNotDone))
			Expect(res).To(BeNil())
		})
	})
})

var _ = Describe("parseLine", func() {
	var (
		sampleLine string
		res        *portmap.Response
		err        error
	)

	JustBeforeEach(func() {
		timeNow = mockTime
		res, err = parseLine(sampleLine)
		timeNow = time.Now
	})

	Context("with valid IPv4 sample line", func() {
		BeforeEach(func() {
			sampleLine = `::ffff:192.168.0.1   TCP  ::ffff:192.168.0.2  32100   ::                       0   ::ffff:1.2.3.4 32101   0  succ Sat Feb 13 19:41:56 2021`
		})

		It("should produce a correct response", func() {
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(32101),
				GatewayIP:   net.IPv4(1, 2, 3, 4),
				Lifetime:    portmap.Lifetime(120),
			}))
		})
	})

	Context("with valid IPv6 sample line", func() {
		BeforeEach(func() {
			sampleLine = `fe80::abcd:abcd:abcd:abcd TCP  fe80::abcd:abcd:abcd:ffff 32100   ::                       0   fe80::abcd:abcd:abcd:aaaa 32101   0  succ Sat Feb 13 19:41:56 2021`
		})

		It("should produce a correct response", func() {
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&portmap.Response{
				Protocol:    portmap.ProtocolTCP,
				NodePort:    portmap.Port(32100),
				GatewayPort: portmap.Port(32101),
				GatewayIP:   net.ParseIP("fe80::abcd:abcd:abcd:aaaa"),
				Lifetime:    portmap.Lifetime(120),
			}))
		})
	})

	Context("with a fail line", func() {
		BeforeEach(func() {
			sampleLine = `::ffff:172.17.0.1    TCP  ::ffff:172.17.0.2    32100   ::                       0   ::                   32100   1  fail  -`
		})

		It("should produce an io.EOF error", func() {
			Expect(err).To(BeIdenticalTo(ErrFailResponse))
			Expect(res).To(BeNil())
		})
	})

	Context("with a proc line", func() {
		BeforeEach(func() {
			sampleLine = `::ffff:172.17.0.1    TCP  ::ffff:172.17.0.2    32100   ::                       0   ::                   32100   0  proc  -`
		})

		It("should produce an expected error", func() {
			Expect(err).To(BeIdenticalTo(ErrNotDone))
			Expect(res).To(BeNil())
		})
	})

	Context("with a slerr line (short lifetime error)", func() {
		BeforeEach(func() {
			sampleLine = `::ffff:192.168.0.1   TCP  ::ffff:192.168.0.2   32101   ::                       0   ::                      80   8 slerr  -`
		})

		It("should produce an expected error", func() {
			Expect(err).To(BeIdenticalTo(ErrFailResponse))
			Expect(res).To(BeNil())
		})
	})

	Context("with a fail line (2)", func() {
		BeforeEach(func() {
			sampleLine = `::ffff:192.168.0.1   TCP  ::ffff:192.168.0.2   32101   ::                       0   ::                      80   2  fail  -`
		})

		It("should produce an expected error", func() {
			Expect(err).To(BeIdenticalTo(ErrFailResponse))
			Expect(res).To(BeNil())
		})
	})

	Context("with empty line", func() {
		BeforeEach(func() {
			sampleLine = ``
		})

		It("should produce an io.EOF error", func() {
			Expect(err).To(BeIdenticalTo(io.EOF))
			Expect(res).To(BeNil())
		})
	})
})

func mockTime() time.Time {
	time, err := time.ParseInLocation(time.ANSIC, "Sat Feb 13 19:39:56 2021", time.Local)
	if err != nil {
		panic(err)
	}
	return time
}
