# Changelog: @go/starter

## v1.0.0 (2026-05-10)

### 🚀 Features
- **Initial Release**: Migrated from `ssgo/starter` to `apigo.cc/go/starter`.
- **Background Mode**: Supports running services in the background with `-log` redirection.
- **PID Management**: Automatic PID file creation and cleanup.
- **Signal Handling**: Graceful shutdown on `SIGINT`/`SIGTERM`, reload support on `SIGHUP`.
- **Infrastructure Alignment**: Integrated with `go/file` and `go/log`.
