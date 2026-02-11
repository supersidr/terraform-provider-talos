// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package talos

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/siderolabs/talos/cmd/talosctl/pkg/talos/action"
	machineapi "github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
)

type machineUpgradeOptions struct {
	Image      string
	RebootMode string
	Preserve   bool
	Stage      bool
	Force      bool
	Wait       bool
	Timeout    time.Duration
}

func machineUpgrade(ctx context.Context, endpoint, node string, tc *clientconfig.Config, opts machineUpgradeOptions) error {
	rebootMode, err := parseUpgradeRebootMode(opts.RebootMode)
	if err != nil {
		return err
	}

	upgradeOpts := []client.UpgradeOption{
		client.WithUpgradeImage(opts.Image),
		client.WithUpgradeRebootMode(rebootMode),
		client.WithUpgradePreserve(opts.Preserve),
		client.WithUpgradeStage(opts.Stage),
		client.WithUpgradeForce(opts.Force),
	}

	if !opts.Wait {
		return talosClientOp(ctx, endpoint, node, tc, func(nodeCtx context.Context, c *client.Client) error {
			_, err := c.UpgradeWithOptions(nodeCtx, upgradeOpts...)

			return err
		})
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	return talosClientOp(ctx, endpoint, node, tc, func(_ context.Context, c *client.Client) error {
		executor := newClientExecutor(c, []string{node})

		return action.NewTracker(
			executor,
			action.MachineReadyEventFn,
			func(actionCtx context.Context, actionClient *client.Client) (string, error) {
				return upgradeGetActorID(actionCtx, actionClient, upgradeOpts)
			},
			action.WithPostCheck(action.BootIDChangedPostCheckFn),
			action.WithDebug(false),
			action.WithTimeout(timeout),
		).Run()
	})
}

func parseUpgradeRebootMode(rebootMode string) (machineapi.UpgradeRequest_RebootMode, error) {
	mode := strings.ToUpper(strings.TrimSpace(rebootMode))
	if mode == "" {
		mode = machineapi.UpgradeRequest_DEFAULT.String()
	}

	rebootModeEnum, ok := machineapi.UpgradeRequest_RebootMode_value[mode]
	if !ok {
		return machineapi.UpgradeRequest_DEFAULT, fmt.Errorf("invalid reboot mode: %s", rebootMode)
	}

	return machineapi.UpgradeRequest_RebootMode(rebootModeEnum), nil
}

func upgradeGetActorID(ctx context.Context, c *client.Client, opts []client.UpgradeOption) (string, error) {
	resp, err := c.UpgradeWithOptions(ctx, opts...)
	if err != nil {
		return "", err
	}

	if len(resp.GetMessages()) == 0 {
		return "", errors.New("no messages returned from action run")
	}

	return resp.GetMessages()[0].GetActorId(), nil
}
