# Changelog: @go/starter

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
