// this file provides a Compiler for managing Go plugin compilation.
//
// The Compiler is responsible for:
// - Compiling plugins to .so files for dynamic loading
// - Validating plugin structure and dependencies
// - Discovering new plugins in the plugins directory
// - Cleaning up old plugin binaries
// - Managing plugin source and binary directories

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"benana/types"
)

type Compiler struct {
	pluginsDir  string
	binariesDir string
}

func NewCompiler(pluginsDir, binariesDir string) *Compiler {
	return &Compiler{
		pluginsDir:  pluginsDir,
		binariesDir: binariesDir,
	}
}

func (c *Compiler) CompilePlugin(plugin *types.Plugin) error {
	if plugin.Type != "dynamic" {
		return fmt.Errorf("plugin %s is not a dynamic plugin", plugin.Name)
	}

	sourcePath := plugin.SourcePath
	if sourcePath == "" {
		sourcePath = filepath.Join(c.pluginsDir, plugin.Name)
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("plugin source directory does not exist: %s", sourcePath)
	}

	if err := os.MkdirAll(c.binariesDir, 0755); err != nil {
		return fmt.Errorf("failed to create binaries directory: %w", err)
	}

	absBinDir, err := filepath.Abs(c.binariesDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for binaries directory: %w", err)
	}
	binaryPath := filepath.Join(absBinDir, plugin.Name+".so")

	mainFile := filepath.Join(sourcePath, "main.go")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		mainFile = filepath.Join(sourcePath, "plugin.go")
		if _, err := os.Stat(mainFile); os.IsNotExist(err) {
			return fmt.Errorf("no main.go or plugin.go found in %s", sourcePath)
		}
	}

	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", binaryPath, mainFile)
	cmd.Dir = sourcePath
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to compile plugin %s: %w\nOutput: %s", plugin.Name, err, string(output))
	}

	plugin.BinaryPath = binaryPath

	return nil
}

func (c *Compiler) ValidatePlugin(pluginPath string) error {
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin directory does not exist: %s", pluginPath)
	}

	mainFile := filepath.Join(pluginPath, "main.go")
	pluginFile := filepath.Join(pluginPath, "plugin.go")

	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		if _, err := os.Stat(pluginFile); os.IsNotExist(err) {
			return fmt.Errorf("no main.go or plugin.go found in %s", pluginPath)
		}
	}

	return nil
}

func (c *Compiler) DiscoverPlugins() ([]types.Plugin, error) {
	var plugins []types.Plugin

	entries, err := os.ReadDir(c.pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if entry.Name() == "core" || entry.Name() == "types" || entry.Name() == ".git" {
			log.Printf("Skipping system directory: %s", entry.Name())
			continue
		}

		pluginPath := filepath.Join(c.pluginsDir, entry.Name())

		if err := c.ValidatePlugin(pluginPath); err != nil {
			log.Printf("Invalid plugin structure for %s: %v", entry.Name(), err)
			continue
		}

		plugin := types.Plugin{
			Name:       entry.Name(),
			Type:       "dynamic",
			SourcePath: pluginPath,
			Status:     "inactive",
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

func (c *Compiler) CleanupPlugin(plugin *types.Plugin) error {
	if plugin.BinaryPath != "" {
		if err := os.Remove(plugin.BinaryPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove binary %s: %w", plugin.BinaryPath, err)
		}
	}
	return nil
}


// ValidatePluginInterface compiles and tests if plugin implements the required interface
func (c *Compiler) ValidatePluginInterface(plugin *types.Plugin) error {
	if err := c.CompilePlugin(plugin); err != nil {
		return fmt.Errorf("plugin compilation failed: %w", err)
	}

	loader := NewPluginLoader()
	loadedPlugin, err := loader.LoadPlugin(context.Background(), plugin)
	if err != nil {
		c.CleanupPlugin(plugin)
		return fmt.Errorf("plugin interface validation failed: %w", err)
	}

	metadata := loadedPlugin.Instance.GetMetadata()
	if metadata.Name == "" {
		c.CleanupPlugin(plugin)
		return fmt.Errorf("plugin metadata validation failed: name is empty")
	}

	if metadata.Version == "" {
		c.CleanupPlugin(plugin)
		return fmt.Errorf("plugin metadata validation failed: version is empty")
	}

	if err := loader.UnloadPlugin(context.Background(), plugin.Name); err != nil {
		log.Printf("Warning: failed to cleanup test plugin instance: %v", err)
	}

	return nil
}
