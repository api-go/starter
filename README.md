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
	"time"

	"apigo.cc/go/starter"
)

func main() {
	starter.SetInfo("myapp", "1.0.0")

	starter.OnStart(func(ctx context.Context) {
		fmt.Println("Service starting...")
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Service stopping...")
				return
			case <-time.After(1 * time.Second):
				fmt.Println("Tick")
			}
		}
	})

	starter.OnStop(func() {
		fmt.Println("Service stopped.")
	})

	starter.Run()
}
```

## Commands

- `start`: Start the service in background.
- `stop`: Stop the service.
- `restart`: Restart the service.
- `status`: Show service status.
- `-v`, `--version`: Show version.
- `-h`, `--help`: Show help.

## Options

- `-pid`: PID file path (default `.pid`).
- `-log`: Log file path for background mode.
