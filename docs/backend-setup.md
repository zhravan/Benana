# Backend Setup (Go core + Plugins)

This guide explains how to run the Go backend, wire up PostgreSQL, and develop/load dynamic plugins.

## Prerequisites

- Go 1.24.x installed and matching the host OS/arch
- pnpm and Node.js 20+ (for the frontend, optional here)
- Docker (optional, for PostgreSQL via `docker-compose`)

Notes on Go plugins:
- Go plugin `.so` builds only work on Unix-like systems (Linux/macOS) and must be built and loaded on the same OS/arch with the same Go toolchain version.
- `CGO_ENABLED=1` is used when building plugins.

## Repository Layout

```
plugins/
  core/          # Backend server (Go) with plugin manager
  types/         # Shared types, DB models, migrations, config
  powertable/    # Example dynamic plugin implementing PluginInterface
bin/
  core           # Built server binary (via Makefile)
```

`go.work` ties together multiple Go modules under `plugins/`.

## Database

Use the included `docker-compose.yml` to run PostgreSQL with sensible defaults.

### Start PostgreSQL

```
docker compose up -d postgres
```

Defaults (work both for container and local host connections):
- DB: `benana`
- User: `benana_user`
- Password: `benana_password`
- Port: `5432`

These values are also the defaults used by the backend if no config is provided.

### Configuration options

The backend reads config using Viper, with three sources (in priority order):
- Environment variables: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `config.yaml` in current working directory or `./config/config.yaml`
- Built-in defaults (match the Docker values above)

An example YAML exists at `plugins/types/config.yaml`. If you want to run the server with a file config, copy it near your working directory:

```
cp plugins/types/config.yaml plugins/core/config/config.yaml
```

## Build and Run the Backend

There are two supported ways to run the server: via Makefile from repo root or directly from the `plugins/core` module. Because the plugin discovery paths are currently relative (see Important Notes), the recommended way for development is to run from within `plugins/core`.

### Option A: Makefile (build) + run

From repo root:

```
make build             # builds ./bin/core by running `go build` in plugins/core
cd plugins/core        # IMPORTANT: run from this directory for correct plugin discovery paths
../../bin/core         # start the server
```

### Option B: Run directly from module

```
cd plugins/core
go run .
```

On startup, the server:
- Connects to PostgreSQL and runs migrations from `plugins/types/migrations` (tables + LISTEN/NOTIFY triggers)
- Discovers plugins in the `../` folder (i.e., `plugins/`), validates, compiles, registers, and loads them
- Exposes its own routes and any routes registered by dynamic plugins

## Verify it’s working

With the example plugin `powertable` present:

```
curl http://localhost:9999/plugins
curl http://localhost:9999/plugins/powertable
curl http://localhost:9999/hello           # route registered by powertable
```

Ports: Fuego’s default dev port is `9999` unless changed via server options.

## Developing a New Plugin

1) Create a directory under `plugins/` (peer of `core`/`types`), e.g. `plugins/myplugin`

2) Initialize a Go module that depends on `benana/types` and Fuego. Example `go.mod`:

```
module myplugin

go 1.24.2

replace benana/types => ../types

require (
  benana/types v0.0.0-00010101000000-000000000000
  github.com/go-fuego/fuego v0.18.8
)
```

3) Implement the required factory and interface in `main.go` or `plugin.go`:

```
package main

import (
  "context"
  "benana/types"
  "github.com/go-fuego/fuego"
)

type MyPlugin struct { metadata types.PluginMetadata }

func NewPlugin() types.PluginInterface { return &MyPlugin{ metadata: types.PluginMetadata{Name: "myplugin", Version: "0.1.0"} } }
func (p *MyPlugin) Initialize(ctx context.Context) error { return nil }
func (p *MyPlugin) GetMetadata() types.PluginMetadata { return p.metadata }
func (p *MyPlugin) RegisterRoutes(s *fuego.Server) error { fuego.Get(s, "/myplugin", p.hello); return nil }
func (p *MyPlugin) Cleanup(ctx context.Context) error { return nil }
func (p *MyPlugin) hello(c fuego.ContextNoBody) (interface{}, error) { return map[string]string{"ok":"true"}, nil }
```

4) Start the server from `plugins/core` as described above. The server will:
- Auto-discover your new directory
- Validate it (checks for `main.go` or `plugin.go`)
- Compile to a `.so` under `plugins/core/bin/plugins`
- Register it in DB (if new) and load its routes

## Available HTTP Endpoints

- `GET /plugins`: list registered plugins (active)
- `GET /plugins/{name}`: details for a single plugin
- Plus any routes exported by loaded plugins (e.g., Powertable adds `GET /hello`)

## Important Notes and Limitations

- Working directory matters today:
  - `plugins/core/main.go` uses relative constants: `pluginsDir = "../"` and `binariesDir = "./bin/plugins"`.
  - Run the binary with CWD = `plugins/core` for discovery to read `../` (the `plugins/` folder).
- Go plugin constraints:
  - Build and run on the same GOOS/GOARCH and Go toolchain. Cross-compiling `.so` is not supported.
  - Requires `CGO_ENABLED=1`.
- Config:
  - If no `config.yaml` or env vars are provided, defaults connect to the local Postgres from `docker-compose.yml`.

