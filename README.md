# 🌐 Traefik RealIP Plugin

[![Build Status](https://github.com/david-garcia-garcia/traefik-realip/actions/workflows/ci.yml/badge.svg)](https://github.com/david-garcia-garcia/traefik-realip/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/david-garcia-garcia/traefik-realip)](https://goreportcard.com/report/github.com/david-garcia-garcia/traefik-realip)
[![Latest GitHub release](https://img.shields.io/github/v/release/david-garcia-garcia/traefik-realip?sort=semver)](https://github.com/david-garcia-garcia/traefik-realip/releases/latest)
[![License](https://img.shields.io/badge/license-Apache%202.0-brightgreen.svg)](LICENSE)

A Traefik plugin that extracts the real client IP address from proxy headers and populates a specified request header.

## ✨ Features

- **Depth-based IP extraction**: Configure exactly which IP to extract from comma-separated lists
- **Flexible header configuration**: Define headers with custom depth settings for precise control
- **Synthetic headers**: Special `clientAddress` header provides direct access to `request.RemoteAddr`
- **Anti-spoofing protection**: `forceOverwrite` option prevents header injection attacks
- **Trust-based security**: Only process headers from trusted proxy sources using fast radix tree lookups
- **Trust indication**: Optional header to indicate if the request source was trusted
- **Smart IP extraction**: Handles multiple IPs, port numbers, and IPv6 addresses
- **Header priority**: Processes headers in configured order with fallback support
- **Port stripping**: Automatically removes port numbers from IP addresses
- **IPv6 support**: Full support for IPv6 addresses including bracketed notation with ports
- **Enable/disable control**: Easy on/off switch for the plugin functionality
- **High performance**: O(log k) IP lookups using radix trees for trusted IP checking
- **Access log integration**: Extracted IPs appear in Traefik access logs

## 📥 Installation

### Local Plugin Installation

Create or modify your Traefik static configuration:

```yaml
experimental:
  localPlugins:
    realip:
      moduleName: github.com/david-garcia-garcia/traefik-realip
```

Clone the plugin into your container:

```dockerfile
# Create the directory for the plugins
RUN set -eux; \
    mkdir -p /plugins-local/src/github.com/david-garcia-garcia

RUN set -eux && git clone https://github.com/david-garcia-garcia/traefik-realip /plugins-local/src/github.com/david-garcia-garcia/traefik-realip --branch v1.0.0 --single-branch
```

### Traefik Plugin Registry Installation

Add to your Traefik static configuration:

```yaml
experimental:
  plugins:
    realip:
      moduleName: github.com/david-garcia-garcia/traefik-realip
      version: v1.0.0
```

## 🧪 Testing and Development

You can spin up a fully working environment with docker compose:

```bash
docker compose up --build
```

The codebase includes a full set of integration and unit tests:

```bash
# Run unit tests
go test -v

# Run integration tests (PowerShell)
./Test-Integration.ps1

# Run integration tests (Cross-platform)
pwsh ./Test-Integration.ps1
```

## ⚙️ Configuration

### Basic Configuration

```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true                    # Enable/disable the plugin
          headerName: "X-Real-IP"          # Header to populate with the real IP
          processHeaders:                  # Headers to check for IP addresses (in order)
            - headerName: "X-Forwarded-For"
              depth: -1                    # Leftmost IP (original client)
            - headerName: "CF-Connecting-IP"
              depth: -1                    # Leftmost IP
            - headerName: "clientAddress"  # Synthetic header for request.RemoteAddr
              depth: -1
          forceOverwrite: true             # Always set header (prevents spoofing)
          trustAll: true                   # Trust all sources (default)
          trustedIPs: []                   # CIDR blocks of trusted proxies
          trustedHeader: ""                # Optional trust indication header
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable or disable the plugin |
| `headerName` | string | `"X-Real-IP"` | Name of the header to populate with the extracted IP |
| `processHeaders` | array of objects | See below | List of headers to process with depth configuration |
| `forceOverwrite` | boolean | `true` | Always set the header, even if empty (prevents header spoofing) |
| `trustAll` | boolean | `true` | Trust all sources (if false, trustedIPs must be configured) |
| `trustedIPs` | array of strings | `[]` | CIDR blocks of trusted proxy IPs (required if trustAll is false) |
| `trustedHeader` | string | `""` | Header name to indicate trust status (e.g., "X-Is-Trusted") |

#### ProcessHeaders Configuration

Each header in `processHeaders` is an object with:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `headerName` | string | required | Name of the header to check for IP addresses |
| `depth` | integer | `-1` | IP extraction depth: `-1` = leftmost, `0` = rightmost, `1` = second from right, etc. |

**Default processHeaders:**
```yaml
processHeaders:
  - headerName: "X-Forwarded-For"
    depth: -1
  - headerName: "X-Real-IP"  
    depth: -1
  - headerName: "CF-Connecting-IP"
    depth: -1
  - headerName: "clientAddress"  # Synthetic header mapping to request.RemoteAddr
    depth: -1
```

### Example Docker Compose Setup

```yaml
version: "3.8"

services:
  traefik:
    image: traefik:v3.0
    command:
      - --api.insecure=true
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --entrypoints.web.address=:80
      - --experimental.localPlugins.realip.moduleName=github.com/david-garcia-garcia/traefik-realip
    ports:
      - "8080:8080"  # Traefik dashboard
      - "80:80"      # HTTP
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik-realip:/plugins-local/src/github.com/david-garcia-garcia/traefik-realip:ro

  app:
    image: your-app:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.app.rule=Host(`app.localhost`)"
      - "traefik.http.routers.app.entrypoints=web"
      - "traefik.http.routers.app.middlewares=realip"
      - "traefik.http.middlewares.realip.plugin.realip.enabled=true"
      - "traefik.http.middlewares.realip.plugin.realip.headerName=X-Real-IP"
      - "traefik.http.middlewares.realip.plugin.realip.processHeaders[0].headerName=X-Forwarded-For"
      - "traefik.http.middlewares.realip.plugin.realip.processHeaders[0].depth=-1"
      - "traefik.http.middlewares.realip.plugin.realip.processHeaders[1].headerName=CF-Connecting-IP" 
      - "traefik.http.middlewares.realip.plugin.realip.processHeaders[1].depth=-1"
      - "traefik.http.middlewares.realip.plugin.realip.forceOverwrite=true"
```

## 🔄 How It Works

The plugin processes requests in the following order:

1. **Check if enabled**: If `enabled` is `false`, the plugin passes the request through unchanged
2. **Process headers in order**: Iterates through the `processHeaders` list in the specified order
3. **Extract IPs from headers**: For each header configuration:
   - Gets header value (or `req.RemoteAddr` for synthetic `clientAddress` header)
   - Splits comma-separated IP addresses
   - Cleans each IP (removes whitespace and port numbers)
   - Validates IP address format
   - Applies depth logic to select the appropriate IP
4. **Apply depth logic**: 
   - `depth: -1` = Leftmost IP (original client)
   - `depth: 0` = Rightmost IP (last proxy)
   - `depth: 1` = Second from right, etc.
   - If depth is out of bounds, skip to next header
5. **Set the result**: Populates the `headerName` header with the selected IP
6. **Check source trust**: If `trustAll` is false, verify the request source IP against `trustedIPs`
7. **Set trust header**: If `trustedHeader` is configured, add "yes"/"no" to indicate trust status
8. **Apply trust filtering**: Untrusted sources can only use synthetic headers (like `clientAddress`)
9. **Force overwrite**: If `forceOverwrite` is true, always sets the header (even if empty) to prevent spoofing
10. **Forward the request**: Passes the modified request to the next handler

### Synthetic Headers

**`clientAddress`** - Special synthetic header that maps directly to `req.RemoteAddr`
- Provides access to the actual network connection's remote address
- Useful as a fallback when no proxy headers are available
- Automatically handles port stripping like other headers
- **Always processed regardless of trust status** (cannot be spoofed)

### Trust-Based Security

When `trustAll` is set to `false`, the plugin implements trust-based header processing:

- **Trusted sources**: Process all configured headers normally
- **Untrusted sources**: Only process synthetic headers (like `clientAddress`)
- **Trust verification**: Uses fast radix tree lookups to check if `request.RemoteAddr` is in `trustedIPs`
- **Trust indication**: Optional `trustedHeader` adds "yes"/"no" to indicate trust status

This prevents header spoofing attacks where malicious clients send fake proxy headers.

### Header Processing Examples

#### Single IP Address
```yaml
Configuration:
  processHeaders:
    - headerName: "X-Forwarded-For"
      depth: -1

Header: X-Forwarded-For: 203.0.113.1
Result: X-Real-IP: 203.0.113.1
```

#### Multiple IP Addresses with Depth Control
```yaml
Configuration:
  processHeaders:
    - headerName: "X-Forwarded-For"
      depth: -1  # Leftmost

Header: X-Forwarded-For: 203.0.113.1, 198.51.100.1, 192.168.1.1
Result: X-Real-IP: 203.0.113.1  (leftmost IP)
```

```yaml
Configuration:
  processHeaders:
    - headerName: "X-Forwarded-For"
      depth: 0   # Rightmost

Header: X-Forwarded-For: 203.0.113.1, 198.51.100.1, 192.168.1.1
Result: X-Real-IP: 192.168.1.1  (rightmost IP)
```

```yaml
Configuration:
  processHeaders:
    - headerName: "X-Forwarded-For"
      depth: 1   # Second from right

Header: X-Forwarded-For: 203.0.113.1, 198.51.100.1, 192.168.1.1
Result: X-Real-IP: 198.51.100.1  (second from right)
```

#### Synthetic clientAddress Header
```yaml
Configuration:
  processHeaders:
    - headerName: "clientAddress"
      depth: -1

Request: RemoteAddr = "203.0.113.1:8080"
Result: X-Real-IP: 203.0.113.1  (port stripped from RemoteAddr)
```

#### Header Priority with Fallback
```yaml
Configuration:
  processHeaders:
    - headerName: "CF-Connecting-IP"
      depth: -1
    - headerName: "clientAddress"
      depth: -1

Headers:
  CF-Connecting-IP: 198.51.100.1
  (no other headers)

Result: X-Real-IP: 198.51.100.1  (CF-Connecting-IP processed first)
```

#### Force Overwrite (Anti-Spoofing)
```yaml
Configuration:
  forceOverwrite: true
  processHeaders:
    - headerName: "X-Forwarded-For"
      depth: -1

Incoming Request:
  X-Real-IP: "spoofed-value"  # Malicious header
  (no X-Forwarded-For header)

Result: X-Real-IP: ""  (spoofed header overwritten with empty value)
```

#### Trust-Based Security Example
```yaml
Configuration:
  trustAll: false
  trustedIPs: ["192.168.0.0/16", "10.0.0.0/8"]
  trustedHeader: "X-Is-Trusted"
  processHeaders:
    - headerName: "X-Forwarded-For"
      depth: -1
    - headerName: "clientAddress"
      depth: -1

Trusted Request (from 192.168.1.1):
  X-Forwarded-For: "203.0.113.1, 198.51.100.1"
  
Result: 
  X-Real-IP: "203.0.113.1"  (processes X-Forwarded-For)
  X-Is-Trusted: "yes"

Untrusted Request (from 8.8.8.8):
  X-Forwarded-For: "fake-ip, spoofed-ip"
  
Result:
  X-Real-IP: "8.8.8.8"      (ignores headers, uses RemoteAddr)
  X-Is-Trusted: "no"
```

## 📋 Use Cases

### Behind Cloudflare
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders:
            - headerName: "CF-Connecting-IP"
              depth: -1              # Leftmost IP (original client)
            - headerName: "X-Forwarded-For"
              depth: -1              # Fallback to standard header
          forceOverwrite: true
```

### Behind AWS Application Load Balancer
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders:
            - headerName: "X-Forwarded-For"
              depth: -1              # Leftmost IP (original client)
          forceOverwrite: true
```

### Behind NGINX Proxy
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders:
            - headerName: "X-Real-IP"
              depth: -1              # NGINX sets this header
            - headerName: "X-Forwarded-For"
              depth: -1              # Fallback
          forceOverwrite: true
```

### Custom Header Name
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Client-IP"  # Custom header name
          processHeaders:
            - headerName: "X-Forwarded-For"
              depth: -1
            - headerName: "CF-Connecting-IP"
              depth: -1
          forceOverwrite: true
```

### Multiple Proxy Layers with Depth Control
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders:
            - headerName: "CF-Connecting-IP"
              depth: -1              # Trust Cloudflare (leftmost)
            - headerName: "X-Forwarded-For"
              depth: 1               # Skip first IP, use second from right
            - headerName: "clientAddress"  # Synthetic header for direct connection
              depth: -1
          forceOverwrite: true
```

### Direct Connection Fallback (Synthetic Header)
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders:
            - headerName: "X-Forwarded-For"
              depth: -1
            - headerName: "clientAddress"  # Synthetic: maps to request.RemoteAddr
              depth: -1
          forceOverwrite: true
```

### Trust-Based Security (Recommended for Production)
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders:
            - headerName: "X-Forwarded-For"
              depth: -1
            - headerName: "clientAddress"  # Fallback for untrusted sources
              depth: -1
          forceOverwrite: true
          trustAll: false                  # Enable trust checking
          trustedIPs:                      # Only trust these proxy sources
            - "127.0.0.0/8"                # IPv4 loopback
            - "10.0.0.0/8"                 # RFC1918 private
            - "172.16.0.0/12"              # RFC1918 private
            - "192.168.0.0/16"             # RFC1918 private
            - "::1/128"                    # IPv6 loopback
            - "fc00::/7"                   # IPv6 unique local
            - "fe80::/10"                  # IPv6 link-local
          trustedHeader: "X-Is-Trusted"   # Add trust indication header
```

### Behind Cloudflare with Trust Checking
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders:
            - headerName: "CF-Connecting-IP"
              depth: -1
            - headerName: "clientAddress"
              depth: -1
          forceOverwrite: true
          trustAll: false
          trustedIPs:
            - "173.245.48.0/20"          # Cloudflare IP ranges
            - "103.21.244.0/22"          # (example ranges)
            - "103.22.200.0/22"
            - "103.31.4.0/22"
          trustedHeader: "X-Is-Trusted"
```

### Access Logging Configuration
To log the extracted IP addresses in Traefik access logs, configure the access log to include the headers:

```yaml
# traefik.yml or command line flags
accesslog:
  format: json
  fields:
    headers:
      names:
        X-Real-IP: keep          # Log our custom header
        X-Client-IP: keep        # Log custom header name
        CF-Connecting-IP: keep   # Log Cloudflare header
        X-Forwarded-For: keep    # Log standard proxy header
```

The access logs will then include entries like:
```json
{
  "RequestMethod": "GET",
  "RequestPath": "/api/data",
  "request_X-Real-Ip": "203.0.113.1",
  "request_CF-Connecting-IP": "203.0.113.1",
  "DownstreamStatus": 200
}
```

## 🛠️ Advanced Configuration

### Conditional Enabling
You can enable/disable the plugin based on different routes or services:

```yaml
http:
  middlewares:
    realip-enabled:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders: ["X-Forwarded-For"]
    
    realip-disabled:
      plugin:
        realip:
          enabled: false
          headerName: "X-Real-IP"
          processHeaders: ["X-Forwarded-For"]

  routers:
    api:
      rule: "Host(`api.example.com`)"
      middlewares:
        - "realip-enabled"    # Enable for API
    
    static:
      rule: "Host(`static.example.com`)"
      middlewares:
        - "realip-disabled"   # Disable for static content
```

### Header Validation

The plugin performs the following validations:

- **IP Address Format**: Uses Go's `net.ParseIP()` for robust IP validation
- **IPv4 and IPv6 Support**: Handles both IPv4 and IPv6 addresses correctly
- **Port Removal**: Automatically strips port numbers using `net.SplitHostPort()`
- **Whitespace Handling**: Trims whitespace from IP addresses
- **Invalid IP Skipping**: Skips malformed IP addresses and continues to the next

## 🔍 Troubleshooting

### Plugin Not Working
1. Check that the plugin is enabled: `enabled: true`
2. Verify the plugin is loaded in Traefik logs
3. Ensure the middleware is applied to your routes
4. Check that `headerName` and `processHeaders` are configured

### Wrong IP Address Extracted
1. Verify the order of headers in `processHeaders`
2. Check the actual headers sent by your proxy/CDN
3. Use Traefik's access logs to see incoming headers
4. Test with different header configurations

### No IP Address Set
1. Ensure the configured headers are actually present in requests
2. Check if the IP addresses in headers are valid
3. Verify that `headerName` is not empty
4. Test with a simple configuration first

### Debug Configuration
Enable Traefik debug logging to see detailed plugin behavior:

```yaml
log:
  level: DEBUG
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Setup

1. Clone the repository
2. Run tests: `go test -v`
3. Run integration tests: `./Test-Integration.ps1`
4. Make your changes
5. Ensure tests pass
6. Submit a pull request

### Testing

The project includes comprehensive unit tests and integration tests:

- **Unit Tests**: Test individual functions and edge cases
- **Integration Tests**: Test the plugin in a real Traefik environment using Docker Compose

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

This plugin was inspired by the need for a simple, reliable way to extract real client IP addresses in Traefik environments with multiple proxy layers.

---

**Need help?** Open an issue on GitHub or check the [Traefik documentation](https://doc.traefik.io/traefik/plugins/) for more information about plugins.
