package types

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Migrator struct {
	db       *bun.DB
	migrator *migrate.Migrator
}

func NewMigrator(db *bun.DB) *Migrator {
	migrations := migrate.NewMigrations()
	if err := migrations.Discover(migrationsFS); err != nil {
		log.Fatalf("Failed to discover migrations: %v", err)
	}

	migrator := migrate.NewMigrator(db, migrations)
	return &Migrator{
		db:       db,
		migrator: migrator,
	}
}

func (m *Migrator) Init(ctx context.Context) error {
	return m.migrator.Init(ctx)
}

func (m *Migrator) Migrate(ctx context.Context) error {
	group, err := m.migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if group.IsZero() {
		log.Println("No new migrations to run")
		return nil
	}

	log.Printf("Migrated to %s", group)
	return nil
}

func (m *Migrator) Rollback(ctx context.Context) error {
	group, err := m.migrator.Rollback(ctx)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	if group.IsZero() {
		log.Println("No migrations to rollback")
		return nil
	}

	log.Printf("Rolled back %s", group)
	return nil
}

func (m *Migrator) Status(ctx context.Context) error {
	ms, err := m.migrator.MigrationsWithStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	fmt.Printf("Migration Status:\n")
	for _, migration := range ms {
		status := "pending"
		if migration.IsApplied() {
			status = "applied"
		}
		fmt.Printf("  %s: %s\n", migration.Name, status)
	}
	return nil
}
