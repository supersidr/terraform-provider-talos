**Title:** `talos_machine_configuration`: generated multi-doc / HostnameConfig conflicts with v1alpha1 patches

Priority: P3 (we have workarounds). Cost: small for docs/validation, larger to change behaviour.
Not a provider bug strictly: this is `generate` in pkg/machinery. Kept here for completeness.

### Problems seen

1. `talos_version = 1.14.x` → generator emits multi-doc config (`KubeletConfig`,
   `UnattendedInstallConfig`, `KubeAPIServerConfig`…). Our v1alpha1 patches
   (`machine.kubelet.*`, `cluster.apiServer.*`) then fail apply with ~14 `InvalidArgument`
   ("… is already set in the v1alpha1 config"). Workaround: `config_version` (= `talos_version` input) 1.13.
2. `talos_version >= 1.12` → generator emits `HostnameConfig{auto: stable}`
   (`pkg/machinery/config/generate/network.go:40`). A patch with `machine.network.hostname` then fails with
   "static hostname is already set". Workaround: patch `{apiVersion: v1alpha1, kind: HostnameConfig, $patch: delete}`.

### Cheap improvements in the provider

- Validate at plan time: load the generated bundle with the patches applied and run
  `cfg.Validate(mode)` / `container.Validate` — then the error appears in `plan`, not at apply after a
  reboot (today `machine_configuration` is sensitive, so plan shows nothing).
- If `machine.network.hostname` is set by a patch and a `HostnameConfig` doc is generated, drop the
  generated one (mirrors what `talosctl gen config` does? — check upstream behaviour first).
- Docs: describe the version contract switch in `docs/data-sources/machine_configuration.md`.
