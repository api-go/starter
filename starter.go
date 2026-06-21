package starter

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	"apigo.cc/go/crypto"
	"apigo.cc/go/id"
	"apigo.cc/go/log"
)

var (
	// Default configuration
	appName    = filepath.Base(os.Args[0])
	appVersion = "1.0.1"
	appUsage   = ""

	// Internal state
	commands = make(map[string]*command)

	// New Service registry
	services          = make(map[int][]*managedService)
	startedPriorities []int

	// Flags
	flagSet = flag.NewFlagSet(appName, flag.ContinueOnError)

	// IPC Security
	ipcSecret = "apigo-starter-secret-2026"

	// Starter Logger
	starterLogger *log.Logger

	// Signal handling
	sigs      chan os.Signal
	cancelCtx context.CancelFunc
)

type managedService struct {
	Name         string
	svc          Service
	priority     int
	startTimeout time.Duration
	stopTimeout  time.Duration
	onStarting   []func()
	onStarted    []func()
	onStopping   []func()
	onStopped    []func()
}

func (ms *managedService) OnStarting(fn func()) *managedService {
	ms.onStarting = append(ms.onStarting, fn)
	return ms
}

func (ms *managedService) OnStarted(fn func()) *managedService {
	ms.onStarted = append(ms.onStarted, fn)
	return ms
}

func (ms *managedService) OnStopping(fn func()) *managedService {
	ms.onStopping = append(ms.onStopping, fn)
	return ms
}

func (ms *managedService) OnStopped(fn func()) *managedService {
	ms.onStopped = append(ms.onStopped, fn)
	return ms
}

type command struct {
	name string
	desc string
	fn   func()
}

func init() {
	AddCommand("start", "Start the service in background", startCmd)
	AddCommand("stop", "Stop the service", stopCmd)
	AddCommand("restart", "Restart the service", restartCmd)
	AddCommand("status", "Show service status", statusCmd)
	AddCommand("kill", "Send signal to a specific service: kill <svc_name> <signal_num>", killCmd)
}

// getStarterLogger returns the singleton logger for the starter context.
func getStarterLogger() *log.Logger {
	if starterLogger == nil {
		starterLogger = log.DefaultLogger.New(id.Get8Bytes4KPerSecond())
	}
	return starterLogger
}

// Register adds a service to be managed by the starter.
func Register(name string, svc Service, priority int, startTimeout, stopTimeout time.Duration) *managedService {
	ms := &managedService{
		Name:         name,
		svc:          svc,
		priority:     priority,
		startTimeout: startTimeout,
		stopTimeout:  stopTimeout,
	}
	services[priority] = append(services[priority], ms)
	return ms
}

// SetAppInfo sets the application name and version.
func SetAppInfo(name, version string) {
	appName = name
	appVersion = version
}

// SetUsage sets custom usage text to be displayed in help.
func SetUsage(text string) {
	appUsage = text
}

// AddCommand adds a custom command.
func AddCommand(name, desc string, fn func()) {
	commands[name] = &command{name: name, desc: desc, fn: fn}
}

// TODO 使用 Start / Wait 代替 Run，方便在启动后做初始化操作，或者支持注册某服务的 OnStarting / OnStarted / OnStopping / OnStopped  事件

// Start parses arguments and starts the services.
func Start() error {
	flagSet.Usage = showHelp
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if cmd, ok := commands[arg]; ok {
			// Subcommand detected, parse flags after the command
			_ = flagSet.Parse(os.Args[2:])
			cmd.fn()
			os.Exit(0)
		}

		// Check for help/version
		switch arg {
		case "-h", "--help", "help":
			showHelp()
			os.Exit(0)
		case "-v", "--version", "version":
			fmt.Printf("%s version %s\n", appName, appVersion)
			os.Exit(0)
		}
	}

	// No starter command, treat all as app flags
	_ = flagSet.Parse(os.Args[1:])

	// Setup signal handling for graceful shutdown
	sigs = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)

	var ctx context.Context
	ctx, cancelCtx = context.WithCancel(context.Background())

	if err := startServices(ctx); err != nil {
		getStarterLogger().Error(fmt.Sprintf("[starter] start services failed: %v", err))
		return err
	}
	return nil
}

