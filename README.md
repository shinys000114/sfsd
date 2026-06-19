# sfsd - Static File Serving Server

> [!WARNING]
> - This program is currently under development. Please use with caution.

`sfsd` is a lightweight, high-performance, and feature-rich static file serving server written in Go. It is designed to be simple to use while providing modern web features like HTTP/3 and Virtual Hosting.

## Key Features

- **Efficient File Serving**: High-speed static file delivery from any directory.
- **Modern Protocols**:
  - **HTTP/2**: Automatic support with TLS.
  - **HTTP/3 (QUIC)**: Low-latency, modern protocol support for faster connections.
- **Virtual Hosting (vHost)**: Serve multiple domains on a single IP and port with independent configurations and SSL certificates (SNI).
- **Multi-Server Instance**: Run multiple independent servers (different ports or hosts) within a single process.
- **Advanced Caching**: 
  - Full support for **ETag**, **Last-Modified**, and **304 Not Modified** for efficient conditional requests.
  - Flexible **Cache-Control** rules based on file patterns.
- **Middleware Support**:
  - **CORS**: Support for Cross-Origin Resource Sharing.
  - **Compression**: Automatic compression using Gzip, Deflate, Brotli, and Zstd. Intelligently targets compressible content types.
  - **Logging**: Access and error logging with configurable formats (Plain/JSON).
  - **Request Counter**: Track file download statistics individually per file in `stats.json`.
- **Security**:
  - **Basic Authentication**: Simple username/password protection.
  - **Hidden File Management**: Option to hide or show hidden files.
  - **Symlink Control**: Manage internal and external symbolic link access.
- **Custom Error Pages**: Define custom HTML pages for 403, 404, and 500 errors.
- **CLI Interface**: Straightforward commands for launching and configuring.

## Installation

### From Source

Ensure you have [Go](https://golang.org/dl/) installed (version 1.25 or later recommended).

```bash
git clone https://github.com/shinys000114/sfsd
cd sfsd
go build -o sfsd main.go
```

## Usage

### Start the Server

```bash
./sfsd launch config.yaml
```

### Generate Self-Signed Certificate

Generates a self-signed TLS certificate and private key for testing.
**Warning: These certificates are for development/testing purposes only.**

```bash
# Default (RSA 2048)
./sfsd gen-cert ./certs

# Specific Algorithm (rsa4096, ecdsa, ed25519)
./sfsd gen-cert ./certs rsa4096
./sfsd gen-cert ./certs ed25519
```

### Print Version

```bash
./sfsd version
```

### Clean Download Statistics

```bash
./sfsd clean-stats <serve-directory> <stats-file-path>
```

### Generate Default Configuration

```bash
./sfsd config > config.yaml
```

## Configuration

Configuration is managed via a `config.yaml` file. `sfsd` supports multiple server instances in one file.

```yaml
# Instance 1: Standard Web Server
web-server:
  server:
    host: 0.0.0.0
    port: 8080
    domains: ["example.com", "www.example.com"]
  directory:
    path: ./public
    allow_symlink: true
    allow_external_symlink: false
    hide_hidden: true
    render_readme_md: true
    exclude:
      - .git/
      - '*.tmp'
      - private/
  features:
    compression: ["gzip", "brotli"]
    stats_file: data/stats-web.json
    cache_rules:
      - pattern: '*.png'
        max_age: 86400
      - pattern: 'assets/**'
        max_age: 31536000

# Instance 2: Secure Storage (HTTPS/HTTP3)
secure-storage:
  server:
    host: 0.0.0.0
    port: 8443
    tls:
      enabled: true
      http3: true
      cert_file: /path/to/cert.pem
      key_file: /path/to/key.pem
  directory:
    path: ./storage
  features:
    auth:
      enabled: true
      username: admin
      password: password123
```

`directory.exclude` uses gitignore-like patterns. Blank lines and `#` comments are ignored, `!` negates a previous match, `/` anchors a pattern to the serve root, and a trailing `/` matches a directory and its contents.

`directory.render_readme_md` renders a `README.md` file at the top of directory listings. Excluded or hidden README files are not rendered.

`directory.hide_hidden` checks every path component, so files inside hidden directories such as `.git/config` are hidden too.

`directory.allow_external_symlink` controls whether symlink targets outside the served directory are accessible. If disabled, external symlinks are rejected.

`features.cache_rules[].pattern` uses glob patterns. Patterns without `/`, such as `*.png`, match any path component. Patterns with `/`, such as `assets/**`, match the request path relative to the served directory.

Each server instance should normally use its own `features.stats_file`. Instances that share the same `stats_file` also share one download counter.

For more detailed scenarios (Reverse Proxy, vHost, etc.), check the [**Example Directory**](./example/README.md).

## Project Structure

- `main.go`: Entry point for the application.
- `internal/`: Core logic and internal packages.
  - `config/`: Configuration loading and structure.
  - `handler/`: HTTP handlers for file serving and directory listing.
  - `middleware/`: HTTP middleware components.
  - `server/`: Server initialization and lifecycle.
- `example/`: Example configuration and reverse proxy guides.

## *
While running multiple repositories, I was looking for a lightweight file server to replace nginx and caddy. So, I built my own.

I am using this program here.
 - [https://storage.sys114.com/](https://storage.sys114.com/): Monthly Odroid Image Build
