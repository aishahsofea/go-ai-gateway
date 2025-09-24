package db

import (
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func MigrateFS(db *DB, migrationsFS fs.FS, dir string) error {

	goose.SetBaseFS(migrationsFS)

	defer func() { goose.SetBaseFS(nil) }()
	return Migrate(db, dir)
}

func Migrate(db *DB, dir string) error {

	err := goose.SetDialect("postgres")
	if err != nil {
		return fmt.Errorf("error setting goose dialect: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(db.Pool)

	err = goose.Up(sqlDB, dir)
	if err != nil {
		return fmt.Errorf("error running goose.Up: %w", err)
	}

	err = sqlDB.Close()
	if err != nil {
		return fmt.Errorf("error closing sqlDB: %w", err)
	}

	return nil
}
