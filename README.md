# sfsd - Static File Serving Server

> [!WARNING]
> - This program is currently under development. Please use with caution.

`sfsd` is a lightweight and feature-rich static file serving server written in Go. It is designed to be simple to use while providing essential features for modern web applications.

## Key Features

- **Efficient File Serving**: Serves static files from any specified directory.
- **TLS/SSL Support**: Easily enable HTTPS for secure connections.
- **Middleware Support**:
  - **CORS**: Support for Cross-Origin Resource Sharing.
  - **Compression**: Automatic compression using Gzip, Deflate, Brotli, and Zstd. Intelligently targets compressible content types (like `text/*`, `application/json`, `image/svg+xml`) to preserve `Content-Length` headers for optimal binary download tracking.
  - **Caching**: Flexible caching rules based on file patterns.
  - **Logging**: Access and error logging with configurable formats.
  - **Request Counter**: Track file download statistics individually per file. Creates a detailed `stats.json` map of download counts.
- **Security**:
  - **Basic Authentication**: Simple username/password protection.
  - **Hidden File Management**: Option to hide or show hidden files.
  - **Symlink Control**: Manage internal and external symbolic link access.
- **Custom Error Pages**: Define custom HTML pages for 403, 404, and 500 errors.
- **CLI Interface**: Straightforward commands for launching and configuring the server.

## Installation

### From Source

Ensure you have [Go](https://golang.org/dl/) installed (version 1.25 or later recommended).

```bash
git clone https://github.com/shinys000114/sfsd
cd sfsd
go build -o sfsd main.go
```

## Usage

The `sfsd` binary provides several commands:

### Start the Server

```bash
./sfsd launch config.yaml
```

### Print Version

```bash
./sfsd version
```

### Clean Download Statistics

If you delete files from your serving directory, you can remove those non-existent files from your tracking `stats.json` by running:

```bash
./sfsd clean-stats <serve-directory> <stats-file-path>
# Example
./sfsd clean-stats ./public data/stats.json
```

### Generate Default Configuration

Prints the default configuration in YAML format, which can be redirected to a file:

```bash
./sfsd config > config.yaml
```

## Configuration

Configuration is managed via a `config.yaml` file. Below is an example structure:

```yaml
server:
  host: localhost
  port: 8080
  tls:
    enabled: false
#   cert_file: /path/to/cert.pem
#   key_file: /path/to/key.pem
directory:
  path: ./public # serve directory
  allow_symlink: true
  allow_external_symlink: false
  hide_hidden: true
features:
  cors_enabled: false
  compression:
    - gzip
    - deflate
    - brotli
    - zstd
#   - none
  stats_file: data/stats.json
  auth:
    enabled: false
#   username: admin
#   password: password123
  pages:
#   404: ./public/404.html
  cache_rules:
    - pattern: '*.html'
      max_age: 0
    - pattern: '*.png'
      max_age: 86400
    - pattern: '*'
      max_age: 3600
logging:
  format: plain # json
  access_log: access.log
  error_log: error.log
```

## Project Structure

- `main.go`: Entry point for the application.
- `internal/`: Core logic and internal packages.
  - `config/`: Configuration loading and structure.
  - `handler/`: HTTP handlers for file serving and directory listing.
  - `middleware/`: HTTP middleware components.
  - `server/`: Server initialization and lifecycle.
- `example/`: Example configuration and error pages.

