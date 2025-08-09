package main

import (
	"benana/types"
	"context"
	"log"

	"github.com/go-fuego/fuego"
	"github.com/google/uuid"
)

// TODO: Move to config or setup a discovery property in the plugin manager
const (
	pluginsDir   = "../"
	binariesDir  = "./bin/plugins"
	activeStatus = "active"
)

func main() {
	db, err := types.ConnectDatabaseAndRunMigrations()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	s := fuego.NewServer()
	setupPluginSystem(s, db)
	setupRoutes(s)
	s.Run()
}

func setupPluginSystem(s *fuego.Server, db *types.Database) {
	InitPluginController(db)

	discovery := NewDiscovery(NewRepository(db))
	pluginManager := NewPluginManager(s, discovery, pluginsDir, binariesDir)

	discoverAndRegisterPlugins(pluginManager, discovery)
	loadActivePlugins(pluginManager)
	startEventListener(pluginManager)
}

func discoverAndRegisterPlugins(pluginManager *PluginManager, discovery *Discovery) {
	discoveredPlugins, err := pluginManager.DiscoverNewPlugins(context.Background())
	if err != nil {
		log.Printf("Failed to discover plugins: %v", err)
		return
	}

	for _, discoveredPlugin := range discoveredPlugins {
		if isPluginAlreadyRegistered(discovery, discoveredPlugin.Name) {
			continue
		}

		registerNewPlugin(pluginManager, &discoveredPlugin)
	}
}

func isPluginAlreadyRegistered(discovery *Discovery, pluginName string) bool {
	if existing, err := discovery.GetPluginByName(context.Background(), pluginName); err == nil {
		log.Printf("Plugin %s already exists in database, skipping registration", existing.Name)
		return true
	}
	return false
}

func registerNewPlugin(pluginManager *PluginManager, discoveredPlugin *types.Plugin) {
	discoveredPlugin.ID = uuid.New().String()
	discoveredPlugin.Status = activeStatus

	if err := pluginManager.ValidateAndRegisterPlugin(context.Background(), discoveredPlugin); err != nil {
		log.Printf("Failed to validate and register plugin %s: %v", discoveredPlugin.Name, err)
		return
	}

	log.Printf("Successfully validated and registered plugin: %s", discoveredPlugin.Name)
}

func loadActivePlugins(pluginManager *PluginManager) {
	if err := pluginManager.LoadActivePlugins(context.Background()); err != nil {
		log.Printf("Failed to load active plugins: %v", err)
	}
}

func startEventListener(pluginManager *PluginManager) {
	if err := pluginManager.StartEventListener(context.Background()); err != nil {
		log.Printf("Failed to start plugin event listener: %v", err)
	}
}

func setupRoutes(s *fuego.Server) {
	fuego.Get(s, "/plugins", GetPlugins)
	fuego.Get(s, "/plugins/{name}", GetPluginByName)
}
