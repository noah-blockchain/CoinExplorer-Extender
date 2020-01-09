package main

import (
	"fmt"
	"log"
	"math"

	"github.com/go-pg/pg"
	"github.com/noah-blockchain/noah-extender/internal/env"
	"github.com/noah-blockchain/noah-extender/internal/validator"
)

func main() {
	envData := env.New()

	db := pg.Connect(&pg.Options{
		Addr:            fmt.Sprintf("%s:%d", envData.DbHost, envData.DbPort),
		User:            envData.DbUser,
		Password:        envData.DbPassword,
		Database:        envData.DbName,
		ApplicationName: envData.AppName,
		MinIdleConns:    envData.DbMinIdleConns,
		PoolSize:        envData.DbPoolSize,
		MaxRetries:      10,
		OnConnect: func(conn *pg.Conn) error {
			fmt.Println("Connection with PostgresDB successful created.")
			return nil
		},
	})

	s := validator.NewRepository(db)
	valid, err := s.FindValidatorById(6)
	if err != nil {
		log.Fatal(err)
		return
	}

	signedCount, err := s.GetFullSignedCountValidatorBlock(valid.ID, valid.CreatedAt)
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(valid.CreatedAt)
	validatorBlocksHeight, err := s.GetCountBlockFromDate(valid.CreatedAt)
	if err != nil {
		log.Fatal(err)
		return
	}

	var value float64
	if validatorBlocksHeight > 0 {
		value = float64(signedCount / validatorBlocksHeight)
	}

	var uptime = math.Min(value*100, 100.0)
	if err = s.UpdateValidatorUptime(6, uptime); err != nil {
		log.Fatal(err)
		return
	}

	log.Println("OK")
}
