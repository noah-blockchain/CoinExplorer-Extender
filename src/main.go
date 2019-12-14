package main

import (
	"fmt"
	"log"

	"database/sql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/noah-blockchain/coinExplorer-tools/models"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
	"github.com/noah-blockchain/CoinExplorer-Extender/api"
	"github.com/noah-blockchain/CoinExplorer-Extender/core"
	"github.com/noah-blockchain/CoinExplorer-Extender/env"
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
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		envData.DbUser, envData.DbPort, envData.DbUser, envData.DbPassword, envData.DbName)
	db, err := sql.Open(
		"postgres",
		psqlInfo,
	)
	if err != nil {
		return err
	}
	defer db.Close()

	driver, _ := postgres.WithInstance(db, &postgres.Config{})
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
	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Printf("Migration error: %s", err)
		return err
	}
	return nil
}