// Wait blocks until a termination signal is received.
func Wait() {
	if sigs == nil {
		return
	}

	for {
		sig := <-sigs
		if sig == syscall.SIGHUP {
			reloadServices()
			continue
		}

		// Handle user custom signals
		if sig == syscall.SIGUSR1 || sig == syscall.SIGUSR2 {
			if !handleUserSignal(nil, sig) {
				getStarterLogger().Info(fmt.Sprintf("[starter] received signal %v, but no service handled it.", sig))
			}
			continue
		}

		getStarterLogger().Info(fmt.Sprintf("[starter] received signal: %v, shutting down...", sig))
		break
	}

	if cancelCtx != nil {
		cancelCtx() // Trigger context cancellation
	}
	stopServices()
	getStarterLogger().Info("[starter] shutdown complete")
}

func startServices(ctx context.Context) error {
	var priorities []int
	for p := range services {
		priorities = append(priorities, p)
	}
	sort.Ints(priorities)

	for _, p := range priorities {
		svcs := services[p]
		var wg sync.WaitGroup
		errChan := make(chan error, len(svcs))

		for _, ms := range svcs {
			wg.Add(1)
			go func(ms *managedService) {
				defer wg.Done()
				sctx := ctx
				if ms.startTimeout > 0 {
					var cancel context.CancelFunc
					sctx, cancel = context.WithTimeout(ctx, ms.startTimeout)
					defer cancel()
				}

				// Each service gets its own unique 10-byte trace ID for its internal logs
				serviceTraceId := id.Get10Bytes14MPerSecond()
				serviceLogger := log.DefaultLogger.New(serviceTraceId)

				// Log using starter's logger (8-byte ID) but include service's trace ID in extra fields
				getStarterLogger().Info(fmt.Sprintf("service [%s] starting", ms.Name), "trace", serviceTraceId)
				for _, fn := range ms.onStarting {
					fn()
				}
				if err := ms.svc.Start(sctx, serviceLogger); err != nil {
					errChan <- fmt.Errorf("service [%s] start error: %w", ms.Name, err)
				} else {
					getStarterLogger().Info(fmt.Sprintf("service [%s] started", ms.Name), "trace", serviceTraceId)
					for _, fn := range ms.onStarted {
						fn()
					}
				}
			}(ms)
		}
		wg.Wait()
		close(errChan)
		for err := range errChan {
			if err != nil {
				return err
			}
		}
		startedPriorities = append(startedPriorities, p)
	}
	return nil
}

func stopServices() {
	sort.Slice(startedPriorities, func(i, j int) bool {
		return startedPriorities[i] > startedPriorities[j]
	})

	for _, p := range startedPriorities {
		svcs := services[p]
		var wg sync.WaitGroup
		for _, ms := range svcs {
			wg.Add(1)
			go func(ms *managedService) {
				defer wg.Done()
				sctx := context.Background()
				if ms.stopTimeout > 0 {
					var cancel context.CancelFunc
					sctx, cancel = context.WithTimeout(sctx, ms.stopTimeout)
					defer cancel()
				}
				getStarterLogger().Info(fmt.Sprintf("service [%s] stopping", ms.Name))
				for _, fn := range ms.onStopping {
					fn()
				}
				if err := ms.svc.Stop(sctx); err != nil {
					getStarterLogger().Error(fmt.Sprintf("service [%s] stop error: %v", ms.Name, err))
				} else {
					getStarterLogger().Info(fmt.Sprintf("service [%s] stopped", ms.Name))
					for _, fn := range ms.onStopped {
						fn()
					}
				}
			}(ms)
		}
		wg.Wait()
	}
	startedPriorities = nil
}

func reloadServices() {
	getStarterLogger().Info("[starter] reloading all services...")
	for _, p := range startedPriorities {
		for _, ms := range services[p] {
			if r, ok := ms.svc.(Reloader); ok {
				if err := r.Reload(); err != nil {
					getStarterLogger().Error(fmt.Sprintf("service [%s] reload error: %v", ms.Name, err))
				}
			}
		}
	}
}

