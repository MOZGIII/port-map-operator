package pcpcliwrap

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/MOZGIII/port-map-operator/pkg/portmap"
)

type Command struct {
	CommandName string

	// If empty, server autodiscovery will be attempted.
	// Might fail though, depending on the runtime environment.
	ServerAddr string
}

func (c *Command) Exec(ctx context.Context, req *portmap.Request) (*portmap.Response, error) {
	cmd := c.prepareCommand(ctx, req)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseOutput(out)
}

func (c *Command) prepareCommand(ctx context.Context, req *portmap.Request) *exec.Cmd {
	args := []string{
		"--protocol", fmt.Sprintf("%d", req.Protocol),
		"--internal", fmt.Sprintf(":%d", req.NodePort),
		"--external", fmt.Sprintf(":%d", req.GatewayPort),
		"--lifetime", fmt.Sprintf("%d", req.Lifetime),
	}

	if c.CommandName != "" {
		args = append(args, "--server", c.ServerAddr)
	}

	// nolint: gosec
	return exec.CommandContext(ctx, c.CommandName, args...)
}
