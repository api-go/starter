## v1.0.1 (2026-05-12)

### 🚀 Features
- **Secure IPC**: SHA256 token-based authentication for Unix Domain Sockets.
- **Precise Signaling**: `kill <svc_name> <signal_num>` command.
- **Trace ID Propagation**: Shared Trace ID for service startup logs.
- **Enhanced `status`**: Secured detailed health reporting.

### 🧹 Cleanup
- **Minimalist API**: `Register`, `Run`, `AddCommand`, `SetAppInfo`.
- **Automated PID**: System temp directory placement.

### 🛠 Improvements
- **Race Condition Fix**: Delayed IPC server activation until startup completion.
- **Infrastructure Alignment**: `cast.To[T]`, `timer.Retry`, and `id`.
