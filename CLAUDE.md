# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Redirect Helper is a Go-based URL redirection service that supports two modes:
1. **Path-based redirects**: Traditional `/go/name` URL redirects
2. **Domain-based redirects**: Host header-based redirects with full URL preservation

The application features token-based authentication, JSON configuration persistence, and both CLI and interactive server management interfaces.

## Build and Development Commands

```bash
# Build the application
go build -o redirect_helper ./cmd/redirect_helper

# Run the server (default port 8001)
./redirect_helper -server

# Run with custom port
./redirect_helper -server -port 9090

# Run with custom config file
./redirect_helper -server -config /path/to/config.json

# Run interactive server mode (server + interactive menu)
./redirect_helper -server
```

## Architecture Overview

### Core Components

1. **Main CLI (`cmd/redirect_helper/main.go`)**: Entry point with comprehensive flag parsing, interactive menu system, and background server management
2. **HTTP Server (`internal/server/server.go`)**: HTTP request handling, API routing, and domain-based redirect logic
3. **Configuration (`internal/config/config.go`)**: JSON-based configuration management with persistent storage
4. **Storage Layer (`internal/storage/`)**: Unified data access with ConfigStorage pattern implementing multiple interfaces
5. **Models (`internal/models/forwarding.go`)**: Data structures for forwardings and domains

### Key Architectural Patterns

**Dual Operating Modes**:
- **CLI Mode**: Direct command execution for management operations
- **Interactive Server Mode**: Background HTTP server with foreground interactive menu system

**Storage Architecture**:
- `ConfigStorage` implements `Storage`, `ExtendedStorage`, and `DomainStorage` interfaces
- Single JSON file persistence with atomic updates
- Configuration hot-reload for most changes (port changes require restart)

**Request Routing Priority**:
1. Domain-based redirects (checked first via `checkDomainRedirect`)
2. Path-based redirects (`/go/name`)
3. API endpoints (`/api/*`)
4. Homepage (`/`)

**Security Model**:
- Individual tokens for each forwarding/domain (32-character hex)
- Global admin token for management APIs
- Token validation at both application and storage layers

## Configuration Structure

The application uses a single JSON configuration file (`redirect_helper.json`) with three main sections:

```json
{
  "forwardings": {
    "name": {
      "name": "string",
      "token": "32-char-hex", 
      "target": "url-or-host:port",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  },
  "domains": {
    "domain.com": {
      "domain": "string",
      "token": "32-char-hex",
      "target": "full-url",
      "created_at": "timestamp", 
      "updated_at": "timestamp"
    }
  },
  "server": {
    "port": "string",
    "admin_token": "32-char-hex"
  }
}
```

## API Endpoints

### Management APIs (require admin_token)
- `GET /api/list?admin_token=<token>` - List all forwardings
- `POST /api/create?name=<name>&admin_token=<token>` - Create forwarding
- `DELETE /api/remove?name=<name>&admin_token=<token>` - Remove forwarding
- `GET /api/list-domains?admin_token=<token>` - List all domains
- `POST /api/create-domain?domain=<domain>&admin_token=<token>` - Create domain
- `DELETE /api/remove-domain?domain=<domain>&admin_token=<token>` - Remove domain

### Target Setting APIs (require individual tokens)
- `GET /api/set?name=<name>&token=<token>&target=<target>` - Set forwarding target
- `GET /api/set-domain?domain=<domain>&token=<token>&target=<target>` - Set domain target

### Redirect Endpoints
- `GET /go/<name>` - Redirect to forwarding target
- `GET /*` - Domain-based redirect (any path, preserves full URL)

## Interactive Menu System

When running in server mode, the application provides an interactive menu system:

```
Main Menu:
1. Settings (view config, change port, reset admin token)
2. Forwardings (list, create, update, remove)
3. Domains (list, create, update, remove)
```

Navigation: Use numbers to select, 'b' to go back, 'q' to quit.

## Domain vs Path Redirects

**Path Redirects** (`/go/name`):
- Simple URL redirection
- Path and query parameters are lost
- Suitable for basic link shortening

**Domain Redirects** (Host header based):
- Full URL preservation including path and query parameters
- Supports all HTTP methods (GET, POST, PUT, DELETE, etc.)
- Uses HTTP 302 redirects (no server bandwidth consumption)
- Ideal for proxying domains through the service

## Token Management

- **Individual Tokens**: Each forwarding/domain has a unique 32-character hex token
- **Admin Token**: Global token for management operations, can be reset via CLI or interactive menu
- **Token Generation**: Uses crypto/rand for secure token generation via `utils.GenerateToken()`

## Hot Configuration Reload

Most configuration changes take effect immediately:
- Adding/removing forwardings and domains
- Updating targets
- Admin token changes

**Requires Restart**:
- Port changes (server must be restarted to bind to new port)

## Common Development Tasks

When working on this codebase:

1. **Adding New API Endpoints**: Add routes in `server.go` `setupRoutes()` and implement handlers
2. **Modifying Storage**: Update interfaces in `storage/interface.go` and implement in `storage/config.go`
3. **Configuration Changes**: Update structs in `config/config.go` and add corresponding methods
4. **Interactive Menu**: Add menu options in `main.go` interactive functions

## Testing the Application

Create test entries:
```bash
# Create a forwarding
./redirect_helper -create test

# Create a domain
./redirect_helper -create-domain test.example.com

# List entries
./redirect_helper -list
./redirect_helper -list-domains
```

Test redirects:
```bash
# Test path redirect
curl -I "http://localhost:8001/go/test"

# Test domain redirect (requires proper Host header)
curl -I -H "Host: test.example.com" "http://localhost:8001/any/path"
```