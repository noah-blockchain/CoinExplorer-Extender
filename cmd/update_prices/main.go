package main

import (
	"fmt"
	"log"

	"github.com/go-pg/pg"
	coin2 "github.com/noah-blockchain/CoinExplorer-Extender/coin"
	"github.com/noah-blockchain/coinExplorer-tools/models"
)

func main() {
	db := pg.Connect(&pg.Options{
		Addr:            "",
		User:            "",
		Password:        "",
		Database:        "",
		ApplicationName: "Coin Extender",
		MinIdleConns:    10,
		PoolSize:        10,
		MaxRetries:      10,
	})

	var coins []*models.Coin
	err := db.Model(&coins).Order("symbol ASC").Select()
	if err != nil {
		log.Panicln(err)
	}

	if len(coins) == 0 {
		log.Panic("its bad 1")
	}

	for i, coin := range coins {
		fmt.Println("---------------------------------")
		fmt.Println(coin.Symbol)
		fmt.Println(coin.Price)
		coins[i].Price = coin2.GetTokenPrice(coin.Volume, coin.ReserveBalance, coin.Crr)
		fmt.Println(coins[i].Price)
		fmt.Println("---------------------------------")
	}

	_, err = db.Model(&coins).OnConflict("(symbol) DO UPDATE").Insert()
	if err != nil {
		log.Panic(err)
	}
}
