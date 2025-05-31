package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hogeyama/ddns-updater/internal/natts"
)

func main() {
	var (
		listenAddr = flag.String("listen", ":0", "Address to listen on (e.g., :8080)")
		sshTarget  = flag.String("ssh-target", "127.0.0.1:22", "SSH server to proxy to")
		targetFQDN = flag.String("target-fqdn", "", "FQDN to register in DNS")
		cfToken    = flag.String("cf-token", "", "Cloudflare API token")
	)
	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  --cf-token string\n")
		fmt.Fprintf(os.Stderr, "    \tCloudflare API token\n")
		fmt.Fprintf(os.Stderr, "  --listen string\n")
		fmt.Fprintf(os.Stderr, "    \tAddress to listen on (e.g., :8080) (default \":0\")\n")
		fmt.Fprintf(os.Stderr, "  --ssh-target string\n")
		fmt.Fprintf(os.Stderr, "    \tSSH server to proxy to (default \"127.0.0.1:22\")\n")
		fmt.Fprintf(os.Stderr, "  --target-fqdn string\n")
		fmt.Fprintf(os.Stderr, "    \tFQDN to register in DNS\n")
	}
	flag.Parse()

	// Get values from environment if not provided via flags
	if *cfToken == "" {
		*cfToken = os.Getenv("CF_API_TOKEN")
	}
	if *targetFQDN == "" {
		*targetFQDN = os.Getenv("TARGET_FQDN")
	}

	if *cfToken == "" {
		log.Fatal("CF_API_TOKEN is required (via flag or environment variable)")
	}
	if *targetFQDN == "" {
		log.Fatal("TARGET_FQDN is required (via flag or environment variable)")
	}

	// Create server
	server, err := natts.New(natts.Config{
		SSHTarget:  *sshTarget,
		TargetFQDN: *targetFQDN,
		CFToken:    *cfToken,
	})
	if err != nil {
		log.Fatalf("Failed to create natts server: %v", err)
	}

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Start server
	log.Printf("Starting natts server...")
	log.Printf("  SSH target: %s", *sshTarget)
	log.Printf("  Target FQDN: %s", *targetFQDN)
	log.Printf("  Listen address: %s", *listenAddr)

	if err := server.Start(ctx, *listenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for shutdown
	<-ctx.Done()
	log.Println("Shutting down...")

	if err := server.Close(); err != nil {
		log.Printf("Error closing server: %v", err)
	}

	log.Println("Server stopped")
}