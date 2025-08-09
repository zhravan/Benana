# Dynamic Plugin Architecture

## Overview

This system implements a dynamic plugin architecture for Go applications. Plugins are compiled at runtime to shared object files and loaded dynamically without requiring server restarts.

## Architecture Components

### Core Components

The system consists of a main server that manages three primary subsystems. The HTTP server handles incoming requests using the Fuego framework, while the Plugin Manager orchestrates the entire plugin lifecycle. A PostgreSQL database stores plugin metadata and configuration. Below this sits the Plugin Subsystem, which contains six specialized services: Discovery Service finds new plugins, Compiler builds them into shared objects, Loader handles dynamic loading, Repository manages database operations, Controller provides API endpoints, and Events handles real-time notifications through PostgreSQL's LISTEN/NOTIFY system.

Plugin files are organized with source code in parallel directories (like ../powertable/) containing main.go, go.mod, and plugin.yaml files. Compiled shared object files are stored in ./bin/plugins/ for runtime loading.

## System Flow

### Auto-Discovery Process

When the server starts, it scans the ../plugins/ directory for plugin directories. For each directory found, the system validates the plugin structure by checking for main.go or plugin.go files. If the plugin doesn't exist in the database, the system validates the plugin interface, compiles it to a shared object file, and tests loading. Valid plugins are registered in the database, loaded into memory, and have their routes registered with the HTTP server.

### Plugin Loading Process

The Plugin Manager coordinates with the Compiler to build the plugin using "go build -buildmode=plugin". Once compiled, the Loader opens the shared object file, looks up the "NewPlugin" factory function, creates an instance, and initializes it. The plugin then registers its routes with the server, and the database is updated to reflect the active status.

## Plugin Interface

### Required Interface Implementation

Every plugin must implement the PluginInterface, which requires four methods: Initialize for setup with context, GetMetadata to return plugin information, RegisterRoutes to register HTTP endpoints with the server, and Cleanup for resource cleanup when the plugin is unloaded.

### Example Plugin Implementation

A typical plugin implementation exports a NewPlugin factory function that returns a struct implementing the PluginInterface. The plugin includes metadata like name, version, description, and author. Route registration typically uses the Fuego framework to define HTTP handlers for GET, POST, and other HTTP methods.

```go
package main

import (
    "context"
    "benana/types"
    "github.com/go-fuego/fuego"
)

type MyPlugin struct {
    metadata types.PluginMetadata
}

func NewPlugin() types.PluginInterface {
    return &MyPlugin{
        metadata: types.PluginMetadata{
            Name:        "myplugin",
            Version:     "1.0.0",
            Description: "My awesome plugin",
            Author:      "Developer",
        },
    }
}

func (p *MyPlugin) Initialize(ctx context.Context) error {
    return nil
}

func (p *MyPlugin) GetMetadata() types.PluginMetadata {
    return p.metadata
}

func (p *MyPlugin) RegisterRoutes(server *fuego.Server) error {
    fuego.Get(server, "/myplugin", p.handleGet)
    fuego.Post(server, "/myplugin", p.handlePost)
    return nil
}

func (p *MyPlugin) Cleanup(ctx context.Context) error {
    return nil
}

func (p *MyPlugin) handleGet(c fuego.ContextNoBody) (interface{}, error) {
    return map[string]string{"message": "Hello from MyPlugin!"}, nil
}

func (p *MyPlugin) handlePost(c fuego.ContextWithBody[map[string]interface{}]) (interface{}, error) {
    body, _ := c.Body()
    return map[string]interface{}{"received": body}, nil
}
```

## Database Schema

The system uses two main tables. The plugins table stores core plugin information including id, name, description, version, author, type, paths to source and binary files, status, and timestamps. The plugin_endpoints table tracks individual HTTP endpoints registered by each plugin, linking them back to the parent plugin with foreign key constraints.

