# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based NAT traversal system for SSH connections using Cloudflare DNS for service discovery. The system provides a complete NAT traversal solution using reliable UDP (KCP) proxies.

### Current State

- **DDNS updater** that automatically updates Cloudflare DNS records with public IP/port via STUN
- **natts** (NAT Traversal TCP Server) - KCP-based UDP-to-TCP proxy server for SSH connections
- **nattc** (NAT Traversal TCP Client) - KCP-based TCP-to-UDP proxy client for SSH connections
- Complete SSH NAT traversal solution using reliable UDP (KCP) tunneling

**Target scenario**: External host (e.g., EC2) connects to PC behind full-cone NAT via SSH  
(e.g., `ssh -p 10022 localhost` on EC2 connects to NAT-ed PC's SSH server)

**Architecture**:

```
[SSH Client]
        |
        v (TCP)
[TCP-to-UDP Proxy Client (nattc)]
        |
        v (KCP over UDP)
[UDP packets through NAT]
        |
        v (KCP over UDP)
[UDP-to-TCP Proxy Server (natts)]
        |
        v (TCP)
[Local SSH Server (127.0.0.1:22)]
```

The proxy server (natts) discovers its external IP address and port via STUN, then registers them in DNS. The proxy client (nattc) resolves that DNS name and connects to the proxy server using KCP (reliable UDP) for secure, ordered data transmission.

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
nix develop -c ./test-e2e.sh

# Run applications (require environment variables)
./natts         # NAT traversal server (NAT内で実行)
./nattc         # NAT traversal client (外部ホストで実行)
```

## Configuration

Both applications support configuration via command-line flags or environment variables. Command-line flags take precedence over environment variables.

### For natts

Command-line flags:

- `--cf-token` - Cloudflare API token with DNS edit permissions
- `--target-fqdn` - Fully qualified domain name to update
- `--ssh-target` - SSH server to proxy to (default: "127.0.0.1:22")
- `--listen` - Address to listen on (default: ":30000")

Environment variables (fallback):

- `CF_API_TOKEN` - Cloudflare API token with DNS edit permissions
- `TARGET_FQDN` - Fully qualified domain name to update

### For nattc

Command-line flags:

- `--target` - Target FQDN to connect to (natts server)
- `--listen` - Address to listen on for SSH connections in server mode (default: ":10022")
- `--proxy` - Run in ProxyCommand mode (stdin/stdout)

Environment variables (fallback):

- `TARGET_FQDN` - FQDN to resolve for connecting to natts server

## Usage Examples

### Running natts (on NAT-ed machine)

```bash
# Start natts server that will register itself in DNS
./natts --cf-token your_token --target-fqdn mypc.example.com --ssh-target 127.0.0.1:22 --listen :30000

# Or using environment variables
CF_API_TOKEN=your_token TARGET_FQDN=mypc.example.com ./natts --ssh-target 127.0.0.1:22 --listen :30000
```

### Running nattc (on external machine, e.g., EC2)

#### Option 1: Server Mode (TCP Listener)

```bash
# Start nattc client that connects to natts via DNS resolution
./nattc --listen :10022 --target mypc.example.com

# Now SSH to the NAT-ed machine via the proxy
ssh -p 10022 localhost
```

#### Option 2: ProxyCommand Mode (Recommended)

```bash
# Direct SSH connection using ProxyCommand
ssh -o ProxyCommand='./nattc --proxy --target mypc.example.com' user@dummy

# Or add to ~/.ssh/config:
# Host mypc
#     ProxyCommand /path/to/nattc --proxy --target mypc.example.com
#     User your_username
#
# Then simply: ssh mypc
```

## Dependencies

- **Cloudflare DNS** - Required for DNS record management and service discovery
- `github.com/cloudflare/cloudflare-go` - Cloudflare API client
- `github.com/pion/stun` - STUN protocol implementation
- `github.com/xtaci/kcp-go/v5` - KCP (reliable UDP) library for secure, ordered UDP transmission

The project uses Go modules and Nix flakes for dependency management and reproducible builds.

### External Service Requirements

- **Cloudflare account** with API token having DNS edit permissions
- **Domain managed by Cloudflare** for DNS record updates
- **STUN server access** for NAT traversal (uses public STUN servers)

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
