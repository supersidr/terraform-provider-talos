**Title:** `talos_cluster`: allow overriding Kubernetes component images (and pre-pull) for `upgrade-k8s`

Priority: P0 (blocks k8s upgrades via `talos_cluster` for private-registry clusters). Cost: small.
Status upstream: still hardcoded in `upstream/main` (checked 2026-11-06, d0d4997).

### Problem

`talos_cluster` runs the Kubernetes upgrade with hardcoded image repositories
(`pkg/talos/talos_cluster_resource.go`, `talosClusterUpgradeKubernetes`, v0.12.0):

```go
opts := k8s.UpgradeOptions{
    PrePullImages:          true,
    ...
    KubeletImage:           constants.KubeletImage,                     // ghcr.io/siderolabs/kubelet
    APIServerImage:         constants.KubernetesAPIServerImage,         // registry.k8s.io/kube-apiserver
    ControllerManagerImage: constants.KubernetesControllerManagerImage, // registry.k8s.io/kube-controller-manager
    SchedulerImage:         constants.KubernetesSchedulerImage,         // registry.k8s.io/kube-scheduler
    ProxyImage:             constants.KubeProxyImage,                   // registry.k8s.io/kube-proxy
}
```

`pkg/cluster/kubernetes` builds image refs as `fmt.Sprintf("%s:v%s", options.<X>Image, version)`, so on
upgrade the machine config is patched from e.g. `registry.example.com/kube-apiserver:v1.34.3`
to `registry.k8s.io/kube-apiserver:v1.35.0`.

For clusters that use a private registry (air-gapped, or a policy that requires images to come from an
internal registry), this means:

- pre-pull and the upgrade fail when nodes cannot reach `registry.k8s.io` / `ghcr.io`;
- after a successful upgrade, the custom image repositories in the machine config are silently replaced.

`talosctl upgrade-k8s` handles this with `--kubelet-image`, `--apiserver-image`,
`--controller-manager-image`, `--scheduler-image`, `--proxy-image`, and `--pre-pull-images=false`.
`talos_cluster` has no equivalent.

### Proposal

Optional attributes on `talos_cluster`, 1:1 with `UpgradeOptions` / `talosctl upgrade-k8s` flags:

```hcl
resource "talos_cluster" "this" {
  kubernetes_version = "v1.35.0"
  # ...

  kubelet_image            = "registry.example.com/siderolabs/kubelet"   # default: constants.KubeletImage
  apiserver_image          = "registry.example.com/kube-apiserver"       # default: constants.KubernetesAPIServerImage
  controller_manager_image = "registry.example.com/kube-controller-manager"
  scheduler_image          = "registry.example.com/kube-scheduler"
  proxy_image              = "registry.example.com/kube-proxy"
  pre_pull_images          = true                                        # default: true
}
```

Changing only these attributes must not trigger an upgrade (only a `kubernetes_version` diff runs `upgrade-k8s`, as today).

`ignore_kubernetes_upgrade_drift` on `talos_machine` already strips tags only and keeps the repository,
so it stays consistent with this change.

### Implementation sketch (fork)

- `talosClusterResourceModel`: add 5 `types.String` + `PrePullImages types.Bool`.
- Schema: `Optional+Computed` with `stringdefault.StaticString(constants.<X>Image)` /
  `booldefault.StaticBool(true)`; validator: reject a value containing a tag or digest
  (`stripImageTag(v) != v` or `@`), because `upgrade-k8s` appends `:v<version>` itself.
- `talosClusterUpgradeKubernetes(ctx, endpoint, cfg, toVersion, imgs)`: take the values from the plan.
- Existing state upgrades: Computed+Default fills defaults → plan shows a diff of the new attrs only,
  Update does nothing because `kubernetes_version` is unchanged. Check that no state upgrader is needed
  (schema version unchanged; new attrs are null in old state → defaults applied in plan).
- Tests: unit test that options are passed through (extract `talosClusterUpgradeOptions(plan)` →
  `k8s.UpgradeOptions` and test it without a cluster); schema/validator tests; docs `docs/resources/cluster.md`
  via `make docs` (tfplugindocs).

### Environment

- terraform-provider-talos v0.12.0 (used via Pulumi `terraform-provider` bridge, plugin 1.3.0)
- Talos v1.13 / v1.14, Kubernetes v1.34 → v1.35
- Images served from an internal registry; nodes have no access to `registry.k8s.io` / `ghcr.io`
