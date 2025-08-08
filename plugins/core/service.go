package main

import (
	"context"
	"encoding/json"
	"log"

	"benana/types"

	"github.com/jackc/pgx/v5"
)

type Discovery struct {
	repo Repository
}

func NewDiscovery(repo Repository) *Discovery {
	return &Discovery{repo: repo}
}	

func (d *Discovery) StartPluginChangeListener(ctx context.Context, onPluginChange func(types.Plugin, string)) error {
	config, err := types.LoadConfig()
	if err != nil {
		return err
	}

	dsn := types.GetDSN(config.Database)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}

	_, err = conn.Exec(ctx, "LISTEN plugin_changes")
	if err != nil {
		return err
	}

	go func() {
		defer conn.Close(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				notification, err := conn.WaitForNotification(ctx)
				if err != nil {
					log.Printf("Notification error: %v", err)
					continue
				}
				d.HandlePluginChange(onPluginChange, notification.Payload)
			}
		}
	}()

	return nil
}

func (d *Discovery) HandlePluginChange(onPluginChange func(types.Plugin, string), payload string) {
	var event types.PluginChangeEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return
	}

	var plugin types.Plugin
	if event.NewData != nil {
		// this is the case when the plugin is created or updated
		plugin = MapToPlugin(event.NewData)
	} else {
		// this is the case when the plugin is deleted
		plugin = types.Plugin{ID: event.ID}
	}

	onPluginChange(plugin, event.Operation)
}

func MapToPlugin(data map[string]interface{}) types.Plugin {
	plugin := types.Plugin{}
	if id, ok := data["id"].(string); ok {
		plugin.ID = id
	}
	if name, ok := data["name"].(string); ok {
		plugin.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		plugin.Description = desc
	}
	if version, ok := data["version"].(string); ok {
		plugin.Version = version
	}
	if author, ok := data["author"].(string); ok {
		plugin.Author = author
	}
	if pluginType, ok := data["type"].(string); ok {
		plugin.Type = pluginType
	}
	if path, ok := data["path"].(string); ok {
		plugin.Path = path
	}
	if status, ok := data["status"].(string); ok {
		plugin.Status = status
	}
	return plugin
}

func (d *Discovery) GetAllRegisteredPlugins(ctx context.Context) ([]types.Plugin, error) {
	return d.repo.GetAllRegisteredPlugins(ctx)
}

func (d *Discovery) AddNewPlugin(ctx context.Context, plugin types.Plugin) error {
	return d.repo.AddNewPlugin(ctx, plugin)
}

func (d *Discovery) UpdatePlugin(ctx context.Context, plugin types.Plugin) error {
	return d.repo.UpdatePlugin(ctx, plugin)
}

func (d *Discovery) GetPluginByName(ctx context.Context, name string) (types.Plugin, error) {
	return d.repo.GetPluginByName(ctx, name)
}
