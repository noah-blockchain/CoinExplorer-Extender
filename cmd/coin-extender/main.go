package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/go-pg/pg"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/nats-io/stan.go"
	"github.com/noah-blockchain/coinExplorer-tools/models"
	"github.com/noah-blockchain/noah-extender/internal/api"
	"github.com/noah-blockchain/noah-extender/internal/core"
	"github.com/noah-blockchain/noah-extender/internal/env"
	noah_node_go_api "github.com/noah-blockchain/noah-node-go-api"
)

const (
	badgerFolder = "db/badger"

	fallbackCount   = 10
	fallbackTimeout = 15 * time.Second
)

func main() {
	envData := env.New()

	if err := runMigrations(envData); err != nil {
		log.Panicln(err)
	}

	extenderApi := api.New(envData.ApiHost, envData.ApiPort)
	go extenderApi.Run()

	db, dbBadger, ns, nodeAPI, err := prepareDependencies(envData)
	if err != nil {
		log.Panicln(err)
	}

	ext := core.NewExtender(envData, db, dbBadger, ns, nodeAPI)
	defer ext.Close()

	ext.Run()
}

func prepareDependencies(env *models.ExtenderEnvironment) (*pg.DB, *badger.DB, stan.Conn, *noah_node_go_api.NoahNodeApi, error) {
	db := pg.Connect(&pg.Options{
		Addr:            fmt.Sprintf("%s:%d", env.DbHost, env.DbPort),
		User:            env.DbUser,
		Password:        env.DbPassword,
		Database:        env.DbName,
		ApplicationName: env.AppName,
		MinIdleConns:    env.DbMinIdleConns,
		PoolSize:        env.DbPoolSize,
		MaxRetries:      10,
		OnConnect: func(conn *pg.Conn) error {
			fmt.Println("Connection with PostgresDB successful created.")
			return nil
		},
	})

	if err := os.MkdirAll(badgerFolder, 0774); err != nil {
		return nil, nil, nil, nil, err
	}
	dbBadger, err := badger.Open(badger.DefaultOptions(badgerFolder))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	ns, err := stan.Connect(
		env.NatsClusterID,
		uuid.New().String(),
		stan.NatsURL(env.NatsAddr),
		stan.Pings(5, 15),
		stan.SetConnectionLostHandler(func(con stan.Conn, reason error) {
			log.Panicln("Connection lost, reason: %v", reason)
		}),
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	nodeApi := noah_node_go_api.NewWithFallbackRetries(env.NodeApi, fallbackCount, fallbackTimeout)
	return db, dbBadger, ns, nodeApi, nil
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
