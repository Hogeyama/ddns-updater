package main

import (
	"context"
	"log"
	"os"

	"github.com/Hogeyama/ddns-updater/internal/network"
	"github.com/Hogeyama/ddns-updater/internal/updater"
)

func main() {
	apiToken := os.Getenv("CF_API_TOKEN")
	fqdn := os.Getenv("TARGET_FQDN")

	if apiToken == "" || fqdn == "" {
		log.Fatal("CF_API_TOKEN, TARGET_FQDN must be set")
	}

	ipv4, err := network.GetGlobalIPv4()
	if err != nil {
		log.Fatalf("failed to get IPv4 address: %v", err)
	}

	err = updater.UpdateIPv4Record(context.Background(), apiToken, fqdn, ipv4)
	if err != nil {
		log.Fatalf("failed to update DNS record: %v", err)
	}

	log.Printf("Successfully updated %s to %s", fqdn, ipv4)
}
