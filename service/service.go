package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kardianos/service"
)

const serviceName = "pulse-song-service"
const serviceDisplayName = "Pulse Song Service"
const serviceDescription = "Watches a file for changes and posts content to configured endpoints"

// RunFunc is the function signature for the main app logic.
// The stop channel will be closed when the service needs to shut down.
type RunFunc func(stop <-chan struct{})

// program implements the kardianos/service Interface.
type program struct {
	runFunc RunFunc
	stop    chan struct{}
	done    chan struct{}
}

func (p *program) Start(s service.Service) error {
	// When running as a Windows service, the working directory is C:\Windows\System32.
	// Change to the executable's directory so config.json and logs are found.
	if !service.Interactive() {
		exe, err := os.Executable()
		if err == nil {
			os.Chdir(filepath.Dir(exe))
		}
	}

	p.stop = make(chan struct{})
	p.done = make(chan struct{})
	go func() {
		p.runFunc(p.stop)
		close(p.done)
	}()
	return nil
}

func (p *program) Stop(s service.Service) error {
	close(p.stop)
	<-p.done // Wait for app logic to finish.
	return nil
}

// newServiceConfig returns the service configuration.
// On Windows, it sets the working directory to the executable's directory
// so that config.json and log files are found next to the .exe.
func newServiceConfig() *service.Config {
	cfg := &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
	}

	// On Windows, services start with C:\Windows\System32 as working directory.
	// Setting the Executable option with the full path and using the Arguments/Option
	// is not enough — we need to tell kardianos to set the working directory.
	if runtime.GOOS == "windows" {
		cfg.Option = service.KeyValue{
			"StartType": "automatic",
		}
	}

	return cfg
}

// HandleCommand processes service management commands: install, uninstall, start, stop.
func HandleCommand(cmd string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("service commands are only supported on Windows")
	}

	svcConfig := newServiceConfig()
	prg := &program{}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("cannot initialize service: %w", err)
	}

	switch cmd {
	case "install":
		err = s.Install()
		if err != nil {
			return fmt.Errorf("cannot install service: %w", err)
		}
		fmt.Printf("Service %q installed successfully.\n", serviceName)
		return nil
	case "uninstall":
		err = s.Uninstall()
		if err != nil {
			return fmt.Errorf("cannot uninstall service: %w", err)
		}
		fmt.Printf("Service %q uninstalled successfully.\n", serviceName)
		return nil
	case "start":
		err = s.Start()
		if err != nil {
			return fmt.Errorf("cannot start service: %w", err)
		}
		fmt.Printf("Service %q started.\n", serviceName)
		return nil
	case "stop":
		err = s.Stop()
		if err != nil {
			return fmt.Errorf("cannot stop service: %w", err)
		}
		fmt.Printf("Service %q stopped.\n", serviceName)
		return nil
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// IsInteractive returns true when running from a terminal (not as a Windows service).
func IsInteractive() bool {
	return service.Interactive()
}

// Run starts the app. If running as a Windows service, it uses the service handler.
// Otherwise it calls runFunc directly (development / manual execution).
func Run(runFunc RunFunc, stop <-chan struct{}) {
	svcConfig := newServiceConfig()
	prg := &program{runFunc: runFunc}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		// Can't create service object — just run directly.
		runFunc(stop)
		return
	}

	if service.Interactive() {
		// Running from terminal (development mode) — run directly.
		runFunc(stop)
	} else {
		// Running as a Windows service — let kardianos manage the lifecycle.
		// It will call prg.Start() and prg.Stop() as needed.
		err = s.Run()
		if err != nil {
			fmt.Printf("Service run error: %v\n", err)
		}
	}
}
