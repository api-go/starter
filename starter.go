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
	"strings"
	"sync"
	"syscall"
	"time"

	"apigo.cc/go/cast"
	"apigo.cc/go/crypto"
	"apigo.cc/go/file"
	"apigo.cc/go/id"
	"apigo.cc/go/log"
	"apigo.cc/go/shell"
	"apigo.cc/go/timer"
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
)

type managedService struct {
	Name         string
	svc          Service
	priority     int
	startTimeout time.Duration
	stopTimeout  time.Duration
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

	// Auto-register log writer service with high priority
	Register("log-writer", log.WriterService, -100, 0, 0)
}

// Register adds a service to be managed by the starter.
func Register(name string, svc Service, priority int, startTimeout, stopTimeout time.Duration) {
	services[priority] = append(services[priority], &managedService{
		Name:         name,
		svc:          svc,
		priority:     priority,
		startTimeout: startTimeout,
		stopTimeout:  stopTimeout,
	})
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

// Run parses arguments and executes the service.
func Run() {
	flagSet.Usage = showHelp
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if cmd, ok := commands[arg]; ok {
			// Subcommand detected, parse flags after the command
			_ = flagSet.Parse(os.Args[2:])
			cmd.fn()
			return
		}

		// Check for help/version
		switch arg {
		case "-h", "--help", "help":
			showHelp()
			return
		case "-v", "--version", "version":
			fmt.Printf("%s version %s\n", appName, appVersion)
			return
		}
	}

	// No starter command, treat all as app flags
	_ = flagSet.Parse(os.Args[1:])
	runForeground()
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

	for _, cmdName := range names {
		fmt.Printf("  %-10s %s\n", cmdName, commands[cmdName].desc)
	}

	fmt.Println("\nOptions:")
	flagSet.PrintDefaults()
	fmt.Println("\nIf no command is provided, the service runs in the foreground.")
}

func runForeground() {
	pid := os.Getpid()
	savePid(pid)
	defer removePid()

	// Prepare IPC listener but don't serve yet to avoid race conditions during startup
	sockPath := getSockPath()
	_ = os.Remove(sockPath)
	l, err := net.Listen("unix", sockPath)
	if err == nil {
		defer func() {
			_ = l.Close()
			_ = os.Remove(sockPath)
		}()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 10)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)

	// Start registered services
	if err := startServices(ctx); err != nil {
		log.DefaultLogger.Error(fmt.Sprintf("Start services failed: %v", err))
		stopServices()
		log.DefaultLogger.Error("Service failed to start, exiting.")
		os.Exit(1)
		return
	}

	// Service started successfully, now expose IPC
	if l != nil {
		go serveIPC(l)
	}

	for sig := range sigChan {
		if sig == syscall.SIGHUP {
			log.DefaultLogger.Info("Received SIGHUP. Reloading...")
			reloadServices()
			continue
		}

		// Handle user custom signals
		if sig == syscall.SIGUSR1 || sig == syscall.SIGUSR2 {
			if !handleUserSignal(nil, sig) {
				log.DefaultLogger.Info(fmt.Sprintf("Received signal %v, but no service handled it.", sig))
			}
			continue
		}

		log.DefaultLogger.Info(fmt.Sprintf("Received signal: %v. Shutting down...", sig))
		break
	}

	cancel() // Trigger context cancellation
	stopServices()
	log.DefaultLogger.Info("Shutdown complete.")
}

func startServices(ctx context.Context) error {
	var priorities []int
	for p := range services {
		priorities = append(priorities, p)
	}
	sort.Ints(priorities)

	// Generate a shared logger with trace ID for all services startup
	logger := log.DefaultLogger.New(id.Get8Bytes4KPerSecond())

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
				if err := ms.svc.Start(sctx, logger); err != nil {
					errChan <- fmt.Errorf("service [%s] start error: %w", ms.Name, err)
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
				if err := ms.svc.Stop(sctx); err != nil {
					log.DefaultLogger.Error(fmt.Sprintf("service [%s] stop error: %v", ms.Name, err))
				}
			}(ms)
		}
		wg.Wait()
	}
	startedPriorities = nil
}

func reloadServices() {
	for _, p := range startedPriorities {
		for _, ms := range services[p] {
			if r, ok := ms.svc.(Reloader); ok {
				if err := r.Reload(); err != nil {
					log.DefaultLogger.Error(fmt.Sprintf("service [%s] reload error: %v", ms.Name, err))
				}
			}
		}
	}
}

