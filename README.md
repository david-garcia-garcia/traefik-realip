# 🌐 Traefik RealIP Plugin

[![Build Status](https://github.com/david-garcia-garcia/traefik-realip/actions/workflows/ci.yml/badge.svg)](https://github.com/david-garcia-garcia/traefik-realip/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/david-garcia-garcia/traefik-realip)](https://goreportcard.com/report/github.com/david-garcia-garcia/traefik-realip)
[![Latest GitHub release](https://img.shields.io/github/v/release/david-garcia-garcia/traefik-realip?sort=semver)](https://github.com/david-garcia-garcia/traefik-realip/releases/latest)
[![License](https://img.shields.io/badge/license-Apache%202.0-brightgreen.svg)](LICENSE)

A Traefik plugin that extracts the real client IP address from proxy headers and populates a specified request header.

## ✨ Features

- **Configurable header processing**: Define which headers to check for IP addresses and in what order
- **Flexible output**: Set any custom header name for the extracted IP address
- **Smart IP extraction**: Handles multiple IPs in headers, port numbers, and IPv6 addresses
- **Header priority**: Processes headers in the configured order, using the first valid IP found
- **Port stripping**: Automatically removes port numbers from IP addresses
- **IPv6 support**: Full support for IPv6 addresses including bracketed notation with ports
- **Enable/disable control**: Easy on/off switch for the plugin functionality
- **Robust validation**: Validates IP addresses and handles malformed input gracefully

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

# Run integration tests (Bash)
chmod +x Test-Integration.ps1
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
            - "X-Forwarded-For"
            - "CF-Connecting-IP"
            - "X-Real-IP"
          clientAddrFallback: false        # Fallback to request.RemoteAddr if no IP found
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable or disable the plugin |
| `headerName` | string | `"X-Real-IP"` | Name of the header to populate with the extracted IP |
| `processHeaders` | array | `["X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"]` | List of headers to check for IP addresses (processed in order) |
| `clientAddrFallback` | boolean | `false` | Fallback to request.RemoteAddr if no IP found in configured headers |

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
      - "traefik.http.middlewares.realip.plugin.realip.processHeaders=X-Forwarded-For,CF-Connecting-IP"
```

## 🔄 How It Works

The plugin processes requests in the following order:

1. **Check if enabled**: If `enabled` is `false`, the plugin passes the request through unchanged
2. **Process headers in order**: Iterates through the `processHeaders` list in the specified order
3. **Extract IPs from headers**: For each header:
   - Splits comma-separated IP addresses
   - Processes IPs from left to right (leftmost IP is typically the original client)
   - Cleans each IP (removes whitespace and port numbers)
   - Validates the IP address format
4. **Set the result**: Populates the `headerName` header with the first valid IP address found
5. **Forward the request**: Passes the modified request to the next handler

### Header Processing Examples

#### Single IP Address
```
X-Forwarded-For: 203.0.113.1
Result: X-Real-IP: 203.0.113.1
```

#### Multiple IP Addresses (comma-separated)
```
X-Forwarded-For: 203.0.113.1, 198.51.100.1, 192.168.1.1
Result: X-Real-IP: 203.0.113.1  (first valid IP)
```

#### IP Address with Port
```
X-Forwarded-For: 203.0.113.1:8080
Result: X-Real-IP: 203.0.113.1  (port stripped)
```

#### IPv6 Address
```
X-Forwarded-For: 2001:db8::1
Result: X-Real-IP: 2001:db8::1
```

#### IPv6 Address with Port
```
X-Forwarded-For: [2001:db8::1]:8080
Result: X-Real-IP: 2001:db8::1  (port stripped)
```

#### Header Priority
```
Headers:
  X-Forwarded-For: 203.0.113.1
  CF-Connecting-IP: 198.51.100.1

Configuration:
  processHeaders: ["X-Forwarded-For", "CF-Connecting-IP"]

Result: X-Real-IP: 203.0.113.1  (X-Forwarded-For processed first)
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
            - "CF-Connecting-IP"
            - "X-Forwarded-For"
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
            - "X-Forwarded-For"
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
            - "X-Real-IP"
            - "X-Forwarded-For"
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
            - "X-Forwarded-For"
            - "CF-Connecting-IP"
```

### Multiple Proxy Layers
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders:
            - "X-Forwarded-For"      # Check first (closest to client)
            - "X-Real-IP"            # Fallback
            - "CF-Connecting-IP"     # Cloudflare
            - "X-Client-IP"          # Custom proxy
```

### Client Address Fallback
```yaml
http:
  middlewares:
    realip:
      plugin:
        realip:
          enabled: true
          headerName: "X-Real-IP"
          processHeaders:
            - "X-Forwarded-For"
            - "CF-Connecting-IP"
          clientAddrFallback: true   # Use request.RemoteAddr if no headers found
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
