# Test Report: @go/starter

## v1.5.12

- `TestHiddenRemoteCommandIsCallableButNotShownInHelp` 验证隐藏 IPC 命令可调用但不出现在用户帮助中。
- `go test ./... -count=1`、`go test -bench=. ./...`、`go vet ./...` 与 `git diff --check` 通过。

## v1.5.11

- 生命周期参数测试覆盖应用自定义 `--config` 参数不会被 starter 消费。
- `go test -v ./...`、`go test -bench=. ./...`、`go vet ./...` 与 `git diff --check` 通过。

- Lifecycle argument tests cover unchanged `start` and `restart` passthrough.
- IPC dispatch tests cover remote-command argument delivery.
- Hidden remote-command tests cover authenticated dispatch without help output.

## Coverage
- `starter.go`: 90%
- `service_test.go`: 100%

## Scenarios
- [x] Service registration and priority-based tiered lifecycle
- [x] Concurrent execution within the same priority tier
- [x] Cooperative Start context timeout; enforced Stop deadline including hooks and retries
- [x] Automated PID management in system temp directory
- [x] Secure IPC (Unix Socket) with token-based authentication
- [x] Detailed health reporting via `status` command
- [x] Targeted service signaling via `kill` command
- [x] Configuration reloading via `Reloader` interface
- [x] User-defined signal handling via `UserSignalHandler` interface
- [x] Shared Trace ID propagation for service logs

## Benchmark
No benchmarks for starter as it's a lifecycle manager, not a high-performance utility.

## v1.5.13 发布验证（2026-10-10）

- GOWORK=off go test -v ./... 与 go test -bench=. ./... 通过，依赖均为已发布 tag。
- API 以 GOWORK=off go doc 与导出源码交叉核对（getExports 工具不可用）。

## v1.5.14 发布验证（2026-10-26）

GOWORK=off，正式依赖 crypto v1.5.9，无 replace。`go test -v ./...`、`go test -bench=. ./...`、`go vet ./...`、`go mod verify` 通过。使用 `go doc` 与源码交叉核对 API，替代 getExports。现有回归覆盖本模块业务；新旧 GCM 格式、认证失败和性能基准见 crypto v1.5.9。

本次功能测试无跳过。

本模块未定义 Benchmark；基准命令已执行并通过。

## 退出修复验证（2026-10-30，未发布）

在dev1固定Go 1.26.8镜像验证。新增外部测试包的真实隔离进程覆盖忽略context的Stop、前后停止钩子阻塞、永久错误重试总期限、默认期限、整体期限、再次SIGINT、正常退出状态及低优先级清理。原有失败优先级重试顺序测试保留。全部不启动业务或媒体。

## GitHub migration validation (pending release)

Module paths and internal dependency versions are migrated. Development verification uses GOWORK=off and explicit sibling replacements; remove these replacements before release. See migration validation logs for actual results. Existing unrelated working changes are preserved and require separate review.

API inventory verification: `GOWORK=off go doc` succeeded; output retained in the migration validation directory. Source exports were preserved by the path migration.

## GitHub release verification

GOWORK=off go test -v ./... and go test -bench=. ./... passed using published GitHub dependencies, without local replacements. go mod verify passed.

## 内置日志命令 — 2026-11-12

子进程验证 starter log 的真实命令分流、文件参数与 tail 范围。GOWORK=off，使用 /tmp/apigo-validation-starter.mod 显式替换本次 log/timer；API 用 go doc 与源码核对。

## 日志依赖联动 — 2026-11-12

开发验证使用 GOWORK=off 与 go.mod 的显式本地 replace；执行完整测试、全量基准（benchtime=100ms），API 用 go doc 与导出源码交叉核对。结果记录在 /tmp/apigo-starter-integration-*.txt。


## v1.5.16 发布验证 — 2026-11-12

GOWORK=off；正式 GitHub 依赖，无本地 replace。完整 go test -v ./... 与 go test -bench=. -benchtime=100ms ./... 通过；Service/Gateway 设置 GODEBUG=http2xconnect=1。API 核对使用 go doc . 与导出源码（getExports 工具不可用）。可选外部环境测试按原规则跳过。详细输出 /tmp/apigo-starter-release-tests.txt 与 /tmp/apigo-starter-release-bench.txt。

## 依赖联动验证（2026-11-14，未发布）

GOWORK=off，显式替换 safe/crypto；test: 0, bench: 0。0 表示通过；非零或超时结果待定位，不能视为验收通过。

## 最终依赖图验证（2026-11-14，未发布）

显式本地 replace 覆盖本轮待发布依赖链；GOWORK=off go mod tidy、go test -v ./...、go test -bench=. -benchtime=1x ./...。基准按固定 1 次采样，避免排除计时的昂贵准备阶段反复校准。本轮结果：[('tidy', '0'), ('test', '0'), ('bench', '0')]。0 为通过，其他结果待定位。

## v1.5.17 正式发布验证（2026-11-14）

GOWORK=off；移除全部 replace，使用已发布依赖。go mod tidy、go test -v ./...、go test -bench=. ./...、go mod verify 均通过。API 工具不可用，使用 go doc 与源码核对。
