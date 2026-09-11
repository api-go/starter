# @go/starter

Service starter for @go applications, supporting background mode, PID management, and signal handling.

## Installation

```bash
go get apigo.cc/go/starter
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"apigo.cc/go/log"
	"apigo.cc/go/starter"
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

Applications can expose a local command in the running process with
`AddRemoteCommand`. CLI arguments are transported over a user-only Unix socket
instead of being evaluated by a shell.
