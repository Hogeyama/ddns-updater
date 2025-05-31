package nattc

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Hogeyama/ddns-updater/internal/dns"
	kcp "github.com/xtaci/kcp-go/v5"
)

// ProxyClient implements ProxyCommand functionality for SSH
type ProxyClient struct {
	targetFQDN string
}

func NewProxyClient(targetFQDN string) *ProxyClient {
	return &ProxyClient{
		targetFQDN: targetFQDN,
	}
}

// RunProxy connects to natts and proxies stdin/stdout for SSH ProxyCommand
func (p *ProxyClient) RunProxy() error {
	// Resolve target FQDN to get natts IP and port
	targetAddr, err := dns.ResolveTarget(p.targetFQDN)
	if err != nil {
		return fmt.Errorf("failed to resolve target: %w", err)
	}

	log.Printf("nattc-proxy: resolved target to %s", targetAddr)

	// Connect to natts via KCP
	kcpConn, err := kcp.DialWithOptions(targetAddr, nil, 10, 3)
	if err != nil {
		return fmt.Errorf("failed to connect to natts: %w", err)
	}
	defer kcpConn.Close()

	log.Printf("nattc-proxy: connected to natts at %s", targetAddr)

	// Proxy data between stdin/stdout and KCP connection
	done := make(chan error, 2)

	// Copy from stdin to KCP
	go func() {
		_, err := io.Copy(kcpConn, os.Stdin)
		done <- err
	}()

	// Copy from KCP to stdout
	go func() {
		_, err := io.Copy(os.Stdout, kcpConn)
		done <- err
	}()

	// Wait for either direction to complete
	err = <-done
	if err != nil {
		log.Printf("nattc-proxy: proxy error: %v", err)
	}

	log.Printf("nattc-proxy: connection closed")
	return err
}

