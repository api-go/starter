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
