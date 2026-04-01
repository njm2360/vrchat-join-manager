package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/joho/godotenv"
	"golang.org/x/sys/windows/svc"

	"jm-client/internal/core"
)

func run(ctx context.Context, appDir string) error {
	baseURL := os.Getenv("API_BASE")
	if baseURL == "" {
		return fmt.Errorf("API_BASE environment variable not set")
	}

	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		return fmt.Errorf("LOG_DIR environment variable not set")
	}
	abs, err := filepath.Abs(logDir)
	if err != nil {
		return fmt.Errorf("LOG_DIR: %w", err)
	}
	logDir = abs

	tzName := os.Getenv("LOG_TZ")
	if tzName == "" {
		return fmt.Errorf("LOG_TZ environment variable not set")
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return fmt.Errorf("LOG_TZ: %w", err)
	}

	apiClient := core.NewApiClient(baseURL)
	stateFile := filepath.Join(appDir, "state.json")
	watcher := core.NewLogWatcher(logDir, func(_ string) core.LineHandler {
		return core.NewVRChatLogParser(apiClient, loc).OnLine
	}, stateFile, false)

	log.Printf("Watching: %s", logDir)
	return watcher.Run(ctx)
}

func main() {
	install := flag.Bool("install", false, "Install as Windows service (requires admin)")
	remove := flag.Bool("remove", false, "Remove Windows service (requires admin)")
	flag.Parse()

	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	isService, err := svc.IsWindowsService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsWindowsService: %v\n", err)
		os.Exit(1)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = exeDir
	}
	appDir := filepath.Join(cacheDir, appName)
	logsDir := filepath.Join(appDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot create logs dir: %v\n", err)
	}
	logName := time.Now().Format("2006-01-02T15-04-05") + ".log"
	logFile, err := os.OpenFile(
		filepath.Join(logsDir, logName),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot open log file: %v\n", err)
	} else {
		defer logFile.Close()
		if isService {
			log.SetOutput(logFile)
		} else {
			log.SetOutput(io.MultiWriter(os.Stderr, logFile))
		}
	}
	log.SetFlags(log.Ldate | log.Ltime)

	localAppData, err := os.UserCacheDir()
	if err != nil {
		localAppData = exeDir
	}
	envPath := filepath.Join(localAppData, appName, ".env")
	if err := godotenv.Load(envPath); err != nil {
		_ = godotenv.Load(".env")
	}

	switch {
	case *install:
		if err := installService(exe); err != nil {
			log.Fatalf("install: %v", err)
		}
		return
	case *remove:
		if err := removeService(); err != nil {
			log.Fatalf("remove: %v", err)
		}
		return
	}

	if isService {
		if err := svc.Run(svcName, &windowsService{}); err != nil {
			log.Fatalf("svc.Run: %v", err)
		}
		return
	}

	// Console Mode
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Printf("Shutting down...")
	}()

	if err := run(ctx, appDir); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
	log.Printf("Stopped.")
}
