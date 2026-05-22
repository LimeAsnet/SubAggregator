package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/LimeAsnet/SubAggregator/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const defaultMigrationsPath = "file://internal/migrations"

func main() {
	migrationsPath := flag.String("path", defaultMigrationsPath, "migrations source, e.g. file://internal/migrations")
	flag.Parse()

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	cfg := config.InitConfig()
	m, err := migrate.New(*migrationsPath, cfg.Database.DSN())
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}
	defer closeMigrate(m)

	switch flag.Arg(0) {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate up: %v", err)
		}
		log.Println("migrate up: ok")
	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate down: %v", err)
		}
		log.Println("migrate down: ok")
	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("migrate version: %v", err)
		}
		log.Printf("version: %d, dirty: %v", version, dirty)
	case "force":
		if flag.NArg() < 2 {
			log.Fatal("usage: migrate force <version>")
		}
		v, err := strconv.Atoi(flag.Arg(1))
		if err != nil {
			log.Fatalf("invalid version: %v", err)
		}
		if err := m.Force(v); err != nil {
			log.Fatalf("migrate force: %v", err)
		}
		log.Printf("migrate force: version set to %d", v)
	case "drop":
		if err := m.Drop(); err != nil {
			log.Fatalf("migrate drop: %v", err)
		}
		log.Println("migrate drop: ok")
	default:
		printUsage()
		os.Exit(1)
	}
}

func closeMigrate(m *migrate.Migrate) {
	sourceErr, dbErr := m.Close()
	if sourceErr != nil {
		log.Printf("migrate close source: %v", sourceErr)
	}
	if dbErr != nil {
		log.Printf("migrate close db: %v", dbErr)
	}
}

func printUsage() {
	fmt.Println(`Usage: go run ./cmd/migrate [-path file://internal/migrations] <command>

Commands:
  up        Apply all pending migrations
  down      Roll back the last migration
  version   Print current migration version
  force N   Set version to N (recovery only)
  drop      Drop all tables (destructive)`)
}
