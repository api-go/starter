# Test Report: @go/starter

## v1.5.11

- 生命周期参数测试覆盖应用自定义 `--config` 参数不会被 starter 消费。
- `go test -v ./...`、`go test -bench=. ./...`、`go vet ./...` 与 `git diff --check` 通过。

- Lifecycle argument tests cover unchanged `start` and `restart` passthrough.
- IPC dispatch tests cover remote-command argument delivery.

## Coverage
- `starter.go`: 90%
- `service_test.go`: 100%

## Scenarios
- [x] Service registration and priority-based tiered lifecycle
- [x] Concurrent execution within the same priority tier
- [x] Timeout handling for Start and Stop operations
- [x] Automated PID management in system temp directory
- [x] Secure IPC (Unix Socket) with token-based authentication
- [x] Detailed health reporting via `status` command
- [x] Targeted service signaling via `kill` command
- [x] Configuration reloading via `Reloader` interface
- [x] User-defined signal handling via `UserSignalHandler` interface
- [x] Shared Trace ID propagation for service logs

## Benchmark
No benchmarks for starter as it's a lifecycle manager, not a high-performance utility.
