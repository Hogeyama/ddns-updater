package natts

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Hogeyama/ddns-updater/internal/dns"
	"github.com/Hogeyama/ddns-updater/internal/stun"
	kcp "github.com/xtaci/kcp-go/v5"
)

type Server struct {
	cfToken    string
	sshTarget  string
	targetFQDN string
	listener   *kcp.Listener

	// Connection tracking
	connMutex    sync.RWMutex
	activeConns  int
	lastConnTime time.Time
	localPort    int
}

type Config struct {
	SSHTarget  string
	TargetFQDN string
	CFToken    string
}

func New(cfg Config) (*Server, error) {
	return &Server{
		cfToken:      cfg.CFToken,
		sshTarget:    cfg.SSHTarget,
		targetFQDN:   cfg.TargetFQDN,
		lastConnTime: time.Now(),
	}, nil
}

func (s *Server) Start(ctx context.Context, listenAddr string) error {
	// Parse listen address to get desired port
	var localPort int
	if listenAddr == ":0" {
		// For port 0, we need to discover first, then bind to that port
		externalIP, externalPort, err := stun.GetIPv4AndAvailableTcpPort()
		if err != nil {
			return fmt.Errorf("failed to discover external IP and port: %w", err)
		}
		log.Printf("natts: discovered external IP %s, port %d", externalIP, externalPort)
		
		// Update DNS records first
		dnsCtx := context.Background()
		if err := dns.UpdateRecords(dnsCtx, s.cfToken, s.targetFQDN, externalIP, externalPort); err != nil {
			return fmt.Errorf("failed to update DNS records: %w", err)
		}
		log.Printf("natts: DNS records updated for %s", s.targetFQDN)
		
		// Use the discovered port for KCP listener
		listenAddr = fmt.Sprintf(":%d", externalPort)
		localPort = externalPort
	} else {
		// Parse specific port from listenAddr
		_, portStr, err := net.SplitHostPort("localhost" + listenAddr)
		if err != nil {
			return fmt.Errorf("failed to parse listen address: %w", err)
		}
		port, err := net.LookupPort("udp", portStr)
		if err != nil {
			return fmt.Errorf("failed to parse port: %w", err)
		}
		localPort = port
		
		// Discover external IP and port via STUN using the specified port
		if err := s.discoverAndRegister(localPort); err != nil {
			return fmt.Errorf("failed to discover and register: %w", err)
		}
	}

	s.localPort = localPort

	// Start KCP listener on the determined port
	listener, err := kcp.ListenWithOptions(listenAddr, nil, 10, 3)
	if err != nil {
		return fmt.Errorf("failed to start KCP listener: %w", err)
	}
	s.listener = listener

	actualAddr := listener.Addr().(*net.UDPAddr)
	log.Printf("natts: KCP listener started on %s (actual port: %d)", listenAddr, actualAddr.Port)

	// Start connection monitoring
	go s.connectionMonitor(ctx)

	// Accept connections
	go s.acceptLoop(ctx)

	return nil
}

func (s *Server) discoverAndRegister(localPort int) error {
	// Discover external IP and port via STUN using the same port as KCP listener
	externalIP, externalPort, err := stun.GetIPv4FromLocalPort(localPort)
	if err != nil {
		return fmt.Errorf("failed to discover external IP and port: %w", err)
	}

	log.Printf("natts: discovered external IP %s, port %d (local port: %d)", externalIP, externalPort, localPort)

	// Update DNS records
	ctx := context.Background()
	if err := dns.UpdateRecords(ctx, s.cfToken, s.targetFQDN, externalIP, externalPort); err != nil {
		return fmt.Errorf("failed to update DNS records: %w", err)
	}

	log.Printf("natts: DNS records updated for %s", s.targetFQDN)
	return nil
}

func (s *Server) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := s.listener.AcceptKCP()
		if err != nil {
			log.Printf("natts: failed to accept connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(kcpConn *kcp.UDPSession) {
	defer kcpConn.Close()

	log.Printf("natts: new connection from %s", kcpConn.RemoteAddr())

	// Configure KCP session timeout
	kcpConn.SetDeadline(time.Now().Add(5 * time.Minute))

	// Track connection start
	s.connMutex.Lock()
	s.activeConns++
	connCount := s.activeConns
	s.connMutex.Unlock()

	log.Printf("natts: active connections: %d", connCount)

	// Track connection end
	defer func() {
		s.connMutex.Lock()
		s.activeConns--
		s.lastConnTime = time.Now()
		connCount := s.activeConns
		s.connMutex.Unlock()
		log.Printf("natts: connection closed, active connections: %d", connCount)
	}()

	// Connect to local SSH server
	sshConn, err := net.DialTimeout("tcp", s.sshTarget, 10*time.Second)
	if err != nil {
		log.Printf("natts: failed to connect to SSH server: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("natts: connected to SSH server %s", s.sshTarget)

	// Proxy data between KCP and SSH connections
	done := make(chan error, 2)

	go func() {
		_, err := io.Copy(sshConn, kcpConn)
		done <- err
	}()

	go func() {
		_, err := io.Copy(kcpConn, sshConn)
		done <- err
	}()

	// Wait for either direction to complete
	err = <-done
	if err != nil {
		log.Printf("natts: proxy error: %v", err)
	}
}

func (s *Server) connectionMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.connMutex.RLock()
			activeConns := s.activeConns
			lastConnTime := s.lastConnTime
			s.connMutex.RUnlock()

			// If no active connections and it's been 10 minutes since last connection
			if activeConns == 0 && time.Since(lastConnTime) > 10*time.Minute {
				log.Printf("natts: no connections for 10 minutes, restarting STUN discovery")

				// Restart STUN discovery and DNS registration
				if err := s.discoverAndRegister(s.localPort); err != nil {
					log.Printf("natts: failed to restart STUN discovery: %v", err)
				} else {
					// Update last connection time to prevent immediate re-trigger
					s.connMutex.Lock()
					s.lastConnTime = time.Now()
					s.connMutex.Unlock()
					log.Printf("natts: STUN discovery completed")
				}
			}
		}
	}
}

func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

