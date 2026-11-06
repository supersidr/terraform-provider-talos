<!-- Open: https://github.com/siderolabs/terraform-provider-talos/compare/main...supersidr:terraform-provider-talos:feat/talos-cluster-k8s-images?expand=1 -->

**Title:** feat(talos_cluster): allow overriding Kubernetes images for upgrade-k8s

### Problem

`talos_cluster` runs `upgrade-k8s` with hardcoded image repositories and pre-pull always enabled
(`talosClusterUpgradeKubernetes`):

```go
PrePullImages:          true,
KubeletImage:           constants.KubeletImage,                     // ghcr.io/siderolabs/kubelet
APIServerImage:         constants.KubernetesAPIServerImage,         // registry.k8s.io/kube-apiserver
ControllerManagerImage: constants.KubernetesControllerManagerImage,
SchedulerImage:         constants.KubernetesSchedulerImage,
ProxyImage:             constants.KubeProxyImage,
```

On clusters that pull Kubernetes images from a private registry this:

- replaces the custom image repositories in the machine config with the public ones on upgrade;
- fails pre-pull (and the upgrade) on nodes that cannot reach `registry.k8s.io` / `ghcr.io`.

`talosctl upgrade-k8s` covers this with `--kubelet-image`, `--apiserver-image`,
`--controller-manager-image`, `--scheduler-image`, `--proxy-image` and `--pre-pull-images`;
`talos_cluster` had no equivalent.

### Change

New optional attributes on `talos_cluster`, mapped 1:1 to `k8s.UpgradeOptions`:

```hcl
resource "talos_cluster" "this" {
  kubernetes_version = "v1.36.0"
  # ...
  kubelet_image            = "registry.example.com/siderolabs/kubelet"
  apiserver_image          = "registry.example.com/kube-apiserver"
  controller_manager_image = "registry.example.com/kube-controller-manager"
  scheduler_image          = "registry.example.com/kube-scheduler"
  proxy_image              = "registry.example.com/kube-proxy"
  pre_pull_images          = false
}
```

- Attributes are `Optional` only (no `Computed`/default): null keeps the Talos defaults, so existing
  state produces no plan diff after the provider upgrade.
- Changing only these attributes does not run an upgrade; `kubernetes_version` remains the trigger.
- A repository with a tag or digest is rejected at plan time, since `upgrade-k8s` appends
  `:v<kubernetes_version>` itself.
- `ignore_kubernetes_upgrade_drift` on `talos_machine` strips only tags, so it stays consistent.

### Tests

- Unit: `talosClusterUpgradeOptions` defaults and overrides; `imageRepositoryValid` cases
  (tag, digest, registry port, empty, null/unknown).
- `golangci-lint run ./pkg/talos/` — 0 issues; docs regenerated with tfplugindocs.
- Acceptance tests not run locally (no libvirt environment).
