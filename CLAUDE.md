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

# Generate tokens for management
./redirect_helper -reset-admin-token      # For admin operations
./redirect_helper -reset-redirect-token   # For redirect management
./redirect_helper -reset-domain-token     # For domain management

# Update/create redirects and domains
./redirect_helper -update <name> -target <url>         # Create/update redirect
./redirect_helper -update-domain <domain> -target <url> # Create/update domain

# List and remove entries
./redirect_helper -list              # List all redirects
./redirect_helper -list-domains      # List all domains
./redirect_helper -remove <name>     # Remove redirect
./redirect_helper -remove-domain <domain> # Remove domain

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
- Global tokens for operations: `admin_token`, `redirect_token`, `domain_token` (32-character hex)
- Admin token for management APIs (list, remove)
- Redirect token for creating/updating path redirects
- Domain token for creating/updating domain redirects
- Token validation at both application and storage layers

## Configuration Structure

The application uses a single JSON configuration file (`redirect_helper.json`) with three main sections:

```json
{
  "forwardings": {
    "name": {
      "name": "string",
      "target": "url-or-host:port",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  },
  "domains": {
    "domain.com": {
      "domain": "string",
      "target": "full-url",
      "created_at": "timestamp", 
      "updated_at": "timestamp"
    }
  },
  "server": {
    "port": "string",
    "admin_token": "32-char-hex",
    "redirect_token": "32-char-hex",
    "domain_token": "32-char-hex",
    "max_redirect_count": 20,
    "max_domain_count": 10
  }
}
```

## API Endpoints

### Management APIs (require admin_token)
- `GET /api/list?admin_token=<token>` - List all forwardings
- `DELETE /api/remove?name=<name>&admin_token=<token>` - Remove forwarding
- `GET /api/list-domains?admin_token=<token>` - List all domains
- `DELETE /api/remove-domain?domain=<domain>&admin_token=<token>` - Remove domain

### Update APIs (require specific tokens)
- `GET /api/update?name=<name>&token=<redirect_token>&target=<target>` - Create/update forwarding target
- `GET /api/update-domain?domain=<domain>&token=<domain_token>&target=<target>` - Create/update domain target

### Batch Update APIs (require specific tokens)
- `GET /api/batch-update` - Batch update with indexed parameters
- `POST /api/batch-update` - Batch update with JSON body

#### Batch Update - GET Method (Indexed Parameters)
Use indexed parameters (name1, target1, name2, target2, etc.) for simple batch updates:

```bash
# Batch update path redirects
GET /api/batch-update?redirect_token=<token>&name1=test1&target1=google.com:443&name2=test2&target2=baidu.com:443

# Batch update domain redirects
GET /api/batch-update?domain_token=<token>&domain1=d1.example.com&target1=https://google.com&domain2=d2.example.com&target2=https://baidu.com

# Mixed batch update (paths and domains)
GET /api/batch-update?redirect_token=<r_token>&domain_token=<d_token>&name1=test1&target1=google.com:443&domain2=d.example.com&target2=https://github.com
```

#### Batch Update - POST Method (JSON Body)
Use JSON body for more flexible and structured batch updates:

```bash
POST /api/batch-update
Content-Type: application/json

{
  "redirect_token": "your_redirect_token",
  "domain_token": "your_domain_token",
  "entries": [
    {"name": "test1", "target": "google.com:443"},
    {"name": "test2", "target": "baidu.com:443"},
    {"domain": "d.example.com", "target": "https://github.com"}
  ]
}
```

**Response Format**:
```json
{
  "state": "success",
  "message": "All entries updated successfully",
  "results": [
    {
      "name": "test1",
      "target": "google.com:443",
      "success": true
    }
  ],
  "summary": {
    "total": 3,
    "succeeded": 3,
    "failed": 0
  }
}
```

**Response States**:
- `success` - All entries updated successfully
- `partial` - Some entries succeeded, some failed
- `error` - All entries failed to update

### Redirect Endpoints
- `GET /go/<name>` - Redirect to forwarding target
- `GET /*` - Domain-based redirect (any path, preserves full URL)

## Interactive Menu System

When running in server mode, the application provides an interactive menu system:

```
Main Menu:
1. Settings (view config, change port, reset tokens, manage limits)
2. Forwardings (list, update/create, remove)
3. Domains (list, update/create, remove)
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

- **Global Tokens**: Three global 32-character hex tokens for different operations
  - **Admin Token**: For management operations (list, remove)
  - **Redirect Token**: For creating/updating path redirects
  - **Domain Token**: For creating/updating domain redirects
- **Token Generation**: Uses crypto/rand for secure token generation via `utils.GenerateToken()`
- **Limits**: Configurable maximum counts (default: 20 redirects, 10 domains)

## Hot Configuration Reload

Most configuration changes take effect immediately:
- Adding/removing forwardings and domains
- Updating targets
- Token changes (admin, redirect, domain)
- Limit configuration changes

**Requires Restart**:
- Port changes (server must be restarted to bind to new port)

## Common Development Tasks

When working on this codebase:

1. **Adding New API Endpoints**: Add routes in `server.go` `setupRoutes()` and implement handlers
2. **Modifying Storage**: Update interfaces in `storage/interface.go` and implement in `storage/config.go`
3. **Configuration Changes**: Update structs in `config/config.go` and add corresponding methods
4. **Interactive Menu**: Add menu options in `main.go` interactive functions

## Configuration Management

### Auto-initialization (Server Mode)
When running in server mode, configuration is automatically created with tokens:
```bash
# First time server start - auto-creates config with tokens
./redirect_helper -server
# Output: Shows generated admin, redirect, and domain tokens

# Custom config path - auto-creates directory structure
./redirect_helper -config /path/to/app.json -server
```

### Manual Token Management (Requires Existing Config)
```bash
# These commands require configuration file to exist
./redirect_helper -reset-admin-token      # Reset admin token
./redirect_helper -reset-redirect-token   # Reset redirect token  
./redirect_helper -reset-domain-token     # Reset domain token
```

## Testing the Application

Quick start (server mode auto-creates everything):
```bash
# Start server (auto-creates config and tokens)
./redirect_helper -server

# In another terminal, create/update entries
./redirect_helper -update test -target google.com
./redirect_helper -update-domain test.example.com -target https://google.com

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