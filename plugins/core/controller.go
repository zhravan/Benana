package main

import (
	"fmt"

	"benana/types"

	"github.com/go-fuego/fuego"
)

var discoveryService *Discovery

func InitPluginController(db *types.Database) {
	repo := NewRepository(db)
	discoveryService = NewDiscovery(repo)
}


func validatePluginName(c interface{ PathParam(string) string }) (string, error) {
	name := c.PathParam("name")
	if name == "" {
		return "", fmt.Errorf("plugin name is required")
	}
	return name, nil
}

func GetPlugins(c fuego.ContextNoBody) ([]types.Plugin, error) {
	plugins, err := discoveryService.GetAllRegisteredPlugins(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugins: %w", err)
	}

	return plugins, nil
}

func GetPluginByName(c fuego.ContextNoBody) (types.Plugin, error) {
	name, err := validatePluginName(c)
	if err != nil {
		return types.Plugin{}, err
	}

	plugin, err := discoveryService.GetPluginByName(c.Request().Context(), name)
	if err != nil {
		return types.Plugin{}, fmt.Errorf("plugin not found: %w", err)
	}

	return plugin, nil
}
