package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/storage/postgresql"
	"github.com/jmoiron/sqlx"
	goose "github.com/pressly/goose/v3"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Migrate - утилита для управления миграциями БД сервиса Anti-Bruteforce (на основе goose)\n\n"+
				"вызов: migrate -config=<config file name> -dir=<migration dir> -command=<migration command> arg\n\n"+
				"Примеры:\n"+
				"  migrate -config=config.yaml -dir=./migrations -command up 0002\n"+
				"  migrate -command status\n\n"+
				"Доступные флаги:\n")
		flag.PrintDefaults()
	}
}

func main() {
	var (
		configFile    string
		migrationsDir string
		command       string
		arg           string
	)

	flag.StringVar(&configFile, "config", "config.yaml", "path to config file")
	flag.StringVar(&migrationsDir, "dir", "./migrations", "path to migrations dir")
	flag.StringVar(&command, "command", "up", "goose command: up|down|redo|reset|status|version|up-to|down-to")
	flag.StringVar(&arg, "arg", "", "argument for command (version for up-to/down-to/force)")
	flag.Parse()

	if err := runMigration(configFile, migrationsDir, command, arg); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	fmt.Println("migration completed successfully")
}

func runMigration(configFile, migrationsDir, command, arg string) error {
	var (
		db  *sql.DB
		dbx *sqlx.DB
		v   int64
		err error
	)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	// Загружаем конфиг для подключения к БД
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	// Подключаемся к БД
	if dbx, err = postgresql.OpenDB(cfg.Database); err != nil {
		return fmt.Errorf("DB open error: %w", err)
	}

	defer dbx.Close()
	db = dbx.DB

	cmd := strings.ToLower(command)
	switch cmd {
	case "up":
		err = goose.Up(db, migrationsDir)
	case "down":
		err = goose.Down(db, migrationsDir)
	case "redo":
		err = goose.Redo(db, migrationsDir)
	case "reset":
		err = goose.Reset(db, migrationsDir)
	case "status":
		err = goose.Status(db, migrationsDir)
	case "version":
		v, err2 := goose.GetDBVersion(db)
		if err2 != nil {
			return fmt.Errorf("get DB version: %w", err2)
		}
		fmt.Printf("Current version: %d\n", v)
		return nil
	case "up-to":
		if v, err = mustParseInt64(arg); err != nil {
			return fmt.Errorf("parse version error: %w", err)
		}
		err = goose.UpTo(db, migrationsDir, v)
	case "down-to":
		if v, err = mustParseInt64(arg); err != nil {
			return fmt.Errorf("parse version error: %w", err)
		}
		err = goose.DownTo(db, migrationsDir, v)

	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	if err != nil {
		return fmt.Errorf("migration error: %w", err)
	}

	return nil
}

// Проверка корректности номера версии. Должен конвертироваться в int64.
func mustParseInt64(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("arg is required for this command")
	}
	var v int64
	_, err := fmt.Sscan(s, &v)
	if err != nil {
		return 0, err
	}
	return v, nil
}
