package starter_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/api-go/log"
	"github.com/api-go/starter"
)

type mockService struct {
	name         string
	startDelay   time.Duration
	stopDelay    time.Duration
	startErr     error
	stopErr      error
	healthErr    error
	startedAt    time.Time
	stoppedAt    time.Time
	reloadedAt   time.Time
	signalCaught os.Signal
	startOrder   int
	stopOrder    int
	mu           sync.Mutex
}

var (
	orderMu     sync.Mutex
	globalOrder int
)

func resetOrder() {
	orderMu.Lock()
	defer orderMu.Unlock()
	globalOrder = 0
}

func getOrder() int {
	orderMu.Lock()
	defer orderMu.Unlock()
	globalOrder++
	return globalOrder
}

func (s *mockService) Start(ctx context.Context, logger *log.Logger) error {
	select {
	case <-time.After(s.startDelay):
		if s.startErr != nil {
			return s.startErr
		}
		s.mu.Lock()
		s.startedAt = time.Now()
		s.startOrder = getOrder()
		s.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *mockService) Stop(ctx context.Context) error {
	select {
	case <-time.After(s.stopDelay):
		if s.stopErr != nil {
			return s.stopErr
		}
		s.mu.Lock()
		s.stoppedAt = time.Now()
		s.stopOrder = getOrder()
		s.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *mockService) Status() (string, error) {
	if s.healthErr != nil {
		return "unhealthy", s.healthErr
	}
	return "ok", nil
}

func (s *mockService) Reload() error {
	s.mu.Lock()
	s.reloadedAt = time.Now()
	s.mu.Unlock()
	return nil
}

func (s *mockService) HandleUserSignal(sig os.Signal) bool {
	s.mu.Lock()
	s.signalCaught = sig
	s.mu.Unlock()
	return true
}

func TestServiceRegister(t *testing.T) {
	resetOrder()
	s1 := &mockService{name: "s1"}
	starter.Register("s1", s1, 1, 0, 0)
}

func TestStarterBasics(t *testing.T) {
	starter.SetAppInfo("TestApp", "1.0.1")
	starter.AddCommand("custom", "desc", func() {
		// custom cmd
	})
}