func handleUserSignal(svcName *string, sig os.Signal) bool {
	handled := false
	for _, p := range startedPriorities {
		for _, ms := range services[p] {
			if svcName != nil && *svcName != ms.Name {
				continue
			}
			if h, ok := ms.svc.(UserSignalHandler); ok {
				if h.HandleUserSignal(sig) {
					handled = true
				}
			}
		}
	}
	return handled
}

func showHelp() {
	fmt.Printf("%s (%s)\n\n", appName, appVersion)
	if appUsage != "" {
		fmt.Printf("%s\n\n", appUsage)
	}
	fmt.Printf("Usage:\n  %s [command] [options]\n\nCommands:\n", filepath.Base(os.Args[0]))

	var names []string
	for cmdName := range commands {
		names = append(names, cmdName)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Printf("  %-10s %s\n", name, commands[name].desc)
	}
	fmt.Println()
}

func startCmd() {
	pid := getPid()
	if pid != 0 && isProcessRunning(pid) {
		getStarterLogger().Info(fmt.Sprintf("[starter] %s is already running (PID %d)", appName, pid))
		return
	}

	cmd := exec.Command(os.Args[0])
	cmd.Args = append([]string{os.Args[0]}, flagSet.Args()...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		getStarterLogger().Error(fmt.Sprintf("[starter] failed to start %s: %v", appName, err))
		return
	}

	_ = os.WriteFile(getPidPath(), []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)
	getStarterLogger().Info(fmt.Sprintf("[starter] %s started (PID %d)", appName, cmd.Process.Pid))
}

func stopCmd() {
	pid := getPid()
	if pid == 0 || !isProcessRunning(pid) {
		getStarterLogger().Info(fmt.Sprintf("[starter] %s is not running", appName))
		return
	}

	getStarterLogger().Info(fmt.Sprintf("[starter] stopping %s (PID %d)...", appName, pid))
	_ = syscall.Kill(pid, syscall.SIGTERM)

	for i := 0; i < 30; i++ {
		if !isProcessRunning(pid) {
			removePid()
			getStarterLogger().Info(fmt.Sprintf("[starter] %s stopped", appName))
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	getStarterLogger().Warning(fmt.Sprintf("[starter] %s failed to stop gracefully, killing...", appName))
	_ = syscall.Kill(pid, syscall.SIGKILL)
	removePid()
}

func restartCmd() {
	stopCmd()
	startCmd()
}

func statusCmd() {
	pid := getPid()
	if pid == 0 || !isProcessRunning(pid) {
		fmt.Printf("%s is NOT running\n", appName)
		return
	}

	fmt.Printf("%s is running (PID %d)\n\n", appName, pid)
	res, err := callIPC(pid, "status")
	if err == nil {
		fmt.Println("Services Status:")
		fmt.Println(res)
	}
}

func killCmd() {
	if flagSet.NArg() < 2 {
		fmt.Println("Usage: kill <service_name> <signal_num>")
		return
	}
	svcName := flagSet.Arg(0)
	sigNum := flagSet.Arg(1)
	pid := getPid()
	if pid == 0 || !isProcessRunning(pid) {
		fmt.Println("Application is not running")
		return
	}

	res, err := callIPC(pid, fmt.Sprintf("kill %s %s", svcName, sigNum))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println(res)
	}
}

func callIPC(pid int, cmd string) (string, error) {
	sockPath := getSockPath()
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	token := generateToken(pid)
	_, _ = conn.Write([]byte(fmt.Sprintf("%s %s", token, cmd)))

	buf, err := io.ReadAll(conn)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func generateToken(pid int) string {
	return crypto.SHA256ToHex([]byte(fmt.Sprintf("%s:%d", ipcSecret, pid)))
}

func getPidPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s.pid", appName, appVersion))
}

func getSockPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s.sock", appName, appVersion))
}

func getPid() int {
	data, err := os.ReadFile(getPidPath())
	if err != nil {
		return 0
	}
	var pid int
	fmt.Sscanf(string(data), "%d", &pid)
	return pid
}

func removePid() {
	_ = os.Remove(getPidPath())
}

func isProcessRunning(p int) bool {
	process, err := os.FindProcess(p)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
