package storage

import (
	"fmt"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/noah-blockchain/CoinExplorer-Extender/address"
	"github.com/noah-blockchain/CoinExplorer-Extender/coin"
	"github.com/noah-blockchain/CoinExplorer-Extender/transaction"
	"github.com/sirupsen/logrus"
)

type Service struct {
	dbBadger              *badger.DB
	coinRepository        *coin.Repository
	addressRepository     *address.Repository
	transactionRepository *transaction.Repository
	logger                *logrus.Entry
}

func NewService(dbBadger *badger.DB, coinRepository *coin.Repository,
	addressRepository *address.Repository, transactionRepository *transaction.Repository, logger *logrus.Entry) *Service {

	return &Service{
		dbBadger:              dbBadger,
		coinRepository:        coinRepository,
		addressRepository:     addressRepository,
		transactionRepository: transactionRepository,
		logger:                logger,
	}
}

func (s *Service) Run() error {

	err := s.dbBadger.Update(func(txn *badger.Txn) error {

		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {

			item := it.Item()
			k := item.Key()
			fmt.Println("KEY ", string(k))

			keyParts := strings.Split(string(k), "_") // (trx/address)_symbol_(hash/addr)
			if len(keyParts) != 3 {
				_ = txn.Delete(k)
				continue
			}

			if keyParts[0] == "address" {
				addrID, err := s.addressRepository.FindId(keyParts[2])
				if err != nil {
					s.logger.Error(err)
					continue
				}

				if err = s.coinRepository.UpdateCoinOwner(keyParts[1], addrID); err != nil {
					s.logger.Error(err)
					continue
				}
			} else if keyParts[0] == "trx" {
				trxID, err := s.transactionRepository.FindTransactionIdByHash(keyParts[2])
				if err != nil {
					s.logger.Error(err)
					continue
				}

				if err = s.coinRepository.UpdateCoinTransaction(keyParts[1], trxID); err != nil {
					s.logger.Error(err)
					continue
				}
			}

			if err := txn.Delete(k); err != nil {
				s.logger.Panicln(err)
			}
		}
		return nil
	})

	return err
}

func (s *Service) SetValue(key, value string) error {
	if err := s.dbBadger.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), []byte(value))
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) HasValue(key string) error {
	err := s.dbBadger.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			return err
		})
		return err
	})
	return err
}
