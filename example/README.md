# sfsd Examples & Scenarios

This directory contains various configuration examples for `sfsd`. You can use these as templates to set up your own static file server.

## Launching sfsd

`sfsd` supports multiple server instances within a single configuration file. Launch the server using:
```bash
./sfsd launch <config-file.yaml>
```

---

## 1. sfsd Configuration Scenarios

| Config File | Description | Key Features |
| :--- | :--- | :--- |
| [`config-simple.yaml`](./config-simple.yaml) | Basic HTTP Server | Single port (8080), Gzip/Brotli compression. |
| [`config-tls-h3.yaml`](./config-tls-h3.yaml) | Secure Server (HTTPS/HTTP3) | TLS enabled, HTTP/3 (QUIC) support, Custom error pages. |
| [`config-vhost.yaml`](./config-vhost.yaml) | Virtual Hosting (vHost) | Serve different directories based on the requested domain on the same port (80). |
| [`config-multi-instance.yaml`](./config-multi-instance.yaml) | Multi-Instance Setup | Multiple server instances running on different ports (e.g., 8080 and 9090). |

---

## 2. Reverse Proxy Scenarios

Integrating `sfsd` with external reverse proxies like Nginx or Caddy provides enhanced security, automatic SSL (Let's Encrypt), and load balancing.

### Caddy Scenarios
Caddy is known for its simplicity and automatic HTTPS/HTTP3 support.

- **[Standard SSL Termination](./reverse_proxy/caddy-standard.caddy)**: Front-end HTTPS (Caddy) to Back-end HTTP (sfsd).
- **[vHost Integration](./reverse_proxy/caddy-vhost.caddy)**: Routing multiple domains to a single sfsd backend port.
- **[End-to-End TLS](./reverse_proxy/caddy-tls-reencrypt.caddy)**: Encrypted communication between Caddy and sfsd.

### Nginx Scenarios
Nginx offers fine-grained control over protocols, caching, and upstream management.

- **[Standard SSL & HTTP/3](./reverse_proxy/nginx-standard.conf)**: Termination of HTTPS/HTTP3 with optimized proxy headers.
- **[vHost Support](./reverse_proxy/nginx-vhost.conf)**: Handling multiple domains on a shared backend port using the `Host` header.
- **[TLS Re-encryption](./reverse_proxy/nginx-tls-reencrypt.conf)**: Proxying requests to an sfsd instance that already has TLS enabled.
- **[Load Balancing](./reverse_proxy/nginx-load-balance.conf)**: Distributing traffic across a cluster of sfsd instances.

---

## 3. Important Notes on vHost & HTTP/3

### Virtual Hosting (vHost)
To use virtual hosting effectively behind a reverse proxy, you **must** ensure the proxy forwards the `Host` header.
- **Nginx**: `proxy_set_header Host $host;`
- **Caddy**: (Automatic by default)

### HTTP/3 (QUIC)
When running `sfsd` with HTTP/3 directly:
- Ensure the UDP port (same as TCP port) is open in your firewall.
- Browsers require the `Alt-Svc` header (automatically added by `sfsd` or proxy examples) to discover the HTTP/3 endpoint.

---

## 4. Systemd Service Files

For production environments, use systemd to manage the `sfsd` process:

- [`sfsd.service`](./services/sfsd.service): System-wide service using `/etc/sfsd/config.yaml`.
- [`sfsd-user.service`](./services/sfsd-user.service): User-level service using `~/sfsd/config.yaml`.

**Installation:**
```bash
cp example/services/sfsd-user.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now sfsd-user.service
```
