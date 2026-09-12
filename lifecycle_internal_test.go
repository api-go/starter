package starter

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"

	"apigo.cc/go/log"
)

func TestServiceCommandArgsPreservesApplicationFlags(t *testing.T) {
	want := []string{"--dev", "--config=env.1.yaml", "--profile=test"}
	for _, command := range []string{"start", "restart"} {
		got := serviceCommandArgs([]string{"./master", command, "--dev", "--config=env.1.yaml", "--profile=test"})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s args = %#v, want %#v", command, got, want)
		}
	}
}

func TestDispatchIPCCommandPassesArguments(t *testing.T) {
	oldCommands := ipcCommands
	defer func() { ipcCommands = oldCommands }()
	ipcCommands = map[string]func([]string) (string, error){
		"run": func(args []string) (string, error) { return args[0], nil },
	}
	got, err := dispatchIPCCommand("run", []string{"--dev remains application input"})
	if err != nil || got != "--dev remains application input" {
		t.Fatalf("dispatch result = %q, %v", got, err)
	}
}

type lifecycleTestService struct {
	stop func(context.Context) error
}

func (*lifecycleTestService) Start(context.Context, *log.Logger) error { return nil }
func (s *lifecycleTestService) Stop(ctx context.Context) error         { return s.stop(ctx) }
func (*lifecycleTestService) Status() (string, error)                  { return "ok", nil }

func TestStopServicesDoesNotCrossFailedPriority(t *testing.T) {
	oldServices, oldPriorities := services, startedPriorities
	defer func() {
		services, startedPriorities = oldServices, oldPriorities
	}()

	release := make(chan struct{})
	upstreamRetried := make(chan struct{})
	downstreamStopped := make(chan struct{})
	var calls atomic.Int32
	services = map[int][]*managedService{
		2: {{Name: "upstream", svc: &lifecycleTestService{stop: func(context.Context) error {
			if calls.Add(1) == 1 {
				return errors.New("still active")
			}
			close(upstreamRetried)
			<-release
			return nil
		}}}},
		1: {{Name: "downstream", svc: &lifecycleTestService{stop: func(context.Context) error {
			close(downstreamStopped)
			return nil
		}}}},
	}
	startedPriorities = []int{1, 2}
	done := make(chan struct{})
	go func() {
		stopServices()
		close(done)
	}()
	<-upstreamRetried
	select {
	case <-downstreamStopped:
		t.Fatal("downstream stopped before upstream succeeded")
	default:
	}
	close(release)
	<-done
}
