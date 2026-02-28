package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"sfsd/internal/config"
	"sfsd/internal/middleware"
	"sfsd/internal/server"

	"gopkg.in/yaml.v3"
)

const version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version":
		fmt.Printf("sfsd (File Server) version %s\n", version)
		os.Exit(0)

	case "config":
		cfg := config.CreateDefaultConfig()
		out, _ := yaml.Marshal(cfg)
		fmt.Print(string(out))
		os.Exit(0)

	case "launch":
		if len(os.Args) < 3 {
			fmt.Println("Error: 'launch' command requires a config file path.")
			printUsage()
			os.Exit(1)
		}
		configPath := os.Args[2]
		cfg, err := config.Load(configPath)
		if err != nil {
			log.Fatalf("Failed to init file server: %v\n", err)
		}
		startServer(cfg)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [command] [arguments]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  launch <config.yaml>   Start the server with the specified config file\n")
	fmt.Fprintf(os.Stderr, "  version                Print the version and exit\n")
	fmt.Fprintf(os.Stderr, "  config                 Print the default config (example) and exit\n")
}

func startServer(cfg *config.Config) {
	fmt.Printf("Starting file server on %s:%d (TLS: %v)\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.TLS.Enabled)
	fmt.Printf("Serving directory: %s\n", cfg.Directory.Path)

	middleware.InitCounter(cfg.Features.StatsFile)

	go func() {
		if err := server.Start(cfg); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	fmt.Println("\nShutting down server...")
	middleware.SaveStats()
	os.Exit(0)
}
