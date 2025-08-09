package main

import (
	"context"

	"benana/types"
)

type Repository interface {
	GetAllRegisteredPlugins(ctx context.Context) ([]types.Plugin, error)
	GetPluginsByStatus(ctx context.Context, status string) ([]types.Plugin, error)
	AddNewPlugin(ctx context.Context, plugin types.Plugin) error
	RemovePlugin(ctx context.Context, pluginID string) error
	UpdatePlugin(ctx context.Context, plugin types.Plugin) error
	GetPlugin(ctx context.Context, pluginID string) (types.Plugin, error)
	GetPluginByName(ctx context.Context, name string) (types.Plugin, error)
}

type repository struct {
	db *types.Database
}

func NewRepository(db *types.Database) Repository {
	return &repository{db: db}
}

func (r *repository) GetAllRegisteredPlugins(ctx context.Context) ([]types.Plugin, error) {
	return r.GetPluginsByStatus(ctx, "active")
}

func (r *repository) GetPluginsByStatus(ctx context.Context, status string) ([]types.Plugin, error) {
	var rows []types.PluginTable
	err := r.db.DB.NewSelect().Model(&rows).Where("status = ?", status).Scan(ctx)
	if err != nil {
		return nil, err
	}

	plugins := make([]types.Plugin, len(rows))
	for i, row := range rows {
		endpoints, err := r.getPluginEndpoints(ctx, row.ID)
		if err != nil {
			return nil, err
		}

		plugins[i] = types.Plugin{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			Version:     row.Version,
			Author:      row.Author,
			Type:        row.Type,
			Path:        row.Path,
			BinaryPath:  row.BinaryPath,
			SourcePath:  row.SourcePath,
			Status:      row.Status,
			Endpoints:   endpoints,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
	}

	return plugins, nil
}

func (r *repository) AddNewPlugin(ctx context.Context, plugin types.Plugin) error {
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row := types.PluginTable{
		ID:          plugin.ID,
		Name:        plugin.Name,
		Description: plugin.Description,
		Version:     plugin.Version,
		Author:      plugin.Author,
		Type:        plugin.Type,
		Path:        plugin.Path,
		BinaryPath:  plugin.BinaryPath,
		SourcePath:  plugin.SourcePath,
		Status:      "active",
	}

	_, err = tx.NewInsert().Model(&row).Exec(ctx)
	if err != nil {
		return err
	}

	if len(plugin.Endpoints) > 0 {
		for _, endpoint := range plugin.Endpoints {
			endpointRow := types.PluginEndpointTable{
				PluginID: plugin.ID,
				Method:   endpoint.Method,
				Path:     endpoint.Path,
				Handler:  endpoint.Handler,
				Type:     endpoint.Type,
			}
			_, err = tx.NewInsert().Model(&endpointRow).Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *repository) RemovePlugin(ctx context.Context, pluginID string) error {
	_, err := r.db.DB.NewDelete().
		Model((*types.PluginTable)(nil)).
		Where("id = ?", pluginID).
		Exec(ctx)
	return err
}

func (r *repository) UpdatePlugin(ctx context.Context, plugin types.Plugin) error {
	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.NewUpdate().
		Model((*types.PluginTable)(nil)).
		Set("name = ?", plugin.Name).
		Set("description = ?", plugin.Description).
		Set("version = ?", plugin.Version).
		Set("author = ?", plugin.Author).
		Set("type = ?", plugin.Type).
		Set("path = ?", plugin.Path).
		Set("status = ?", plugin.Status).
		Set("updated_at = current_timestamp").
		Where("id = ?", plugin.ID).
		Exec(ctx)
	if err != nil {
		return err
	}

	if len(plugin.Endpoints) > 0 {
		_, err = tx.NewDelete().
			Model((*types.PluginEndpointTable)(nil)).
			Where("plugin_id = ?", plugin.ID).
			Exec(ctx)
		if err != nil {
			return err
		}

		for _, endpoint := range plugin.Endpoints {
			endpointRow := types.PluginEndpointTable{
				PluginID: plugin.ID,
				Method:   endpoint.Method,
				Path:     endpoint.Path,
				Handler:  endpoint.Handler,
				Type:     endpoint.Type,
			}
			_, err = tx.NewInsert().Model(&endpointRow).Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *repository) GetPlugin(ctx context.Context, pluginID string) (types.Plugin, error) {
	var row types.PluginTable
	err := r.db.DB.NewSelect().
		Model(&row).
		Where("id = ? AND status = ?", pluginID, "active").
		Scan(ctx)
	if err != nil {
		return types.Plugin{}, err
	}

	endpoints, err := r.getPluginEndpoints(ctx, row.ID)
	if err != nil {
		return types.Plugin{}, err
	}

	return types.Plugin{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Version:     row.Version,
		Author:      row.Author,
		Type:        row.Type,
		Path:        row.Path,
		BinaryPath:  row.BinaryPath,
		SourcePath:  row.SourcePath,
		Status:      row.Status,
		Endpoints:   endpoints,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *repository) GetPluginByName(ctx context.Context, name string) (types.Plugin, error) {
	var row types.PluginTable
	err := r.db.DB.NewSelect().
		Model(&row).
		Where("name = ? AND status = ?", name, "active").
		Scan(ctx)
	if err != nil {
		return types.Plugin{}, err
	}

	endpoints, err := r.getPluginEndpoints(ctx, row.ID)
	if err != nil {
		return types.Plugin{}, err
	}

	return types.Plugin{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Version:     row.Version,
		Author:      row.Author,
		Type:        row.Type,
		Path:        row.Path,
		BinaryPath:  row.BinaryPath,
		SourcePath:  row.SourcePath,
		Status:      row.Status,
		Endpoints:   endpoints,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *repository) getPluginEndpoints(ctx context.Context, pluginID string) ([]types.PluginEndpoint, error) {
	var rows []types.PluginEndpointTable
	err := r.db.DB.NewSelect().
		Model(&rows).
		Where("plugin_id = ?", pluginID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	endpoints := make([]types.PluginEndpoint, len(rows))
	for i, row := range rows {
		endpoints[i] = types.PluginEndpoint{
			Method:  row.Method,
			Path:    row.Path,
			Handler: row.Handler,
			Type:    row.Type,
		}
	}

	return endpoints, nil
}
