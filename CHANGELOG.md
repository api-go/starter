# Changelog: @go/starter

## Unreleased · 2026-10-30
- Stop、停止钩子与错误重试共享有界期限；未指定停止期限默认5秒，整体退出默认15秒。
- 超时后尽力清理低优先级依赖；未完成退出返回状态1，再次终止信号立即退出，不再宣称成功。
- 增加真实隔离子进程回归：阻塞Stop/钩子、永久错误、默认/整体期限、二次Ctrl+C及正常退出。

## v1.5.14（2026-10-26）
- 升级正式依赖以统一选用 crypto v1.5.9；AES-GCM 公共解密自动兼容内嵌随机 IV 和原固定 IV 密文。
- 现有公开 API 不变；接收端升级后再启用自动 IV 写入，原 CBC/SM4 算法格式不变。

## [v1.5.13] - 2026-10-10
- 对齐 Log v1.5.17 及当前已发布的上游依赖。

## v1.5.12 (2026-09-18)

- Add `AddHiddenRemoteCommand` for internal IPC commands that should not appear in user-facing CLI help.
- 对齐 `cast v1.5.8` 与 `log v1.5.16`。

## v1.5.11 (2026-09-12)

- 不再预解析和拒绝应用自定义参数，使 `--config` 等宿主参数原样保留给应用配置层。
- 升级 `config v1.5.8`。

## v1.5.10 (2026-09-12)

- Preserve all application arguments after `start` and `restart`, including unknown flags such as `--dev`.
- Add `AddRemoteCommand` for authenticated, user-local IPC commands executed by the running service process.

## v1.5.8 (2026-09-05)
- **依赖对齐**: 升级 `config`、`file` 与 `log` 至当前已发布版本。

## v1.5.7 (2026-09-05)
- **依赖对齐**: 升级全部 `apigo.cc/go/*` 依赖至当前已发布版本。

## v1.5.5 (2026-06-21)
- **依赖对齐**:
  - 升级依赖 `jsmod` 至 `v1.5.3`, `cast` 至 `v1.5.3`, `rand` 至 `v1.5.3`, `encoding` 至 `v1.5.4`, `shell` 至 `v1.5.3`, `safe` 至 `v1.5.2`, `id` 至 `v1.5.4`, `crypto` 至 `v1.5.3`, `file` 至 `v1.5.5`, `config` 至 `v1.5.3`, `log` 至 `v1.5.8`。

## v1.5.4 (2026-06-11)
- **依赖对齐**:
  - 对齐基础设施依赖到 v1.5.x。

## v1.5.3 (2026-06-07)
- **重大架构更迭: 引入 Start/Wait 编排模式与生命周期钩子**:
  - **废弃 `Run()`**: 彻底移除黑盒式的 `Run()` 方法，改为显式的 `Start()` (解析与非阻塞启动) 与 `Wait()` (阻塞与信号处理) 模式。
  - **生命周期钩子**: `Register` 现返回 `*managedService` 指针，支持链式注册 `OnStarting`, `OnStarted`, `OnStopping`, `OnStopped` 回调。
  - **中间状态编排**: 允许开发者在 `Start()` 之后、`Wait()` 之前执行自定义初始化逻辑（如跨服务的业务对齐）。

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
