# Форк terraform-provider-talos: контекст для продолжения

Дата: 2026-11-06. Ветка: `fork/okko-issues` (от `main` = v0.12.0-4-g77d43fb; upstream/main впереди на
1 коммит — d0d4997, bump klauspost/compress).

## Откуда задача

Провайдер используется из Pulumi (репозиторий `pulumi-modules`, модуль `modules/talos`, ветка IDP-7499)
через параметризованный SDK `pulumi_talos==0.12.0.post1` (terraform-provider bridge, плагин 1.3.0).
Ресурсы: `talos.Machine`, `MachineSecrets`, `MachineBootstrap`, data source `get_machine_configuration`.
Образы — из приватного `registry.playteam.ru`, у нод нет доступа к `registry.k8s.io` / `ghcr.io`.

Пользователь хочет:

1. Завести в форке главные претензии, которые дёшево исправить в коде → `fork-notes/issues/*.md`.
2. Сделать «нормальное чтение конфигурации», чтобы при дрифте было видно, что изменилось → issue 04.

## Issues (по приоритету)

| # | Файл | Суть | Цена |
|---|---|---|---|
| 01 | `issues/01-talos-cluster-k8s-images.md` | `talos_cluster`: захардкожены образы и `PrePullImages` в `upgrade-k8s` (`talos_cluster_resource.go:447`). Блокирует обновление k8s через `talos.Cluster`. Upstream не исправлено | малая |
| 04 | `issues/04-talos-machine-drift-diff.md` | Дрифт виден только как смена `machine_configuration_hash`. Предложение — redacted live config в state + diff | средняя, нужен дизайн |
| 02 | `issues/02-talos-machine-installer-errors.md` | Ошибка installer теряется: `upgrade exited with code 1` без вывода | малая |
| 03 | `issues/03-talos-machine-boot-wait-diagnostics.md` | После reboot таймаут без причины (UnmetConditions, DHCP) | малая–средняя |
| 05 | `issues/05-machine-configuration-version-contract.md` | multi-doc 1.14 и HostnameConfig против v1alpha1-патчей. Скорее machinery; дешёвое — валидация в plan | малая (валидация) |

Не к провайдеру (в issues не заносим): несовместимость токенов pulumiverse_talos → `talos:index/*`,
устаревший SDK в Artifactory, игнор `opts.version`, конфликт пакета `talos` в venv — это упаковка
Pulumi-SDK. etcd 3.7 на Talos 1.13 и битые образы в registry — Talos/registry.

## Ключевые места кода

- `pkg/talos/talos_machine_resource.go`
  - `Read` (~520): Version RPC → `image` из тега; `talosMachineLiveConfigHash` (~1578) читает
    `configresource.MachineConfig` (ActiveID) из COSI → `computeConfigHash`.
  - `ModifyPlan` (~362): hash desired vs state → unknown при расхождении.
  - `talosMachineInstallImage` (~1229): поток LifecycleService.Upgrade, сообщения отбрасываются.
  - `talosMachineReboot` (~1346), `talosMachineWaitForBoot` / `talosMachineCheckBootReady` (~929–1000).
- `pkg/talos/config_hash.go`: нормализация YAML, `stripK8sImages` для `ignore_kubernetes_upgrade_drift`.
- `pkg/talos/talos_cluster_resource.go`: `talosClusterUpgradeKubernetes` (~447).
- `pkg/talos/talos_machine_configuration_apply_resource.go`: `dryRunNeedsReboot` (~886) — пример
  `ApplyConfiguration{DryRun: true}`.
- `pkg/talos/util.go`: `machineConfigGenerateOptions.generate` (version contract, патчи).
- machinery v1.14.1 (`go mod download -json github.com/siderolabs/talos/pkg/machinery`):
  `config/configdiff.DiffConfigs`, `config.Provider.RedactSecrets`, `config/configloader`.
- Talos server dry-run: `internal/app/machined/internal/server/v1alpha1/v1alpha1_server.go:254`
  (`generateDiff` — diff НЕ redacted).

## Сборка и проверка

- Go 1.26.10 (toolchain скачивается автоматически). Unit-тесты: `go test ./pkg/talos/ -run <Name>`.
  Acceptance (`make testacc`, `TF_ACC=1`) требует кластер — локально не запускать.
- Доки генерируются `make docs` (tfplugindocs) — после изменения схем.
- Проверка в Pulumi: собрать бинарник, положить как плагин `pulumi-resource-terraform-provider`
  с параметризацией на локальный путь — см. память pulumi-modules «Локальные SDK: --no-build-isolation».

## Статус

- **01 — сделано** в ветке `feat/talos-cluster-k8s-images` (от `main`, без fork-notes — пригодна для PR
  в upstream), коммит 119867c. Атрибуты `Optional` без Computed/Default: null = константы Talos, у
  существующего state нет diff. Валидатор `imageRepositoryValid` (util.go) отвергает тег/digest/пустую
  строку. Unit-тесты: `talos_cluster_upgrade_options_internal_test.go`. Доки `docs/resources/cluster.md`
  перегенерированы. Не проверено на живом кластере; не запушено.
- `go test ./pkg/talos/` без кластера: падают `TestAccTalosMachineSecretsResourceUpgrade1..3` — так же на
  чистом `main`, не регрессия.

## Следующие шаги

1. Проверить 01 на тестовом кластере (через Pulumi: `talos.Cluster` с `*_image` из registry.playteam.ru)
   и решить, отправлять ли PR в upstream.
2. Согласовать с пользователем дизайн issue 04 (вариант B: `machine_configuration_live`, redacted).
3. 02 и 03 — по желанию, тоже кандидаты в upstream.
