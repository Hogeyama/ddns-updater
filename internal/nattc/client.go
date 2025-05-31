package nattc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/Hogeyama/ddns-updater/internal/dns"
	kcp "github.com/xtaci/kcp-go/v5"
)

type Client struct {
	targetFQDN string
	listener   net.Listener
}

type Config struct {
	TargetFQDN string
}

func New(cfg Config) *Client {
	return &Client{
		targetFQDN: cfg.TargetFQDN,
	}
}

func (c *Client) Start(ctx context.Context, listenAddr string) error {
	// Start TCP listener
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to start TCP listener: %w", err)
	}
	c.listener = listener

	log.Printf("nattc: TCP listener started on %s", listenAddr)
	log.Printf("nattc: will proxy to %s", c.targetFQDN)

	// Accept connections
	go c.acceptLoop(ctx)

	return nil
}

func (c *Client) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := c.listener.Accept()
		if err != nil {
			log.Printf("nattc: failed to accept connection: %v", err)
			continue
		}

		go c.handleConnection(conn)
	}
}

func (c *Client) handleConnection(tcpConn net.Conn) {
	defer tcpConn.Close()

	log.Printf("nattc: new connection from %s", tcpConn.RemoteAddr())

	// Resolve target FQDN to get natts IP and port
	targetAddr, err := dns.ResolveTarget(c.targetFQDN)
	if err != nil {
		log.Printf("nattc: failed to resolve target: %v", err)
		return
	}

	log.Printf("nattc: resolved target to %s", targetAddr)

	// Connect to natts via KCP
	kcpConn, err := kcp.DialWithOptions(targetAddr, nil, 10, 3)
	if err != nil {
		log.Printf("nattc: failed to connect to natts: %v", err)
		return
	}
	defer kcpConn.Close()

	log.Printf("nattc: connected to natts at %s", targetAddr)

	// Proxy data between TCP and KCP connections
	done := make(chan error, 2)

	go func() {
		_, err := io.Copy(kcpConn, tcpConn)
		done <- err
	}()

	go func() {
		_, err := io.Copy(tcpConn, kcpConn)
		done <- err
	}()

	// Wait for either direction to complete
	err = <-done
	if err != nil {
		log.Printf("nattc: proxy error: %v", err)
	}

	log.Printf("nattc: connection closed")
}


func (c *Client) Close() error {
	if c.listener != nil {
		return c.listener.Close()
	}
	return nil
}

