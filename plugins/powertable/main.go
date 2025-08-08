package main

import (
	"context"
	"log"

	"benana/types"

	"github.com/go-fuego/fuego"
)

type PowerTablePlugin struct {
	metadata types.PluginMetadata
}

func NewPlugin() types.PluginInterface {
	return &PowerTablePlugin{
		metadata: types.PluginMetadata{
			Name:        "powertable",
			Version:     "0.1.0",
			Description: "A plugin for creating powerful tables",
			Author:      "Benana Team",
			Endpoints: []types.PluginEndpoint{
				{
					Method:  "GET",
					Path:    "/hello",
					Handler: "hello",
					Type:    "REST",
				},
			},
		},
	}
}

func (p *PowerTablePlugin) Initialize(ctx context.Context) error {
	log.Printf("Initializing PowerTable plugin v%s", p.metadata.Version)
	return nil
}

func (p *PowerTablePlugin) GetMetadata() types.PluginMetadata {
	return p.metadata
}

func (p *PowerTablePlugin) RegisterRoutes(server *fuego.Server) error {
	fuego.Get(server, "/hello", p.hello)
	log.Printf("PowerTable plugin registered routes: GET /hello")
	return nil
}

func (p *PowerTablePlugin) Cleanup(ctx context.Context) error {
	log.Printf("Cleaning up PowerTable plugin")
	return nil
}

func (p *PowerTablePlugin) hello(c fuego.ContextNoBody) (map[string]interface{}, error) {
	return map[string]interface{}{
		"message": "PowerTable plugin - GET /hello",
		"plugin":  p.metadata.Name,
		"version": p.metadata.Version,
		"tables":  []string{"hello"},
	}, nil
}
