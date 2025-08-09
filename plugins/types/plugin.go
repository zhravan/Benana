package types

import (
	"time"

	"github.com/uptrace/bun"
)

type PluginTable struct {
	bun.BaseModel `bun:"table:plugins,alias:p"`

	ID          string    `bun:"id,pk"`
	Name        string    `bun:"name,notnull,unique"`
	Description string    `bun:"description"`
	Version     string    `bun:"version,notnull"`
	Author      string    `bun:"author"`
	Type        string    `bun:"type,notnull,default:'static'"`
	Path        string    `bun:"path"`
	BinaryPath  string    `bun:"binary_path"`
	SourcePath  string    `bun:"source_path"`
	Status      string    `bun:"status,notnull,default:'active'"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}

type PluginEndpointTable struct {
	bun.BaseModel `bun:"table:plugin_endpoints,alias:pe"`

	ID        int64     `bun:"id,pk,autoincrement"`
	PluginID  string    `bun:"plugin_id,notnull"`
	Method    string    `bun:"method,notnull"`
	Path      string    `bun:"path,notnull"`
	Handler   string    `bun:"handler,notnull"`
	Type      string    `bun:"type,notnull,default:'REST'"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}

type PluginEndpoint struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
	Type    string `json:"type"`
}

type Plugin struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Version     string           `json:"version"`
	Author      string           `json:"author"`
	Type        string           `json:"type"` // "dynamic" for .so plugins, "static" for built-in
	Path        string           `json:"path"` // path to .so file or source directory
	Status      string           `json:"status"`
	Endpoints   []PluginEndpoint `json:"endpoints,omitempty"`
	BinaryPath  string           `json:"binary_path,omitempty"` // path to compiled .so file
	SourcePath  string           `json:"source_path,omitempty"` // path to source code
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type PluginChangeEvent struct {
	Operation string                 `json:"operation"`
	Table     string                 `json:"table"`
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	OldData   map[string]interface{} `json:"old_data,omitempty"`
	NewData   map[string]interface{} `json:"new_data,omitempty"`
	Timestamp float64                `json:"timestamp"`
}