```sql
CREATE TABLE plugins (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    version VARCHAR(100) NOT NULL,
    author VARCHAR(255),
    type VARCHAR(100) NOT NULL DEFAULT 'static',
    path VARCHAR(500),
    binary_path VARCHAR(500),
    source_path VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE plugin_endpoints (
    id BIGSERIAL PRIMARY KEY,
    plugin_id VARCHAR(255) NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    handler VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'REST',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## API Endpoints

The system provides a RESTful API for plugin management. GET /plugins lists all registered plugins, while GET /plugins/{name} returns details for a specific plugin. POST /plugins/{name}/reload forces recompilation and reloading of a plugin. POST /plugins/{name}/toggle enables or disables a plugin.

API responses include comprehensive plugin information such as unique identifiers, metadata, file paths, and current status. The reload endpoint provides feedback on the reload process, while the toggle endpoint confirms status changes.

### API Examples

#### List All Plugins
```bash
curl http://localhost:9999/plugins
```

```json
[
  {
    "id": "uuid-here",
    "name": "powertable",
    "description": "A plugin for creating powerful tables",
    "version": "0.1.0",
    "author": "Benana Team",
    "type": "dynamic",
    "status": "active",
    "source_path": "../powertable",
    "binary_path": "/abs/path/bin/plugins/powertable.so"
  }
]
```

#### Get Plugin by Name
```bash
curl http://localhost:9999/plugins/powertable
```

#### Reload Plugin
```bash
curl -X POST http://localhost:9999/plugins/powertable/reload
```

```json
{
  "message": "Plugin reload triggered successfully",
  "plugin": "powertable",
  "status": "reloading"
}
```

#### Toggle Plugin Status
```bash
curl -X POST http://localhost:9999/plugins/powertable/toggle
```

```json
{
  "message": "Plugin status updated successfully",
  "plugin": "powertable",
  "status": "inactive"
}
```

## Event-Driven Updates

The system uses PostgreSQL triggers to automatically notify the application when plugins are modified. A trigger function builds a JSON payload containing the operation type, table name, plugin information, and timestamp, then sends it via pg_notify. The Go application listens for these notifications and responds by loading, reloading, or unloading plugins as appropriate, then updating the HTTP routes accordingly.

### PostgreSQL Triggers

```sql
CREATE OR REPLACE FUNCTION notify_plugin_change()
RETURNS TRIGGER AS $$
DECLARE payload JSON;
BEGIN
    -- Build notification payload
    payload = json_build_object(
        'operation', TG_OP,
        'table', TG_TABLE_NAME,
        'id', NEW.id,
        'name', NEW.name,
        'new_data', row_to_json(NEW),
        'timestamp', extract(epoch from now())
    );
    
    -- Send notification
    PERFORM pg_notify('plugin_changes', payload::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

## Plugin Development Guide

Creating a new plugin involves several steps. First, create a new directory for the plugin and initialize a Go module. Add dependencies, particularly the shared types module and Fuego framework. Implement the required PluginInterface in main.go, including the NewPlugin factory function. Test the plugin by starting the server, which will automatically discover, compile, and load the new plugin. Verify the plugin registration and test its endpoints using standard HTTP requests.

### Step-by-Step Process

#### 1. Create Plugin Directory
```bash
mkdir ../plugins/myplugin
cd ../plugins/myplugin
```

#### 2. Initialize Go Module
```bash
go mod init myplugin
```

#### 3. Add Dependencies
```go
// go.mod
module myplugin

go 1.24.2

replace benana/types => ../types

require (
    benana/types v0.0.0-00010101000000-000000000000
    github.com/go-fuego/fuego v0.18.8
)
```

#### 4. Implement Plugin
Create `main.go` with the required interface implementation (see example above).

#### 5. Test Plugin
Start the server - your plugin will be auto-discovered, compiled, and loaded:
```bash
cd ../core
go run .
```

#### 6. Verify Plugin
```bash
curl http://localhost:9999/plugins/myplugin  # Check registration
curl http://localhost:9999/myplugin          # Test plugin endpoint
```

## Key Features

The system provides automatic discovery of plugins in the plugins directory without manual registration. Go source code is compiled to shared object files at runtime using the standard Go build system with plugin mode. Interface validation ensures plugins implement the required interface before registration. Hot reload capability allows plugins to be reloaded without server restart through API calls. Database integration stores plugin metadata in PostgreSQL with real-time updates via LISTEN/NOTIFY. The event-driven architecture uses database triggers to notify the application of changes, enabling automatic plugin lifecycle management. Route management automatically registers and removes plugin routes with the HTTP server.

### Feature Summary

- **Auto-Discovery**: Plugins are automatically found in `../plugins/` directory
- **Runtime Compilation**: Go source code compiled to `.so` files at runtime
- **Interface Validation**: Plugins must implement `PluginInterface`
- **Hot Reload**: Plugins can be reloaded without server restart
- **Database Integration**: Plugin metadata stored in PostgreSQL
- **Event-Driven Architecture**: Database triggers notify application of changes
- **Route Management**: Plugin routes automatically registered with HTTP server
