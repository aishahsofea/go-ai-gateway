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

	// ensure there is no leak when there is a panic
	defer func() {
		err := sqlDB.Close()
		if err != nil {
			fmt.Printf("error closing sqlDB: %v\n", err)
		}
	}()

	err = goose.Up(sqlDB, dir)
	if err != nil {
		return fmt.Errorf("error running goose.Up: %w", err)
	}

	return nil
}
