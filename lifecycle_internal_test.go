package starter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/api-go/log"
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

func TestHiddenRemoteCommandIsCallableButNotShownInHelp(t *testing.T) {
	oldCommands, oldIPCCommands := commands, ipcCommands
	oldAppName, oldAppVersion, oldArgs := appName, appVersion, os.Args
	defer func() {
		commands, ipcCommands = oldCommands, oldIPCCommands
		appName, appVersion, os.Args = oldAppName, oldAppVersion, oldArgs
	}()
	commands = map[string]*command{}
	ipcCommands = map[string]func([]string) (string, error){}
	appName, appVersion, os.Args = "test", "1.0.0", []string{"test"}
	AddCommand("visible", "Visible command", func() {})
	AddHiddenRemoteCommand("internal", func(args []string) (string, error) {
		return "ok", nil
	})

	if got, err := dispatchIPCCommand("internal", nil); err != nil || got != "ok" {
		t.Fatalf("hidden command dispatch = %q, %v", got, err)
	}

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = write
	showHelp()
	os.Stdout = oldStdout
	_ = write.Close()
	var output bytes.Buffer
	_, _ = io.Copy(&output, read)
	_ = read.Close()
	if bytes.Contains(output.Bytes(), []byte("internal")) {
		t.Fatalf("hidden command was shown in help:\n%s", output.String())
	}
	if !bytes.Contains(output.Bytes(), []byte("visible")) {
		t.Fatalf("visible command missing from help:\n%s", output.String())
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
