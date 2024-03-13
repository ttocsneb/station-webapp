package database

import (
	"database/sql"
	"embed"
	"fmt"

	log "github.com/sirupsen/logrus"
)

//go:embed migrations/*
var migrationFiles embed.FS

func getVersion(db *sql.DB) int {
	row := db.QueryRow(`SELECT (version) FROM db_info WHERE id = 1;`)

	var version int
	err := row.Scan(&version)
	if err != nil {
		return 0
	}

	return version
}

func migrate(file string, version int) func(*sql.DB) (int, error) {
	return func(db *sql.DB) (int, error) {
		log.Infof("Migrating to V%d", version)
		var err error

		f, err := migrationFiles.ReadFile(file)
		if err != nil {
			return 0, err
		}

		_, err = db.Exec(string(f))
		if err != nil {
			return 0, err
		}
		return version, nil
	}
}

var migrations = map[int]func(*sql.DB) (int, error){
	0: migrate("migrations/v2.sql", 2),
}

func Migrate(db *sql.DB) error {
	version := getVersion(db)
	var err error

	for version != 2 {
		migrate, exists := migrations[version]
		if !exists {
			return fmt.Errorf("Unknown Version %v", version)
		}
		version, err = migrate(db)
		if err != nil {
			return err
		}
	}
	return nil
}
