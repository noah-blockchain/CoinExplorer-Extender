package main

import (
	"github.com/noah-blockchain/noah-explorer-extender/api"
	"github.com/noah-blockchain/noah-explorer-extender/core"
	"github.com/noah-blockchain/noah-explorer-extender/database/migrate"
	"github.com/noah-blockchain/noah-explorer-extender/env"
)

func main() {
	envData := env.New()

	migrate.Migrate(envData)

	extenderApi := api.New(envData.ApiHost, envData.ApiPort)
	go extenderApi.Run()
	ext := core.NewExtender(envData)
	ext.Run()
}
