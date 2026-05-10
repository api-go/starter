package starter_test

import (
	"context"
	"testing"

	"apigo.cc/go/starter"
)

func TestStarterBasics(t *testing.T) {
	starter.SetInfo("TestApp", "1.0.0")

	// Since we are using a singleton, we just check if the methods can be called
	starter.OnStart(func(ctx context.Context) {
		// Mock start
	})

	starter.OnStop(func() {
		// Mock stop
	})

	starter.OnReload(func() {
		// Mock reload
	})

	// Run() cannot be easily tested here as it calls os.Exit and parses os.Args
}
