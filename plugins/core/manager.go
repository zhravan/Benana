package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"benana/types"

	"github.com/go-fuego/fuego"
)

type PluginManager struct {
	server       *fuego.Server
	discovery    *Discovery
	plugins      map[string]*types.Plugin
	pluginLoader *PluginLoader
	compiler     *Compiler
	mutex        sync.RWMutex
}

func NewPluginManager(server *fuego.Server, discovery *Discovery, pluginsDir, binariesDir string) *PluginManager {
	return &PluginManager{
		server:       server,
		discovery:    discovery,
		plugins:      make(map[string]*types.Plugin),
		pluginLoader: NewPluginLoader(),
		compiler:     NewCompiler(pluginsDir, binariesDir),
	}
}

func (pm *PluginManager) LoadActivePlugins(ctx context.Context) error {
	plugins, err := pm.discovery.GetAllRegisteredPlugins(ctx)
	if err != nil {
		return fmt.Errorf("failed to load active plugins: %w", err)
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for _, plugin := range plugins {
		if plugin.Status == "active" {
			pm.plugins[plugin.ID] = &plugin
			if plugin.Type == "dynamic" {
				if err := pm.loadPlugin(ctx, &plugin); err != nil {
					log.Printf("Failed to load dynamic plugin %s: %v", plugin.Name, err)
				}
			} else {
				log.Printf("Skipping non-dynamic plugin %s (type: %s)", plugin.Name, plugin.Type)
			}
		}
	}

	log.Printf("Loaded %d active plugins", len(pm.plugins))
	return nil
}

func (pm *PluginManager) HandlePluginChange(plugin types.Plugin) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	exists := pm.plugins[plugin.ID] != nil

	if plugin.Status == "active" {
		pm.plugins[plugin.ID] = &plugin
		if plugin.Type == "dynamic" {
			action := "load"
			if exists {
				action = "reload"
			}
			if err := pm.loadPlugin(context.Background(), &plugin); err != nil {
				log.Printf("Failed to %s dynamic plugin %s: %v", action, plugin.Name, err)
			}
		}
	} else if exists {
		delete(pm.plugins, plugin.ID)
	}
}

func (pm *PluginManager) StartEventListener(ctx context.Context) error {
	onPluginChange := func(plugin types.Plugin, operation string) {
		switch operation {
		case "INSERT", "UPDATE":
			pm.HandlePluginChange(plugin)
		case "DELETE":
			plugin.Status = "inactive"
			pm.HandlePluginChange(plugin)
		}
	}

	return pm.discovery.StartPluginChangeListener(ctx, onPluginChange)
}

func (pm *PluginManager) loadPlugin(ctx context.Context, plugin *types.Plugin) error {
	if plugin.BinaryPath == "" || pm.needsRecompilation(plugin) {
		if err := pm.compiler.CompilePlugin(plugin); err != nil {
			return fmt.Errorf("failed to compile plugin %s: %w", plugin.Name, err)
		}
		if err := pm.discovery.UpdatePlugin(ctx, *plugin); err != nil {
			log.Printf("Failed to update plugin binary path in database: %v", err)
		}
	}

	loadedPlugin, err := pm.pluginLoader.LoadPlugin(ctx, plugin)
	if err != nil {
		return fmt.Errorf("failed to load plugin %s: %w", plugin.Name, err)
	}

	if err := pm.pluginLoader.RegisterPluginRoutes(pm.server, plugin.Name); err != nil {
		return fmt.Errorf("failed to register routes for plugin %s: %w", plugin.Name, err)
	}

	log.Printf("Dynamic plugin %s loaded successfully with %d endpoints", plugin.Name, len(loadedPlugin.Metadata.Endpoints))
	return nil
}

func (pm *PluginManager) needsRecompilation(plugin *types.Plugin) bool {
	return plugin.BinaryPath == "" || plugin.SourcePath != ""
}

func (pm *PluginManager) DiscoverNewPlugins(ctx context.Context) ([]types.Plugin, error) {
	return pm.compiler.DiscoverPlugins()
}

func (pm *PluginManager) ValidateAndRegisterPlugin(ctx context.Context, plugin *types.Plugin) error {
	if err := pm.compiler.ValidatePluginInterface(plugin); err != nil {
		return fmt.Errorf("plugin validation failed for %s: %w", plugin.Name, err)
	}

	if err := pm.discovery.AddNewPlugin(ctx, *plugin); err != nil {
		pm.compiler.CleanupPlugin(plugin)
		return fmt.Errorf("failed to register plugin %s in database: %w", plugin.Name, err)
	}

	return nil
}
