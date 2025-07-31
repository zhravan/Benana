package main

import (
	"fmt"
	"sync"

	"github.com/go-fuego/fuego"
)

func main() {
	s := fuego.NewServer()
	s.Run()
}

type Plugins struct {
	data sync.Map
}

type Plugin struct {
	ID      string
	Name    string
	Version string
	Type    string
}

type PluginsRepository interface {
	Add()
	Delete()
	Update()
}

func (p *Plugins) Add(Plugin Plugin) {
	p.data.Store(Plugin.ID, Plugin)
}

func (p *Plugins) Delete(ID string) {
	p.data.Delete(ID)
}

func (p *Plugins) GetByName(name string) (*Plugin, error) {
	var result *Plugin
	found := false

	p.data.Range(func(_, value any) bool {
		plugin, ok := value.(Plugin)
		if !ok {
			return true
		}
		if plugin.Name == name {
			result = &plugin
			found = true
			return false
		}
		return true
	})

	if !found {
		return nil, fmt.Errorf("Error: plugin with name %q not found", name)
	}
	return result, nil
}

func (p *Plugins) GetByID(ID string) (plugin *Plugin, err error) {
	value, ok := p.data.Load(ID)

	if !ok {
		return nil, err
	}
	plugin, ok = value.(*Plugin)
	if !ok {
		return nil, fmt.Errorf("Error: Failure in conversion of value to type *Plugin")
	}
	return plugin, nil
}

// func (p *Plugins) GetAll(ID string) (plugin )