func handleUserSignal(svcName *string, sig os.Signal) bool {
	handled := false
	for _, p := range startedPriorities {
		for _, ms := range services[p] {
			if svcName != nil && ms.Name != *svcName {
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

func serveIPC(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			data := make([]byte, 4096)
			n, err := c.Read(data)
			if err != nil || n == 0 {
				return
			}

			// Protocol: TOKEN COMMAND ARGS...
			parts := strings.Split(string(data[:n]), " ")
			if len(parts) < 2 {
				return
			}

			token := parts[0]
			if token != getIPCToken(os.Getpid()) {
				_, _ = c.Write([]byte("Error: Unauthorized"))
				return
			}

			cmd := parts[1]
			args := parts[2:]

			switch cmd {
			case "status":
				_, _ = c.Write([]byte(getInternalStatus()))
			case "kill":
				if len(args) < 2 {
					_, _ = c.Write([]byte("Error: Missing arguments for kill"))
					return
				}
				svcName := args[0]
				sigNum := cast.Int(args[1])
				if handleUserSignal(&svcName, syscall.Signal(sigNum)) {
					_, _ = c.Write([]byte(fmt.Sprintf("Signal %d sent to %s", sigNum, svcName)))
				} else {
					_, _ = c.Write([]byte(fmt.Sprintf("Error: Service %s not found or didn't handle signal", svcName)))
				}
			default:
				_, _ = c.Write([]byte("Error: Unknown command"))
			}
		}(conn)
	}
}

func getInternalStatus() string {
	var out string
	var priorities []int
	for p := range services {
		priorities = append(priorities, p)
	}
	sort.Ints(priorities)

	for _, p := range priorities {
		for _, ms := range services[p] {
			statusMsg, err := ms.svc.Status()
			indicator := shell.Green("OK")
			if err != nil {
				indicator = shell.Red(fmt.Sprintf("FAIL (%v)", err))
			}
			if statusMsg != "" {
				out += fmt.Sprintf("[%d] %-20s %s (%s)\n", p, ms.Name, indicator, statusMsg)
			} else {
				out += fmt.Sprintf("[%d] %-20s %s\n", p, ms.Name, indicator)
			}
		}
	}
	return out
}

func startCmd() {
	pid := loadPid()
	if pid > 0 && isProcessRunning(pid) {
		log.DefaultLogger.Info(fmt.Sprintf("%s is already running (PID %d)", appName, pid))
		return
	}

	args := []string{}
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] != "start" {
			args = append(args, os.Args[i])
		}
	}

	cmd := exec.Command(os.Args[0], args...)
	err := cmd.Start()
	if err != nil {
		log.DefaultLogger.Error(fmt.Sprintf("Failed to start %s: %v", appName, err))
		os.Exit(1)
	}

	log.DefaultLogger.Info(fmt.Sprintf("%s started (PID %d)", appName, cmd.Process.Pid))
}

func stopCmd() {
	pid := loadPid()
	if pid <= 0 || !isProcessRunning(pid) {
		log.DefaultLogger.Info(fmt.Sprintf("%s is not running", appName))
		return
	}

	process, _ := os.FindProcess(pid)
	log.DefaultLogger.Info(fmt.Sprintf("Stopping %s (PID %d)...", appName, pid))
	_ = process.Signal(syscall.SIGTERM)

	err := timer.Retry(func() error {
		if isProcessRunning(pid) {
			return fmt.Errorf("still running")
		}
		return nil
	}, timer.WithMaxRetries(25), timer.WithBackoff(200*time.Millisecond, 1.0))

	if err == nil {
		log.DefaultLogger.Info("Stopped OK")
		removePid()
		return
	}

	log.DefaultLogger.Info("Stop timeout, killing...")
	_ = process.Kill()
	removePid()
}

func restartCmd() {
	stopCmd()
	_ = timer.Retry(func() error { return nil }, timer.WithMaxRetries(1), timer.WithBackoff(500*time.Millisecond, 1.0))
	startCmd()
}

func statusCmd() {
	pid := loadPid()
	isRunning := pid > 0 && isProcessRunning(pid)
	if isRunning {
		fmt.Printf("%s is %s (PID %d)\n", appName, shell.Green("running"), pid)
		res, err := callIPC(pid, "status")
		if err == nil {
			fmt.Println("\nServices:")
			fmt.Print(res)
		}
	} else {
		fmt.Printf("%s is %s\n", appName, shell.Red("not running"))
	}
}

func killCmd() {
	if len(flagSet.Args()) < 2 {
		fmt.Println("Usage: kill <service_name> <signal_num>")
		return
	}
	pid := loadPid()
	if pid <= 0 || !isProcessRunning(pid) {
		fmt.Println("Error: process not running")
		return
	}
	svcName := flagSet.Arg(0)
	sigNum := flagSet.Arg(1)
	res, err := callIPC(pid, fmt.Sprintf("kill %s %s", svcName, sigNum))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println(res)
	}
}

func callIPC(pid int, cmd string) (string, error) {
	conn, err := net.Dial("unix", getSockPath())
	if err != nil {
		return "", err
	}
	defer conn.Close()

	token := getIPCToken(pid)
	_, _ = conn.Write([]byte(fmt.Sprintf("%s %s", token, cmd)))

	data, err := io.ReadAll(conn)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getIPCToken(pid int) string {
	// Use Sha256 for better security
	return crypto.Sha256ToHex([]byte(fmt.Sprintf("%s:%d", ipcSecret, pid)))
}

func getPidPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s.pid", appName, appVersion))
}

func getSockPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s.sock", appName, appVersion))
}

func savePid(p int) {
	_ = file.Write(getPidPath(), cast.To[string](p))
}

func loadPid() int {
	data, err := file.Read(getPidPath())
	if err != nil {
		return 0
	}
	return cast.To[int](data)
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
