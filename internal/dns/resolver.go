package dns

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ResolveTarget resolves FQDN to get IP and port from TXT record with kcp-port prefix
func ResolveTarget(fqdn string) (string, error) {
	// Resolve A record to get IP
	ips, err := net.LookupIP(fqdn)
	if err != nil {
		return "", fmt.Errorf("failed to lookup IP for %s: %w", fqdn, err)
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found for %s", fqdn)
	}

	ip := ips[0]

	// Resolve TXT record to get port
	txtRecords, err := net.LookupTXT(fqdn)
	if err != nil {
		return "", fmt.Errorf("failed to lookup TXT records for %s: %w", fqdn, err)
	}

	var port string
	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "kcp-port=") {
			port = strings.TrimPrefix(txt, "kcp-port=")
			break
		}
	}

	if port == "" {
		return "", fmt.Errorf("no port found in TXT records for %s", fqdn)
	}

	// Validate port
	if _, err := strconv.Atoi(port); err != nil {
		return "", fmt.Errorf("invalid port in TXT record: %s", port)
	}

	return net.JoinHostPort(ip.String(), port), nil
}