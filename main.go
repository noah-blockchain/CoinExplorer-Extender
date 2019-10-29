package main

import (
	"github.com/noah-blockchain/CoinExplorer-Extender/api"
	"github.com/noah-blockchain/CoinExplorer-Extender/core"
	"github.com/noah-blockchain/CoinExplorer-Extender/database/migrate"
	"github.com/noah-blockchain/CoinExplorer-Extender/env"
)

func main() {
	envData := env.New()

	migrate.Migrate(envData)

	extenderApi := api.New(envData.ApiHost, envData.ApiPort)
	go extenderApi.Run()
	ext := core.NewExtender(envData)
	ext.Run()
}
