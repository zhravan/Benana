package types

import (
	"context"
	"github.com/go-fuego/fuego"
)

// PluginInterface defines the contract that all dynamic plugins must implement
type PluginInterface interface {
	// Initialize the plugin with any required setup
	Initialize(ctx context.Context) error
	
	// Get plugin metadata
	GetMetadata() PluginMetadata
	
	// Register routes with the server
	RegisterRoutes(server *fuego.Server) error
	
	// Cleanup resources when plugin is unloaded
	Cleanup(ctx context.Context) error
}

// PluginMetadata contains basic information about the plugin
type PluginMetadata struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	Endpoints   []PluginEndpoint      `json:"endpoints"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// RouteHandler represents a plugin route handler
type RouteHandler struct {
	Method  string
	Path    string
	Handler interface{}
}

// PluginFactory is the function signature that plugins must export
// This will be looked up by symbol name when loading the plugin
type PluginFactory func() PluginInterface