package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hogeyama/ddns-updater/internal/nattc"
)

func main() {
	var (
		listenAddr  = flag.String("listen", ":10022", "Address to listen on for SSH connections (server mode)")
		targetFQDN  = flag.String("target", "", "Target FQDN to connect to (natts server)")
		proxyMode   = flag.Bool("proxy", false, "Run in ProxyCommand mode (stdin/stdout)")
	)
	flag.Parse()

	// Get target FQDN from environment if not provided via flag
	if *targetFQDN == "" {
		*targetFQDN = os.Getenv("TARGET_FQDN")
	}

	if *targetFQDN == "" {
		log.Fatal("TARGET_FQDN is required (via -target flag or TARGET_FQDN environment variable)")
	}

	if *proxyMode {
		// ProxyCommand mode: proxy stdin/stdout
		proxyClient := nattc.NewProxyClient(*targetFQDN)
		if err := proxyClient.RunProxy(); err != nil {
			log.Fatalf("Proxy failed: %v", err)
		}
		return
	}

	// Server mode: TCP listener
	// Create client
	client := nattc.New(nattc.Config{
		TargetFQDN: *targetFQDN,
	})

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

	// Start client
	log.Printf("Starting nattc client...")
	log.Printf("  Listen address: %s", *listenAddr)
	log.Printf("  Target FQDN: %s", *targetFQDN)
	log.Printf("  Usage: ssh -p %s localhost", (*listenAddr)[1:]) // Remove ':' from port

	if err := client.Start(ctx, *listenAddr); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Wait for shutdown
	<-ctx.Done()
	log.Println("Shutting down...")

	if err := client.Close(); err != nil {
		log.Printf("Error closing client: %v", err)
	}

	log.Println("Client stopped")
}