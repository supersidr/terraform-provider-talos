**Title:** `talos_machine`: OS upgrade failures hide the installer output

Priority: P1. Cost: small.

### Problem

`talosMachineInstallImage` (`pkg/talos/talos_machine_resource.go`) reads the
`LifecycleService.Upgrade` stream and discards every progress message; on failure it returns only

```
installing new OS image: upgrade exited with code 1
```

The reason (in our case `grubUseUKICmdline` mismatch, earlier — a broken installer image in the private
registry) is visible only with `talosctl upgrade --debug`. With the Pulumi bridge there is no way to see it
at all, so every failed upgrade turns into a manual `talosctl` session.

### Proposal

Collect `resp.GetProgress().GetMessage()` lines (bounded ring buffer, e.g. last 50 lines), log each one
with `tflog.Debug`, and append the tail to the error when `ExitCode != 0` or the stream errors:

```
installing new OS image: upgrade exited with code 1; installer output (last 50 lines):
  ...
```

Same for `talosMachinePullImage` stream errors (include image ref) and for the legacy
`UpgradeWithOptions` path (`talosMachineUpgradeLegacy`) — include image ref in the error.

### Implementation sketch

- helper `type outputTail struct{ lines []string; max int }` with `Add`/`String`.
- unit test: fake stream (interface over `Recv()`), assert error contains the last lines and exit code.
  Pull the loop out into `consumeInstallStream(stream interface{ Recv() (*machineapi.LifecycleServiceUpgradeResponse, error) }) error`.
