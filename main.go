package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"pulse-song-service/config"
	"pulse-song-service/poster"
	svc "pulse-song-service/service"
	"pulse-song-service/watcher"
)

func main() {
	// Handle service commands: install, uninstall, start, stop.
	if len(os.Args) > 1 {
		if err := svc.HandleCommand(os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// No arguments — run the app.
	// In normal mode (terminal / GoLand), stop on Ctrl+C.
	// As a Windows service, kardianos handles the lifecycle.
	stop := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		close(stop)
	}()

	svc.Run(run, stop)
}

// run is the main app logic. It runs until the stop channel is closed.
func run(stop <-chan struct{}) {
	appDir := getAppDir()

	// Set up logging: recreate the log file on each run.
	// Write to both file and stdout in interactive mode, file only as a service.
	logFile, err := os.Create(filepath.Join(appDir, "pulse-song-service.log"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file: %v\n", err)
		return
	}
	defer logFile.Close()
	if svc.IsInteractive() {
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	} else {
		log.SetOutput(logFile)
	}
	log.SetFlags(log.Ldate | log.Ltime)

	log.Println("starting pulse-song-service")
	log.Printf("app directory: %s", appDir)

	cfg, err := config.Load(appDir)
	if err != nil {
		log.Printf("ERROR: configuration error: %v", err)
		return
	}

	log.Printf("watching file: %s", cfg.File)
	log.Printf("posting to %d endpoint(s)", len(cfg.Endpoints))
	for i, ep := range cfg.Endpoints {
		log.Printf("  endpoint #%d: %s (postKey=%s)", i+1, ep.URL, ep.PostKey)
	}

	// Watch the file for changes and post content to all endpoints.
	err = watcher.Watch(cfg.File, func(content string) {
		log.Printf("file changed to: %s", content)
		poster.PostToAll(cfg.Endpoints, content)
	}, stop)
	if err != nil {
		log.Printf("ERROR: watcher failed: %v", err)
		return
	}

	log.Println("shutting down")
}

// getAppDir returns the directory where config.json and log files live.
func getAppDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot determine working directory: %v\n", err)
		os.Exit(1)
	}
	return cwd
}
