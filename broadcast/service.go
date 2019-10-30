package broadcast

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/centrifugal/gocent"
	"github.com/noah-blockchain/CoinExplorer-Extender/address"
	"github.com/noah-blockchain/CoinExplorer-Extender/coin"
	"github.com/noah-blockchain/coinExplorer-tools/helpers"
	"github.com/noah-blockchain/coinExplorer-tools/models"
	"github.com/noah-blockchain/noah-explorer-api/balance"
	"github.com/noah-blockchain/noah-explorer-api/blocks"
	"github.com/noah-blockchain/noah-explorer-api/transaction"
	"github.com/sirupsen/logrus"
)

type Service struct {
	client            *gocent.Client
	ctx               context.Context
	addressRepository *address.Repository
	coinRepository    *coin.Repository
	logger            *logrus.Entry
}

func NewService(env *models.ExtenderEnvironment, addressRepository *address.Repository, coinRepository *coin.Repository,
	logger *logrus.Entry) *Service {
	wsClient := gocent.New(gocent.Config{
		Addr: fmt.Sprintf("%s:%d", env.WsHost, env.WsPort),
		Key:  env.WsKey,
	})

	return &Service{
		client:            wsClient,
		ctx:               context.Background(),
		addressRepository: addressRepository,
		coinRepository:    coinRepository,
		logger:            logger,
	}
}

func (s *Service) PublishBlock(b *models.Block) {
	channel := `blocks`
	msg, err := json.Marshal(new(blocks.Resource).Transform(*b))
	if err != nil {
		s.logger.Error(err)
	}
	s.publish(channel, []byte(msg))
}

func (s *Service) PublishTransactions(transactions []*models.Transaction) {
	channel := `transactions`
	for _, tx := range transactions {
		mTransaction := *tx
		adr, err := s.addressRepository.FindById(tx.FromAddressID)
		mTransaction.FromAddress = &models.Address{Address: adr}
		msg, err := json.Marshal(new(transaction.Resource).Transform(mTransaction))
		if err != nil {
			log.Printf(`Error parse json: %s`, err)
		}
		s.publish(channel, []byte(msg))
	}
}

func (s *Service) PublishBalances(balances []*models.Balance) {

	var mapBalances = make(map[uint64][]interface{})

	for _, item := range balances {
		symbol, err := s.coinRepository.FindSymbolById(item.CoinID)
		if err != nil {
			continue
		}
		adr, err := s.addressRepository.FindById(item.AddressID)
		helpers.HandleError(err)
		mBalance := *item
		mBalance.Address = &models.Address{Address: adr}
		mBalance.Coin = &models.Coin{Symbol: symbol}
		res := new(balance.Resource).Transform(mBalance)
		mapBalances[item.AddressID] = append(mapBalances[item.AddressID], res)
	}

	for addressId, items := range mapBalances {
		adr, err := s.addressRepository.FindById(addressId)
		helpers.HandleError(err)
		channel := "NOAHx" + adr
		msg, err := json.Marshal(items)
		if err != nil {
			log.Printf(`Error parse json: %s`, err)
		}
		s.publish(channel, []byte(msg))
	}
}

func (s *Service) publish(ch string, msg []byte) {
	err := s.client.Publish(s.ctx, ch, msg)
	if err != nil {
		s.logger.Warn(err)
	}
}
