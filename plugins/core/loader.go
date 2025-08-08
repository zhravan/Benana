package main

import (
	"context"
	"fmt"
	"plugin"
	"sync"

	"benana/types"

	"github.com/go-fuego/fuego"
)

type PluginLoader struct {
	loadedPlugins map[string]*LoadedPlugin
	mutex         sync.RWMutex
}

type LoadedPlugin struct {
	Plugin     *plugin.Plugin
	Instance   types.PluginInterface
	Metadata   types.PluginMetadata
	BinaryPath string
}

func NewPluginLoader() *PluginLoader {
	return &PluginLoader{
		loadedPlugins: make(map[string]*LoadedPlugin),
	}
}

func (pl *PluginLoader) LoadPlugin(ctx context.Context, pluginInfo *types.Plugin) (*LoadedPlugin, error) {
	if pluginInfo.Type != "dynamic" {
		return nil, fmt.Errorf("plugin %s is not a dynamic plugin", pluginInfo.Name)
	}

	if pluginInfo.BinaryPath == "" {
		return nil, fmt.Errorf("no binary path specified for plugin %s", pluginInfo.Name)
	}

	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	if loaded, exists := pl.loadedPlugins[pluginInfo.Name]; exists {
		return loaded, nil
	}

	p, err := plugin.Open(pluginInfo.BinaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin %s: %w", pluginInfo.Name, err)
	}

	factorySymbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin %s does not export NewPlugin function: %w", pluginInfo.Name, err)
	}

	factory, ok := factorySymbol.(func() types.PluginInterface)
	if !ok {
		return nil, fmt.Errorf("plugin %s NewPlugin function has wrong signature", pluginInfo.Name)
	}

	instance := factory()
	if instance == nil {
		return nil, fmt.Errorf("plugin %s NewPlugin returned nil", pluginInfo.Name)
	}

	if err := instance.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize plugin %s: %w", pluginInfo.Name, err)
	}

	metadata := instance.GetMetadata()

	loaded := &LoadedPlugin{
		Plugin:     p,
		Instance:   instance,
		Metadata:   metadata,
		BinaryPath: pluginInfo.BinaryPath,
	}

	pl.loadedPlugins[pluginInfo.Name] = loaded
	return loaded, nil
}

func (pl *PluginLoader) UnloadPlugin(ctx context.Context, pluginName string) error {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	loaded, exists := pl.loadedPlugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", pluginName)
	}

	if err := loaded.Instance.Cleanup(ctx); err != nil {
		return fmt.Errorf("failed to cleanup plugin %s: %w", pluginName, err)
	}

	delete(pl.loadedPlugins, pluginName)
	return nil
}

func (pl *PluginLoader) RegisterPluginRoutes(server *fuego.Server, pluginName string) error {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()

	loaded, exists := pl.loadedPlugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", pluginName)
	}

	return loaded.Instance.RegisterRoutes(server)
}

