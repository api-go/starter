# @go/starter

Service starter for @go applications, supporting background mode, PID management, and signal handling.

## Installation

```bash
go get github.com/api-go/starter
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"github.com/api-go/log"
	"github.com/api-go/starter"
)

type MyService struct {}

func (s *MyService) Start(ctx context.Context, logger *log.Logger) error {
	logger.Info("MyService started")
	return nil
}

func (s *MyService) Stop(ctx context.Context) error {
	log.DefaultLogger.Info("MyService stopped")
	return nil
}

func (s *MyService) Health() error {
	return nil
}

// Optional: Support configuration reload
func (s *MyService) Reload() error {
	log.DefaultLogger.Info("Reloading config...")
	return nil
}

func main() {
	starter.SetAppInfo("myapp", "1.0.0")

	// Register with priority 1 and optional hooks
	starter.Register("myservice", &MyService{}, 1, 0, 0).OnStarted(func() {
		fmt.Println("MyService is ready!")
	})

	if err := starter.Start(); err != nil {
		panic(err)
	}
	
	// Custom initialization logic here
	
	starter.Wait()
}
```

## Interfaces

- **`Service`**: Core lifecycle interface (`Start`, `Stop`, `Status`).
- **`Reloader`**: Optional interface for services that support `Reload()` (triggered by `SIGHUP`).
- **`UserSignalHandler`**: Optional interface for services that handle custom user signals (`SIGUSR1`, `SIGUSR2` or via `kill` command).

## Features

- **Lifecycle Hooks**: `OnStarting`, `OnStarted`, `OnStopping`, `OnStopped` hooks for precise coordination.
- **Start/Wait Pattern**: Replaces `Run()` to allow custom logic between service startup and blocking.
- **Tiered Startup/Shutdown**: Concurrent execution within priority levels, serial execution across them.
- **Trace ID Propagation**: Automatically generates a shared Trace ID for all services during startup, ensuring log correlation.
- **Secure IPC**: Token-based Unix Domain Socket for `status` and `kill` commands.
- **Zero Configuration PID**: Automatic PID management in the system temporary directory.
- **Embedded Log Writer**: Automatically registered log writer service with high priority (-100).

## Commands

- `start`: Start the service in background.
- `stop`: Stop the service.
- `restart`: Restart the service.
- `status`: Show service status (including detailed health of each registered service via secure IPC).
- `kill <svc_name> <signal_num>`: Send a specific signal to a named service.
- `-v`, `--version`: Show version.
- `-h`, `--help`: Show help.

Arguments after `start` and `restart` are passed unchanged to the background
application. For example, `./app restart --dev` starts `./app --dev`.
Application-owned flags such as `--config env.1.yaml` are also left untouched.

Applications can expose a local command in the running process with
`AddRemoteCommand`. CLI arguments are transported over a user-only Unix socket
instead of being evaluated by a shell.

Development or compatibility commands that must remain callable without being
listed in user-facing help can use `AddHiddenRemoteCommand`.

## v1.5.13 依赖基线

本版本对齐 Log v1.5.17 与发布时模块索引中的上游已发布依赖；公开 API 保持兼容。

## AES-GCM 配置密文兼容

正式依赖选用 crypto v1.5.9，共享 `Symmetric.Decrypt/DecryptBytes` 先认证 `[12 字节 IV][密文][16 字节 tag]`，再使用配置 IV 兼容旧格式；现有 API 不变。新数据推荐 `EncryptBytesAutoIV`，先升级接收端再启用新写入。此能力针对 AES-GCM，不改变 CBC/SM4 格式。

## Bounded shutdown (unreleased)

`Register` stop timeouts now bound the entire stop operation, including hooks and
retries, even when `Stop` ignores its context. A non-positive stop timeout uses
`DefaultStopTimeout` (5 seconds). Configure it before `Start` if needed.
`ShutdownTimeout` (15 seconds) bounds all priority tiers together.

Normal shutdown preserves reverse dependency order. A failed service is retried
within its original deadline; after that deadline, remaining dependencies receive
best-effort cleanup. A timed-out goroutine cannot be killed safely by Go and must
not be treated as successfully stopped. The foreground process exits with status
1 on incomplete shutdown or a second SIGINT/SIGTERM, and never reports shutdown
complete for those paths. This manager does not terminate arbitrary child process
groups: applications must own their children and react to parent cancellation.

### 内置日志命令

所有使用 starter 的应用默认提供 `log` 与 `logv` 命令，委托 log.RunCommand，在服务启动前分流。`app log` 默认读取当前配置日志并执行 -tail 100 -f；支持 -head N/-tail N/-cat、-f、-json/-raw 和文件参数。logv 无文件名格式化 stdin，文件参数自动关联 .dict；压缩 stdin 用 -dict。应用 AddCommand 可覆盖默认命令。

### 实时压缩日志（v1.5.16）

正式依赖已按顺序发布，go.mod 不保留本地开发 replace。文件日志默认采用 `.gz` + 同文件级 `.dict`，两份文件须一起管理；通过 logv 或内置 log/logv 读取，NoCompress=true 可保留明文。


### Release v1.5.16 — 2026-11-12

Released GitHub dependencies; no local development replacements. Full tests and benchmarks validated with GOWORK=off.
