### starter > starter.go
```go



var (
	// Default configuration
	appName    = filepath.Base(os.Args[0])
	appVersion = "1.0.1"

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

// Service defines the lifecycle of a component managed by the starter.
type Service interface {
	// Start starts the service. It should block until the service is ready.
	Start(ctx context.Context, logger *log.Logger) error
	// Stop stops the service. It should block until the service is cleaned up.
	Stop(ctx context.Context) error
	// Health returns the health status of the service.
	Health() error
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

func init()


// Register adds a service to be managed by the starter.
func Register(name string, svc Service, priority int, startTimeout, stopTimeout time.Duration)


// SetAppInfo sets the application name and version.
func SetAppInfo(name, version string)


// AddCommand adds a custom command.
func AddCommand(name, desc string, fn func())


// Run parses arguments and executes the service.
func Run()


func showHelp()


func runForeground()


func startServices(ctx context.Context) error


func stopServices()


func reloadServices()


func handleUserSignal(svcName *string, sig os.Signal) bool


func serveIPC(l net.Listener)


func getInternalStatus() string


func startCmd()


func stopCmd()


func restartCmd()


func statusCmd()


func killCmd()


func callIPC(pid int, cmd string) (string, error)


func getIPCToken(pid int) string


func getPidPath() string


func getSockPath() string


func savePid(p int)


func loadPid() int


func removePid()


func isProcessRunning(p int) bool

```

