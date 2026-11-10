package starter

import (
	"github.com/api-go/log"
	"context"
	"os"
)

// Service defines the lifecycle of a component managed by the starter.
type Service interface {
	// Start starts the service. It should block until the service is ready.
	Start(ctx context.Context, logger *log.Logger) error
	// Stop stops the service. It should block until the service is cleaned up.
	Stop(ctx context.Context) error
	// Status returns the current status and health of the service.
	Status() (string, error)
}

// Reloader defines an optional interface for services that support configuration reloading.
type Reloader interface {
	Reload() error
}

// UserSignalHandler defines an optional interface for services that handle custom user signals.
type UserSignalHandler interface {
	// HandleUserSignal handles a custom signal. Return true if the signal was handled.
	HandleUserSignal(sig os.Signal) bool
}
