# Plugins Architecture (Detailed)

This document explains how the Go plugin system is organized, the lifecycle of a plugin from discovery to runtime routing, and the data/storage model.

## High-Level Overview

```
             +-----------------+            +-------------------+
HTTP Client ->|  Fuego Server   |<---------->|   Plugin Manager  |
             +--------+--------+            +---------+---------+
                      ^                              ^
                      | routes                        | loads/compiles
                      |                              |
               +------+--------+            +--------+--------+
               |  Loader       |            |   Compiler      |
               +------+--------+            +--------+--------+
                      ^                              ^
                      | .so                          | source
                      |                              |
               +------+--------+            +--------+--------+
               |    Plugin .so  |<----------|   Plugin Source |
               +----------------+            +-----------------+

               +--------------------+
               | Discovery + Events |  LISTEN/NOTIFY
               +--------+-----------+        ^
                        |                    |
                        v                    |
                   +----+-----+              |
                   |  Repo    |--------------+
                   +----+-----+
                        |
                        v
                   PostgreSQL
```

Core subsystems:
- Plugin Manager (`plugins/core/manager.go`): Orchestrates compile, load, register routes, and reacts to DB changes.
- Compiler (`plugins/core/compiler.go`): Validates structure and builds `.so` using `go build -buildmode=plugin`.
- Loader (`plugins/core/loader.go`): Loads `.so`, looks up `NewPlugin`, creates an instance, initializes, and registers routes.
- Discovery + Events (`plugins/core/service.go`): Exposes querying via repository and listens to PostgreSQL `LISTEN plugin_changes` notifications to react to INSERT/UPDATE/DELETE.
- Repository (`plugins/core/repository.go`): CRUD for plugins and endpoints via Bun ORM.
- Controller (`plugins/core/controller.go`): Exposes `GET /plugins` and `GET /plugins/{name}`.

Shared types, DB models, migrations, and config live in `plugins/types`.

## Plugin Interface and Contract

Dynamic plugins must export a factory `NewPlugin() types.PluginInterface` and implement:
- `Initialize(ctx context.Context) error`
- `GetMetadata() types.PluginMetadata`
- `RegisterRoutes(server *fuego.Server) error`
- `Cleanup(ctx context.Context) error`

Metadata includes name, version, description, author, and an optional list of endpoints.

Example plugin: `plugins/powertable/main.go`.

## Lifecycle

1) Server start (`plugins/core/main.go`)
- Connect DB and run migrations.
- Initialize Discovery and Plugin Manager.
- Discover new plugins under `../` (i.e., the `plugins/` directory) and register them if not in DB.
- Load active plugins, compiling dynamic ones as needed.
- Start event listener for DB changes.

2) Discovery (`compiler.DiscoverPlugins`)
- Scans `pluginsDir` for directories (skips `core`, `types`, `.git`).
- Validates presence of `main.go` or `plugin.go`.
- Proposes as `dynamic` plugin with `SourcePath`.

3) Validation & Compilation (`compiler.ValidatePluginInterface`)
- Compiles the candidate plugin to `.so` in `./bin/plugins`.
- Loads it via Loader to ensure `NewPlugin` exists and returns a valid instance with non-empty metadata.
- Unloads the temporary instance; on success, registration proceeds.

4) Registration (`discovery.AddNewPlugin`)
- Writes plugin and endpoints to PostgreSQL tables.
- Status defaults to `active`.

5) Loading (`pluginLoader.LoadPlugin` + `RegisterPluginRoutes`)
- `plugin.Open` then `Lookup("NewPlugin")` to obtain the factory.
- Create instance, `Initialize`, capture metadata, and `RegisterRoutes(s)` on the Fuego server.
- Keep the plugin in an in-memory map for later reference/unloading.

6) Event-driven updates (`Discovery.StartPluginChangeListener`)
- `LISTEN` on `plugin_changes` channel; triggers are installed by migrations.
- On INSERT/UPDATE/DELETE, payload JSON is deserialized and passed back to Plugin Manager.
- Manager loads/reloads/unloads affected plugins and updates routes accordingly.

## Data Model and Migrations

Tables (from `plugins/types/migrations/*`):
- `plugins`: core metadata, type (`dynamic`/`static`), source/binary paths, status, timestamps
- `plugin_endpoints`: per-plugin HTTP endpoints

Triggers: `notify_plugin_change()` emits JSON payloads on INSERT/UPDATE/DELETE to `plugin_changes` channel, enabling live updates without restarts.

## Routing

Base endpoints provided by core:
- `GET /plugins`
- `GET /plugins/{name}`

Plugins add their own routes via `RegisterRoutes`. For example, Powertable registers `GET /hello`.

## Build, Paths, and Constraints

- Paths are currently relative constants (see `plugins/core/main.go`):
  - `pluginsDir = "../"` (scan parent to find plugin directories)
  - `binariesDir = "./bin/plugins"` (where `.so` files are written)
- For correct behavior, run the server binary with the working directory set to `plugins/core`.
- Plugin `.so` must be built by the same Go version and on the same OS/arch where it’s loaded.

## Example Flow (Powertable)

1) Start DB via Docker and run the server from `plugins/core`.
2) Server discovers `../powertable`, validates, compiles to `./bin/plugins/powertable.so`.
3) Registers powertable in DB (if not present) as `active` and loads it.
4) Powertable registers `GET /hello` on the server.
5) `GET /plugins` lists powertable; `GET /hello` returns its response.

## Future Enhancements

- Move `pluginsDir`/`binariesDir` into config/env to remove CWD sensitivity.
- Add admin endpoints to toggle, reload, or remove plugins via HTTP.
- Support hot recompilation on source changes and richer health checks.

