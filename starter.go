package starter

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"apigo.cc/go/file"
	"apigo.cc/go/log"
)

var (
	// Default configuration
	appName     = filepath.Base(os.Args[0])
	appVersion  = "1.0.0"
	pidPath     = ".pid"
	logPath     = ""
	stopTimeout = 5 * time.Second

	// Internal state
	commands = make(map[string]*command)
	onStart  func(ctx context.Context)
	onStop   func()
	onReload func()

	// Flags
	flagSet = flag.NewFlagSet(appName, flag.ContinueOnError)
)

type command struct {
	name string
	desc string
	fn   func()
}

func init() {
	flagSet.StringVar(&pidPath, "pid", ".pid", "PID file path")
	flagSet.StringVar(&logPath, "log", "", "Log file path (for background mode)")

	AddCmd("start", "Start the service in background", startCmd)
	AddCmd("stop", "Stop the service", stopCmd)
	AddCmd("restart", "Restart the service", restartCmd)
	AddCmd("status", "Show service status", statusCmd)
}

// SetInfo sets the application name and version.
func SetInfo(name, version string) {
	appName = name
	appVersion = version
}

// SetPidFile sets the default PID file path.
func SetPidFile(path string) {
	pidPath = path
}

// SetLogFile sets the default log file path.
func SetLogFile(path string) {
	logPath = path
}

// AddCmd adds a custom command.
func AddCmd(name, desc string, fn func()) {
	commands[name] = &command{name: name, desc: desc, fn: fn}
}

// OnStart sets the function to be called when the service starts.
// The context will be canceled when a stop signal is received.
func OnStart(fn func(ctx context.Context)) {
	onStart = fn
}

// OnStop sets the function to be called when the service is stopping.
func OnStop(fn func()) {
	onStop = fn
}

// OnReload sets the function to be called when SIGHUP is received.
func OnReload(fn func()) {
	onReload = fn
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
	fmt.Printf("%s (%s)\n\nUsage:\n  %s [command] [options]\n\nCommands:\n",
		appName, appVersion, filepath.Base(os.Args[0]))

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if onStart != nil {
		go onStart(ctx)
	}

	for sig := range sigChan {
		if sig == syscall.SIGHUP {
			if onReload != nil {
				log.DefaultLogger.Info("Received SIGHUP. Reloading...")
				onReload()
			}
			continue
		}

		log.DefaultLogger.Info(fmt.Sprintf("Received signal: %v. Shutting down...", sig))
		break
	}

	cancel() // Trigger context cancellation for onStart

	if onStop != nil {
		onStop()
	}
	log.DefaultLogger.Info("Shutdown complete.")
}

func startCmd() {
	pid := loadPid()
	if pid > 0 && isProcessRunning(pid) {
		log.DefaultLogger.Info(fmt.Sprintf("%s is already running (PID %d)", appName, pid))
		return
	}

	// Build arguments for background process
	args := []string{}
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] != "start" {
			args = append(args, os.Args[i])
		}
	}

	cmd := exec.Command(os.Args[0], args...)

	if logPath != "" {
		file.EnsureParentDir(logPath)
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			cmd.Stdout = f
			cmd.Stderr = f
		}
	}

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

	deadline := time.Now().Add(stopTimeout)
	for time.Now().Before(deadline) {
		if !isProcessRunning(pid) {
			log.DefaultLogger.Info("Stopped OK")
			removePid()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	log.DefaultLogger.Info("Stop timeout, killing...")
	_ = process.Kill()
	removePid()
}

func restartCmd() {
	stopCmd()
	time.Sleep(500 * time.Millisecond)
	startCmd()
}

func statusCmd() {
	pid := loadPid()
	if pid > 0 && isProcessRunning(pid) {
		log.DefaultLogger.Info(fmt.Sprintf("%s is running (PID %d)", appName, pid))
	} else {
		log.DefaultLogger.Info(fmt.Sprintf("%s is not running", appName))
	}
}

func savePid(p int) {
	file.EnsureParentDir(pidPath)
	_ = os.WriteFile(pidPath, []byte(strconv.Itoa(p)), 0644)
}

func loadPid() int {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0
	}
	p, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return p
}

func removePid() {
	_ = os.Remove(pidPath)
}

func isProcessRunning(p int) bool {
	process, err := os.FindProcess(p)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
