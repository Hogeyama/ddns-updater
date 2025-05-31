# NAT Traversal SSH System

**Note**: This is a personal project designed for my specific use case and environment. It may not be suitable for general use without modifications.

A Go-based NAT traversal system for SSH connections using Cloudflare DNS for service discovery and KCP (reliable UDP) for tunneling.

## Architecture

**Target scenario**: External host connects to PC behind full-cone NAT via SSH  
(e.g., `ssh -p 10022 localhost` connects to NAT-ed PC's SSH server)

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

## Limitations

**Important**: This system only works with **Full Cone NAT**. It will not work with:
- Symmetric NAT
- Port-restricted NAT
- Address-restricted NAT

Most home routers use Full Cone NAT, but some enterprise firewalls and cellular networks may use more restrictive NAT types that are incompatible with this system.

## Requirements

- **Cloudflare account** with API token having DNS edit permissions
- **Domain managed by Cloudflare** for DNS record updates
- **STUN server access** for NAT traversal (uses public STUN servers)

## Installation

### Using Nix (Recommended)

```bash
# Build the application
nix build

# Build for ARM64
nix build .#arm64
```

### Manual Build

```bash
# Clone the repository
git clone <repository-url>
cd ddns

# Build with Go
go build -o natts ./cmd/natts
go build -o nattc ./cmd/nattc
```

## Configuration

Both applications support configuration via command-line flags or environment variables. Command-line flags take precedence over environment variables.

### For natts (NAT Traversal Server)

Command-line flags:
- `--cf-token` - Cloudflare API token with DNS edit permissions
- `--target-fqdn` - Fully qualified domain name to update
- `--ssh-target` - SSH server to proxy to (default: "127.0.0.1:22")
- `--listen` - Address to listen on (default: ":30000")

Environment variables (fallback):
- `CF_API_TOKEN` - Cloudflare API token with DNS edit permissions
- `TARGET_FQDN` - Fully qualified domain name to update

### For nattc (NAT Traversal Client)

Command-line flags:
- `--target` - Target FQDN to connect to (natts server)
- `--listen` - Address to listen on for SSH connections in server mode (default: ":10022")
- `--proxy` - Run in ProxyCommand mode (stdin/stdout)

Environment variables (fallback):
- `TARGET_FQDN` - FQDN to resolve for connecting to natts server

## Usage

### Step 1: Run natts (on NAT-ed machine)

```bash
# Start natts server that will register itself in DNS
./natts --cf-token your_token --target-fqdn mypc.example.com --ssh-target 127.0.0.1:22 --listen :30000

# Or using environment variables
CF_API_TOKEN=your_token TARGET_FQDN=mypc.example.com ./natts --ssh-target 127.0.0.1:22 --listen :30000
```

### Step 2: Run nattc (on external machine)

#### Option 1: Server Mode (TCP Listener)

```bash
# Start nattc client that connects to natts via DNS resolution
./nattc --listen :10022 --target mypc.example.com

# Now SSH to the NAT-ed machine via the proxy (with KeepAlive)
ssh -o ServerAliveInterval=60 -o ServerAliveCountMax=3 -p 10022 localhost
```

#### Option 2: ProxyCommand Mode (Recommended)

```bash
# Direct SSH connection using ProxyCommand with KeepAlive
ssh -o ProxyCommand='./nattc --proxy --target mypc.example.com' \
    -o ServerAliveInterval=60 \
    -o ServerAliveCountMax=3 \
    user@dummy

# Or add to ~/.ssh/config:
# Host mypc
#     ProxyCommand /path/to/nattc --proxy --target mypc.example.com
#     User your_username
#     ServerAliveInterval 60
#     ServerAliveCountMax 3
#
# Then simply: ssh mypc
```

## Important: SSH KeepAlive Configuration

**KeepAlive settings are essential** because natts uses a 5-minute connection timeout. Without KeepAlive, idle SSH sessions will be disconnected after 5 minutes.

**Recommended SSH client settings:**
- `ServerAliveInterval 60` - Send keepalive every 60 seconds
- `ServerAliveCountMax 3` - Allow up to 3 missed keepalives

**For ~/.ssh/config:**
```
Host mypc
    ProxyCommand /path/to/nattc --proxy --target mypc.example.com
    User your_username
    ServerAliveInterval 60
    ServerAliveCountMax 3
```

**For SSH server (/etc/ssh/sshd_config):**
```
ClientAliveInterval 60
ClientAliveCountMax 3
```

## Dependencies

- **Cloudflare DNS** - Required for DNS record management and service discovery
- `github.com/cloudflare/cloudflare-go` - Cloudflare API client
- `github.com/pion/stun` - STUN protocol implementation
- `github.com/xtaci/kcp-go/v5` - KCP (reliable UDP) library for secure, ordered UDP transmission

The project uses Go modules and Nix flakes for dependency management and reproducible builds.

## Development

See [CLAUDE.md](./CLAUDE.md) for detailed development instructions and technical documentation.
