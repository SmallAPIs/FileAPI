package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SmallAPIs/FileAPI/internal/config"
	"github.com/SmallAPIs/FileAPI/internal/server"
	filetls "github.com/SmallAPIs/FileAPI/internal/tls"
)

// version is set at link time by release builds (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if len(args) == 0 {
		printUsage()
		return 0
	}

	switch args[0] {
	case "serve":
		return serveCommand(args[1:], logger)
	case "status":
		return statusCommand(args[1:])
	case "version", "-v", "--version":
		fmt.Println(version)
		return 0
	case "-h", "--help", "help":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printUsage()
		return 1
	}
}

func serveCommand(args []string, logger *slog.Logger) int {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "", "path to config.yaml (default: OS config dir)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", "error", err)
		return 1
	}

	if err := os.MkdirAll(cfg.ConfigDir, 0o700); err != nil {
		logger.Error("create config dir", "error", err)
		return 1
	}
	if err := cfg.Save(); err != nil {
		logger.Error("save config", "error", err)
		return 1
	}
	if err := filetls.Ensure(cfg.CertFile, cfg.KeyFile); err != nil {
		logger.Error("ensure tls cert", "error", err)
		return 1
	}

	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Error("create server", "error", err)
		return 1
	}

	printBanner(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		_ = srv.Shutdown(context.Background())
		return 0
	case err := <-errCh:
		if err != nil {
			logger.Error("server stopped", "error", err)
			return 1
		}
		return 0
	}
}

func statusCommand(args []string) int {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := fs.String("config", "", "path to config.yaml (default: OS config dir)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}

	healthURL := fmt.Sprintf("https://%s:%d/health", cfg.Host, cfg.Port)
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		fmt.Printf("FileAPI is not running (%s)\n", healthURL)
		fmt.Println("Start it with: fileapi serve")
		return 1
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read health response: %v\n", err)
		return 1
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("FileAPI returned HTTP %d from %s\n", resp.StatusCode, healthURL)
		return 1
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || !envelope.OK {
		fmt.Printf("FileAPI is running at %s but returned an unexpected health payload\n", healthURL)
		return 1
	}

	fmt.Printf("FileAPI is running\n")
	fmt.Printf("  version: %s\n", version)
	fmt.Printf("  health:  %s\n", healthURL)
	fmt.Printf("  api:     %s\n", cfg.BaseURL())
	fmt.Printf("  config:  %s\n", cfg.ConfigPath)
	return 0
}

func printBanner(cfg *config.Config) {
	fmt.Println()
	fmt.Println("FileAPI Local Agent")
	fmt.Println("-------------------")
	fmt.Printf("API base:    %s\n", cfg.BaseURL())
	fmt.Printf("Health:      https://%s:%d/health\n", cfg.Host, cfg.Port)
	fmt.Printf("Config:      %s\n", cfg.ConfigPath)
	fmt.Printf("Certificate: %s\n", cfg.CertFile)
	fmt.Println()
	fmt.Println("Trust the self-signed certificate in your OS and browser before connecting.")
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println()
}

func printUsage() {
	fmt.Println(`FileAPI local desktop agent

Usage:
  fileapi <command> [options]

Commands:
  serve     Start the HTTPS API server
  status    Check whether the agent is running
  version   Print the release version
  help      Show this help text

Examples:
  fileapi serve
  fileapi status
  fileapi version`)
}
