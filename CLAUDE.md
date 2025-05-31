# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based NAT traversal system for SSH connections using Cloudflare DNS for service discovery. The system provides a complete NAT traversal solution using reliable UDP (KCP) proxies.

### Current Components

- **DDNS updater** that automatically updates Cloudflare DNS records with public IP/port via STUN
- **natts** (NAT Traversal TCP Server) - KCP-based UDP-to-TCP proxy server for SSH connections
- **nattc** (NAT Traversal TCP Client) - KCP-based TCP-to-UDP proxy client for SSH connections
- Complete SSH NAT traversal solution using reliable UDP (KCP) tunneling

## Current Architecture

The codebase follows a simple package-based structure:

### Core Components

- `cmd/natts/main.go` - NAT Traversal TCP Server entry point
- `cmd/nattc/main.go` - NAT Traversal TCP Client entry point

### Internal Packages

- `internal/stun/ipv4.go` - STUN client for IP/port discovery via TCP connection to STUN server
- `internal/dns/updater.go` - Cloudflare DNS API client for updating A and TXT records
- `internal/dns/resolver.go` - DNS resolver for finding natts server IP/port from FQDN and TXT records
- `internal/natts/server.go` - KCP-based UDP-to-TCP proxy server implementation
- `internal/nattc/client.go` - KCP-based TCP-to-UDP proxy client implementation (server mode)
- `internal/nattc/proxy.go` - KCP-based proxy client for SSH ProxyCommand mode

### Workflows

- **natts**: KCP listener → STUN discovery → DNS registration → SSH proxy
- **nattc (Server Mode)**: TCP listener → DNS resolution → KCP connection → SSH proxy
- **nattc (ProxyCommand Mode)**: stdin/stdout ↔ DNS resolution → KCP connection → SSH proxy

## Build and Development Commands

```bash
# Build the application (using Nix flake)
nix build

# Build for ARM64
nix build .#arm64

# Enter development shell
nix develop

# Build manually with Go (within dev shell)
go build -o natts ./cmd/natts
go build -o nattc ./cmd/nattc

# Run tests
go test ./...

# Run end-to-end tests (run after any modifications)
(source .envrc.local && ./test-e2e.sh)

# Run applications (require environment variables)
./natts         # NAT traversal server (NAT内で実行)
./nattc         # NAT traversal client (外部ホストで実行)
```

## Dependencies and External Services

- `github.com/cloudflare/cloudflare-go` - Cloudflare API client
- `github.com/pion/stun` - STUN protocol implementation
- `github.com/xtaci/kcp-go/v5` - KCP (reliable UDP) library for secure, ordered UDP transmission
- **Cloudflare account** with API token having DNS edit permissions
- **Domain managed by Cloudflare** for DNS record updates
- **STUN server access** for NAT traversal (uses public STUN servers)

The project uses Go modules and Nix flakes for dependency management and reproducible builds.

## Development Roadmap

### Completed ✅

- [x] Design UDP-to-TCP proxy server (natts) architecture and interface
- [x] Design TCP-to-UDP proxy client (nattc) architecture and interface
- [x] Implement UDP-to-TCP proxy server (natts) using KCP for reliable UDP
- [x] Implement TCP-to-UDP proxy client (nattc) using KCP for reliable UDP
- [x] Integrate STUN IP/port discovery into natts for self-registration
- [x] Integrate Cloudflare DNS registration into natts using existing dns package
- [x] Add command-line interface and configuration for natts and nattc
- [x] Add error handling and logging to both proxy components
- [x] Update build system to include both natts and nattc binaries
- [x] Update documentation to explain the new proxy architecture and usage
- [x] Fix internal/dns/updater.go to create new DNS records when they don't exist instead of erroring
- [x] Add DNS resolver functionality for nattc to find natts server via DNS
- [x] Add end-to-end testing script for complete workflow validation

### TODO

#### Architecture Improvements

**High Priority:**

- [ ] Externalize hardcoded configuration values (STUN server, TXT record prefix, KCP parameters)
- [ ] Fix TXT record double-quoting issue and verify proper cloudflare-go library usage
- [ ] Refactor natts.Server.Start method complexity (responsibility separation, method splitting)

**Medium Priority:**

- [ ] Improve STUN error handling (properly handle SetDeadline errors)
- [ ] Introduce structured logging (log/slog or external library like zap/logrus)
- [ ] Fix context usage (use passed ctx instead of context.Background() in DNS operations)
- [ ] Implement graceful shutdown for active connections in natts server
- [ ] Deduplicate code between nattc/client.go and nattc/proxy.go into common helper functions

**Low Priority:**

- [ ] Optimize Cloudflare API client reuse (avoid repeated initialization)
- [ ] Improve testability through interface abstraction (Cloudflare API, STUN, KCP mocking)
