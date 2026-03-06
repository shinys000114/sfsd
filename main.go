package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"sfsd/internal/config"
	"sfsd/internal/middleware"
	"sfsd/internal/server"

	"gopkg.in/yaml.v3"
)

var version = "dev"

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

	case "gen-cert":
		outputDir := "./certs"
		algo := "rsa"
		if len(os.Args) >= 3 {
			outputDir = os.Args[2]
		}
		if len(os.Args) >= 4 {
			algo = os.Args[3]
		}
		if err := server.GenerateSelfSignedCert(outputDir, algo); err != nil {
			log.Fatalf("Failed to generate certificate: %v\n", err)
		}
		os.Exit(0)

	case "clean-stats":
		if len(os.Args) < 4 {
			fmt.Println("Error: 'clean-stats' requires <directory> and <stats.json> paths.")
			printUsage()
			os.Exit(1)
		}
		dirPath := os.Args[2]
		statsPath := os.Args[3]

		middleware.InitCounter(statsPath)

		absDir, err := filepath.Abs(dirPath)
		if err != nil {
			log.Fatalf("Failed to resolve absolute path for serving directory: %v", err)
		}

		// Iterate through tracked files and check physical existence
		fmt.Printf("Cleaning up stats file '%s' against directory '%s'...\n", statsPath, absDir)

		var removed int
		middleware.ExportDownloadStats(func(key string, value uint64) {
			cleanPath := filepath.Clean(key)
			if cleanPath == "/" {
				cleanPath = ""
			}
			fullPath := filepath.Join(absDir, cleanPath)

			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				middleware.DeleteStat(key)
				fmt.Printf("Removed non-existent file from stats: %s\n", key)
				removed++
			}
		})

		middleware.SaveStats()
		fmt.Printf("Cleanup complete. Removed %d entries.\n", removed)
		os.Exit(0)

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
	fmt.Fprintf(os.Stderr, "  gen-cert [path] [algo] Generate self-signed TLS cert (rsa|rsa2048|rsa4096|ecdsa|ed25519)\n")
	fmt.Fprintf(os.Stderr, "  version                Print the version and exit\n")
	fmt.Fprintf(os.Stderr, "  config                 Print the default config (example) and exit\n")
	fmt.Fprintf(os.Stderr, "  clean-stats <dir> <st> Remove non-existent files from stats.json\n")
}

func startServer(cfg *config.Config) {
	// Group instances by address
	groups := make(map[string]map[string]*config.ServerInstance)
	for name, instance := range cfg.Servers {
		addr := fmt.Sprintf("%s:%d", instance.Server.Host, instance.Server.Port)
		if groups[addr] == nil {
			groups[addr] = make(map[string]*config.ServerInstance)
		}
		groups[addr][name] = &instance

		if instance.Features.StatsFile != "" {
			middleware.InitCounter(instance.Features.StatsFile)
		}
	}

	fmt.Printf("Launching servers across %d unique addresses...\n", len(groups))

	for addr, instances := range groups {
		addr := addr
		instances := instances
		go func() {
			if err := server.StartGroup(addr, instances); err != nil {
				log.Fatalf("Server at %s failed: %v", addr, err)
			}
		}()
	}

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	fmt.Println("\nShutting down all servers...")
	middleware.SaveStats()
	os.Exit(0)
}
