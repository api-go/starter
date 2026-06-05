# Changelog: @go/starter

## v1.5.2 (2026-06-05)
- **可观测性升级: 中心化生命周期审计与 TraceID 链路追踪**:
  - **TraceID 分层**: 为 `starter` 编排层引入了独立的长短 ID 体系。
    - Starter 级（8位短 ID）：代表整个启动/停止批次的编排上下文。
    - 服务级（10位长 ID）：自动派发给每个子服务的 `Start` 方法，确保服务内部日志能与 Starter 状态精准关联。
  - **状态聚合**: 所有的 `service [name] starting / started / stopping / stopped` 日志现由 `starter` 统一负责，并显式记录跨层级的 `trace` ID。
  - **语义统一**: 所有 Starter 核心消息增加 `[starter]` 前缀标识。
- **架构清理**: 移除了内部对 `log-writer` 的自动注册逻辑，将核心组件的启停权责彻底交还给应用开发者。

## v1.5.1 (2026-06-04)
- **依赖升级**: 全面对接 `@go` 基础设施 v1.5.1。

## v1.5.0 (2026-05-10)
- **基础设施对齐**: 全局对齐至 v1.5.0。

## v1.0.1 (2026-05-12)

### 🚀 Features
- **Secure IPC**: Token-based authentication using **SHA256** (of secret + PID) for Unix Domain Socket communication.
- **Precise Signaling**: `kill <svc_name> <signal_num>` command for targeted service signaling.
- **Trace ID Propagation**: Automatically generates and propagates a shared Trace ID for all service startup logs.
- **Enhanced `status`**: Detailed health reporting secured by IPC token.

### 🧹 Cleanup
- **Minimalist API**: Finalized the core API: `Register`, `Run`, `AddCommand`, and `SetAppInfo`.
- **Automated PID**: Refined PID management in system temp directory.

### 🛠 Improvements
- **Race Condition Fix**: IPC server now starts only after all services are successfully initialized to prevent concurrent access panics.
- **Infrastructure Alignment**: Fully aligned with `@go` modules, including `cast.To[T]`, `timer.Retry`, and `id`.

## v1.0.0 (2026-05-10)

### 🚀 Features
- **Initial Release**: Migrated from `ssgo/starter` to `apigo.cc/go/starter`.
- **Background Mode**: Supports running services in the background with `-log` redirection.
- **PID Management**: Automatic PID file creation and cleanup.
- **Signal Handling**: Graceful shutdown on `SIGINT`/`SIGTERM`, reload support on `SIGHUP`.
- **Infrastructure Alignment**: Integrated with `go/file` and `go/log`.
