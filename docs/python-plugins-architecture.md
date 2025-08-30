# Python Plugin Architecture (FastAPI + Poetry)

This document outlines the new Python-based plugin architecture replacing the Go setup. It focuses on simplicity, safe hot-loading, and per-plugin owned migrations, while using Poetry for dependency management.

## Requirements Mapping

- Poetry only: Use Poetry for dependency and environment management.
- FastAPI structure: Host runtime is a FastAPI app served by Uvicorn.
- Dependencies: `fastapi`, `uvicorn[standard]`, `pydantic-settings`, `sqlalchemy`, `psycopg[binary]`, `python-multipart` (for uploads), optional `orjson`.
- Plugin architecture: Load/unload plugins at runtime via API; register routes dynamically.
- Plugin communication: In-process service registry/bus for typed calls and pub/sub.
- PostgreSQL: Primary persistence via psycopg/SQLAlchemy.
- Migrations: Per-plugin SQL migrations, tracked by host (`plugin_migrations` table).
- Host runtime: API server loads plugins and exposes routes/RPC.
- Stable SPI: Plugins implement a Python interface: registration, migrations, routes/handlers, permissions, seed data.
- Two sample plugins: Provided for validation and tests.
- Per-plugin migrations: Each plugin ships `migrations/*.sql` owned by the plugin.
- Makefile: Convenience targets for dev/build/run/migrate.
- Runtime add/remove: API to install (upload/unpack), enable/disable, reload, and remove non-core plugins; core plugins cannot be removed.

## High-Level Architecture

- FastAPI Host
  - Starts DB connection pool.
  - Runs host bootstrap migrations (ensures `plugins` and `plugin_migrations` tables exist).
  - Discovers and loads core plugins on startup.
  - Exposes plugin admin API for listing, installing, enabling/disabling, reloading, and removing plugins.
  - Registers plugin routes at runtime on the FastAPI app/router.
- Plugin Manager
  - Scans `plugins/` for available plugins (directory convention; per-plugin `plugin.toml` is optional metadata).
  - Validates SPI implementation by dynamic import and attribute checks.
  - Applies pending plugin migrations in order (filename ordering) and records checksums in `plugin_migrations`.
  - Loads plugin: instantiate, `on_load`, `register_routes`, `register_permissions`, optional `seed`.
  - Unloads plugin: `on_unload`, deregister routes/services, detach module.
- Service Registry (Inter-Plugin Communication)
  - Plugins may register named services/handlers.
  - Other plugins resolve services by name for direct function calls.
  - Lightweight in-process pub/sub for decoupled notifications.

## Data Model (Host)

- `plugins`
  - `id` (uuid), `name` (unique), `version`, `description`, `is_core` (bool), `status` (`active`/`disabled`), timestamps.
- `plugin_migrations`
  - `plugin` (name), `migration_id` (filename stem), `applied_at`, `checksum`.
- Optional: `permissions` table to centralize permission keys advertised by plugins.

## Plugin SPI (Service Provider Interface)

Each plugin is a Python package under `plugins/<name>` with this structure:

```
plugins/
  myplugin/
    plugin.py           # exports get_plugin() -> BasePlugin
    routes.py           # optional, route handlers
    services.py         # optional, internal services
    migrations/
      0001_init.sql
      0002_feature_x.sql
    plugin.toml         # optional metadata (name, version, description)
```

SPI contract (Python abstract class `BasePlugin`):
- Properties: `name`, `version`, `description`, `is_core: bool = False`, `dependencies: list[str] = []`.
- Hooks:
  - `on_load(ctx)`: called after migrations and before route registration.
  - `register_routes(app, router)`: must register FastAPI routes.
  - `register_permissions() -> list[Permission]`: declarative list of permission keys.
  - `seed(db)`: optional, for seed data.
  - `on_unload(ctx)`: cleanup.
- Static: `migrations_path() -> Path`: path to the plugin’s migrations folder.

Validation: Host imports `plugins.<name>.plugin:get_plugin()` and checks the returned instance against the SPI.

## Plugin Lifecycle

- Install (dynamic): upload zip (API), host unpacks to `plugins/<name>`, validates SPI, records plugin in DB, applies migrations, loads plugin.
- Enable: switch `status=active`, load plugin; apply new migrations if present.
- Disable: `status=disabled`, unload plugin (remove routes/services).
- Reload: unload, re-import, apply pending migrations, load, re-register routes/services.
- Remove: only for non-core and disabled; delete files and DB records (keep `plugin_migrations` history or archive).

## API Surface (Host)

- `GET /health`
- `GET /plugins`: list installed plugins and status
- `GET /plugins/{name}`: details
- `POST /plugins/install` (multipart or URL): upload zip or fetch artifact, install and load
- `POST /plugins/{name}/enable`
- `POST /plugins/{name}/disable`
- `POST /plugins/{name}/reload`
- `DELETE /plugins/{name}`: remove if non-core and disabled

## Migrations

- Migration files are SQL, executed sequentially by lexicographic order.
- Each file is wrapped in a transaction; failures roll back and halt further processing.
- A checksum (e.g., SHA256) is recorded per applied file to detect drift.
- Host bootstrap ensures the `plugins` and `plugin_migrations` tables exist.

## Inter-Plugin Communication

- Service Registry: `register_service(name, obj)` and `get_service(name) -> obj`.
- Plugins can call each other’s services directly in-process.
- Optional pub/sub with simple channels for events (e.g., `bus.publish("student.created", payload)`).

## Makefile & Dev Flow

- Make targets:
  - `poetry-install`: set up venv and install deps
  - `dev`: run FastAPI with reload
  - `run`: run FastAPI
  - `fmt`/`lint`/`test`: formatting and checks
  - `migrate`: run host bootstrap and pending plugin migrations

Suggested repo layout:

```
server/
  app/
    main.py
    core/
      settings.py
      db.py
      models.py
      migrator.py
      plugin_spi.py
      plugin_manager.py
      service_registry.py
    api/
      admin.py
  pyproject.toml
  Makefile
plugins/
  core/
  sample_powertable/
  sample_reports/
```

## Security & Isolation Notes

- Uploaded artifacts are validated and unpacked into `plugins/` within the repo.
- Only load from whitelisted base directory; sanitize filenames when extracting zips.
- True process isolation is out-of-scope (in-process Python); if needed, consider separate processes and IPC in the future.

## Next Steps (Implementation Order)

1) Scaffold Poetry + FastAPI host in `server/` with core modules.
2) Implement DB connection and bootstrap migrations.
3) Implement SPI base class and PluginManager skeleton (discover, validate, load/unload).
4) Implement per-plugin SQL migrator and `plugin_migrations` tracking.
5) Implement plugin admin API (list/install/enable/disable/reload/remove).
6) Implement service registry and simple pub/sub.
7) Add two sample plugins with routes and migrations.
8) Makefile and docs polish.
