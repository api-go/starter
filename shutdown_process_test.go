package starter_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"apigo.cc/go/log"
	"apigo.cc/go/starter"
)

type blockedShutdown struct{ marker string }

func (*blockedShutdown) Start(context.Context, *log.Logger) error { return nil }
func (*blockedShutdown) Status() (string, error)                  { return "ready", nil }
func (s *blockedShutdown) Stop(context.Context) error {
	mode := os.Getenv("STARTER_SHUTDOWN_TEST")
	if mode == "healthy" || mode == "after" {
		return nil
	}
	if mode == "errors" {
		f, err := os.OpenFile(s.marker+".stopping", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		_, _ = f.WriteString("attempt\n")
		_ = f.Close()
		return context.Canceled
	}
	_ = os.WriteFile(s.marker+".stopping", nil, 0600)
	select {}
}

type cleanupService struct{ marker string }

func (*cleanupService) Start(context.Context, *log.Logger) error { return nil }
func (*cleanupService) Status() (string, error)                  { return "ready", nil }
func (s *cleanupService) Stop(context.Context) error {
	return os.WriteFile(s.marker+".cleaned", nil, 0600)
}

// Runs only as a subprocess; no business service or media is opened.
func TestShutdownProcessHelper(t *testing.T) {
	mode := os.Getenv("STARTER_SHUTDOWN_TEST")
	if mode == "" {
		return
	}
	marker := os.Getenv("STARTER_TEST_MARKER")
	os.Args = os.Args[:1]
	starter.SetAppInfo(filepath.Base(marker), "test")
	starter.DefaultStopTimeout = 80 * time.Millisecond
	starter.ShutdownTimeout = time.Second
	timeout := time.Duration(0)
	if mode == "repeat" {
		timeout = 5 * time.Second
		starter.ShutdownTimeout = 5 * time.Second
	}
	if mode == "overall" {
		timeout = time.Second
		starter.ShutdownTimeout = 80 * time.Millisecond
	}
	if mode == "errors" {
		starter.DefaultStopTimeout = 250 * time.Millisecond
	}
	managed := starter.Register("blocked", &blockedShutdown{marker}, 1, 0, timeout)
	hook := func() { _ = os.WriteFile(marker+".stopping", nil, 0600); select {} }
	if mode == "before" {
		managed.OnStopping(hook)
	}
	if mode == "after" {
		managed.OnStopped(hook)
	}
	starter.Register("dependency", &cleanupService{marker}, 0, 0, 0)
	if err := starter.Start(); err != nil {
		panic(err)
	}
	_ = os.WriteFile(marker, nil, 0600)
	starter.Wait()
}

func TestForegroundShutdownIsBounded(t *testing.T) {
	for _, mode := range []string{"default", "overall", "repeat", "before", "after", "errors", "healthy"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			marker := filepath.Join(t.TempDir(), "ready")
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestShutdownProcessHelper$")
			cmd.Env = append(os.Environ(), "STARTER_SHUTDOWN_TEST="+mode, "STARTER_TEST_MARKER="+marker)
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = cmd.Process.Kill() }()
			waitFile := func(path string) {
				t.Helper()
				for {
					if _, err := os.Stat(path); err == nil {
						return
					}
					select {
					case <-ctx.Done():
						t.Fatal("helper did not reach " + path)
					case <-time.After(5 * time.Millisecond):
					}
				}
			}
			waitFile(marker)
			_ = cmd.Process.Signal(syscall.SIGINT)
			if mode != "healthy" {
				waitFile(marker + ".stopping")
			}
			began := time.Now()
			if mode == "repeat" {
				_ = cmd.Process.Signal(syscall.SIGINT)
			}
			err := cmd.Wait()
			if ctx.Err() != nil {
				t.Fatal("shutdown hung")
			}
			if mode == "healthy" {
				if err != nil {
					t.Fatalf("healthy shutdown failed: %v %s", err, output.String())
				}
			} else {
				exit, ok := err.(*exec.ExitError)
				if !ok || exit.ExitCode() != 1 {
					t.Fatalf("incomplete shutdown status: %v %s", err, output.String())
				}
				if strings.Contains(output.String(), "shutdown complete") {
					t.Fatal("incomplete shutdown reported complete")
				}
			}
			if mode != "overall" && mode != "repeat" {
				if _, err := os.Stat(marker + ".cleaned"); err != nil {
					t.Fatalf("lower priority was not cleaned: %s", output.String())
				}
			}
			if mode == "errors" {
				data, _ := os.ReadFile(marker + ".stopping")
				if bytes.Count(data, []byte("attempt")) < 2 {
					t.Fatal("transient errors were not retried")
				}
			}
			if time.Since(began) > 2*time.Second {
				t.Fatal("shutdown deadline or repeated signal ignored")
			}
		})
	}
}
