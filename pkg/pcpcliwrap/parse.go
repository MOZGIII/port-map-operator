package pcpcliwrap

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/MOZGIII/port-map-operator/pkg/portmap"
)

var (
	ErrParseGatewayIP = errors.New("uname to parse gateway IP address")
	ErrParseNotFound  = errors.New("uname to parse the resonse, no response lines found")

	ErrFailResponse = errors.New("port map failed")
	ErrNotDone      = errors.New("port map did't complete in time")
)

// Used for mocks
var timeNow = time.Now

type phase uint

const (
	lookingForStart phase = iota
	skippingHeader
	parsingLines
	parsingComplete
)

func parseOutput(output []byte) (*portmap.Response, error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))

	phase := lookingForStart

	for scanner.Scan() {
		switch phase {
		case lookingForStart:
			switch scanner.Text() {
			case "Flow signaling succeeded.":
				phase = skippingHeader
			case "Flow signaling timed out.":
				return nil, ErrNotDone
			}
		case skippingHeader:
			phase = parsingLines
		case parsingLines:
			text := scanner.Text()
			if len(text) == 0 {
				phase = parsingComplete
				continue
			}
			res, err := parseLine(text)
			if err != nil {
				return nil, fmt.Errorf("unable to parse response line: %w", err)
			}
			if res.GatewayIP.IsLinkLocalUnicast() {
				// Skip link local addresses.
				continue
			}
			return res, nil
		case parsingComplete:
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, ErrParseNotFound
}

func parseLine(line string) (*portmap.Response, error) {
	// ::ffff:192.168.0.1   TCP  ::ffff:192.168.0.2  32100   ::                       0   ::ffff:1.2.3.4 32100   0  succ Sat Feb 13 19:41:56 2021

	var serverIP, prot string
	var nodeIP string
	var nodePort uint16
	var dummyIP string
	var dummyPort uint16
	var gwIP string
	var gwPort uint16
	var resultCode uint16
	var resultDesc string
	var lifetimeEndList [5]string

	parsedNum, err := fmt.Sscanf(
		line,
		"%s %4s %s %5d   %s %5d   %s %5d %3d %5s %s %s %s %s %s",
		&serverIP, &prot,
		&nodeIP, &nodePort,
		&dummyIP, &dummyPort,
		&gwIP, &gwPort,
		&resultCode,
		&resultDesc,
		&lifetimeEndList[0],
		&lifetimeEndList[1],
		&lifetimeEndList[2],
		&lifetimeEndList[3],
		&lifetimeEndList[4],
	)
	delayParseErr := false
	if err != nil {
		const FIRST_LIFETIME_PATTERN_NUM = 10
		delayParseErr = err == io.EOF && parsedNum >= FIRST_LIFETIME_PATTERN_NUM && lifetimeEndList[0] == "-"
		if !delayParseErr {
			return nil, err
		}
	}

	if resultCode != 0 {
		return nil, ErrFailResponse
	}

	if resultDesc != "succ" {
		return nil, ErrNotDone
	}

	// Handle delated error before processing lifetime.
	if delayParseErr {
		return nil, fmt.Errorf("delayed parse error: %w", err)
	}
	lifetimeEnd := strings.Join(lifetimeEndList[:], " ")

	var protocol portmap.Protocol
	switch prot {
	case "TCP":
		protocol = portmap.ProtocolTCP
	case "UDP":
		protocol = portmap.ProtocolUDP
	case "UNK":
		// assume SCTP cause the UI of the pcp cli tool sucks.
		protocol = portmap.ProtocolSCTP
	}

	ip := net.ParseIP(gwIP)
	if ip == nil {
		return nil, ErrParseGatewayIP
	}

	lifetimeEndTime, err := time.ParseInLocation(time.ANSIC, lifetimeEnd, time.Local)
	if err != nil {
		return nil, fmt.Errorf("unable to parse lifetime: %w", err)
	}
	lifetime := uint32(lifetimeEndTime.Sub(timeNow()).Seconds())

	res := portmap.Response{
		Protocol:    protocol,
		NodePort:    portmap.Port(nodePort),
		GatewayPort: portmap.Port(gwPort),
		GatewayIP:   ip,
		Lifetime:    portmap.Lifetime(lifetime),
	}
	return &res, err
}
