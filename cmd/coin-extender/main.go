package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
	"github.com/noah-blockchain/coinExplorer-tools/models"
	"github.com/noah-blockchain/noah-extender/internal/api"
	"github.com/noah-blockchain/noah-extender/internal/core"
	"github.com/noah-blockchain/noah-extender/internal/env"
)

func main() {
	envData := env.New()

	if err := runMigrations(envData); err != nil {
		log.Panicln(err)
	}

	extenderApi := api.New(envData.ApiHost, envData.ApiPort)
	go extenderApi.Run()

	ext := core.NewExtender(envData)
	ext.Run()
}

func runMigrations(envData *models.ExtenderEnvironment) error {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		envData.DbHost, envData.DbPort, envData.DbUser, envData.DbPassword, envData.DbName)
	db, err := sql.Open(
		"postgres",
		psqlInfo,
	)
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	fsrc, err := (&file.File{}).Open("file://migrations")
	if err != nil {
		log.Printf("Cannot open migrations file: %s", err)
		return err
	}

	m, err := migrate.NewWithInstance(
		"file",
		fsrc,
		"postgres",
		driver)
	if err != nil {
		log.Printf("Cannot create migrate instance: %s", err)
		return err
	}

	_ = m.Steps(1)
	return nil
}
