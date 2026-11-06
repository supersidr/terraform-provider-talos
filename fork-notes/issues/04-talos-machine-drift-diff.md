**Title:** `talos_machine`: show what drifted, not just a changed hash

Priority: P1 (main feature request for the fork). Cost: medium. Design to be agreed before coding.

### Problem

Drift detection in `talos_machine` is hash-only:

- `Read` reads the active `MachineConfig` from COSI and stores
  `machine_configuration_hash = sha256(normalize(live))`;
- `ModifyPlan` compares it with `sha256(normalize(desired))` and marks the hash unknown.

Result in `terraform plan` / `pulumi preview`:

```
~ machine_configuration_hash = "3f1a…" -> (known after apply)
```

No information about *what* changed. `machine_configuration` itself is `Sensitive` and is never refreshed
from the node, so even with `--show-secrets` the diff shows desired-vs-desired. Operators cannot tell an
out-of-band `talosctl edit mc` from a harmless normalization difference, and cannot review a
risky change (e.g. etcd image, network) before apply.

### Options

**A. Dry-run diff from the node (recommended first step).**
In `ModifyPlan`, when hashes differ and the node is known, call
`ApplyConfiguration{DryRun: true, Mode: AUTO, Data: desired}`. Talos ≥1.x returns
`ModeDetails` with `configdiff.DiffConfigs(current, new)` — a unified diff computed on the node
(already used by `talos_machine_configuration_apply` in `dryRunNeedsReboot`).
- Expose as a new Computed, non-sensitive attribute `machine_configuration_diff` (string, planned value =
  the diff; after apply = ""), and also `resp.Diagnostics.AddWarning("talos_machine <node>: configuration drift", diff)`
  so it is visible in the plan output even when the bridge hides computed values.
- **Secrets:** the node-side diff is NOT redacted (cluster CA keys, tokens, secretbox key can appear in
  changed lines). Must redact before exposing. Option: do not use the node's diff; instead see B.
- Cost: an extra RPC per changed node at plan time; skipped when node is unknown or hash equal.

**B. Client-side diff with redaction (safer).**
In `Read`, fetch live config from COSI (already done for the hash), load both live and desired via
`configloader.NewFromBytes`, call `.RedactSecrets("<redacted>")` on both, then
`configdiff.DiffConfigs(liveRedacted, desiredRedacted)` (pkg/machinery v1.14.1, `config/configdiff`).
- Store live redacted YAML in a Computed attribute `machine_configuration_live` (non-sensitive, redacted)
  — then the framework/bridge shows a normal attribute-level diff, and `pulumi preview --diff` shows it.
  Alternatively only compute the diff in `ModifyPlan` (needs live config: ModifyPlan has state, so keep
  redacted live YAML in state, or re-read COSI in ModifyPlan).
- With `ignore_kubernetes_upgrade_drift = true`, strip k8s image tags from both sides before diffing
  (reuse `stripK8sImages`), so the diff matches the hash semantics.
- Redaction coverage: `v1alpha1.Config.Redact` + all `config.SecretDocument` types in machinery
  (k8s CAs, service account, etcd encryption, registry auth/TLS, wireguard, discovery identity,
  volume encryption…). Verify with a test that a generated config with all secrets set produces no
  key/cert/token material after redaction.

Recommended: **B**, storing redacted live config in state as `machine_configuration_live` and adding the
diff as a plan warning. A gives the exact node view but leaks secrets.

### Open questions

- State size: a full machine config is ~10–20 KB per node; acceptable.
- Should `machine_configuration_live` be in the planned value as the desired (redacted) config, so the
  plan diff is `live -> desired` at attribute level? That is the cleanest UX for Pulumi
  (`pulumi preview --diff` prints the YAML diff). Planned value = redacted desired; Read value = redacted
  live. Then `machine_configuration_hash` becomes redundant for display but stays for compatibility.
- Pulumi bridge: verify the attribute diff renders as a multiline string diff and is not marked secret
  (bridge marks `Sensitive` attrs only).
- Talos-normalization noise: Talos may add defaults when persisting; the hash code assumes it does not
  (`talosMachineLiveConfigHash` comment). Compare normalized YAML on both sides; add a test using a
  config round-tripped through `configloader` → `EncodeBytes`.
