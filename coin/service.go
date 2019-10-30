package coin

import (
	"errors"
	"strconv"
	"time"

	"github.com/noah-blockchain/CoinExplorer-Extender/address"
	"github.com/noah-blockchain/coinExplorer-tools/helpers"
	"github.com/noah-blockchain/coinExplorer-tools/models"
	"github.com/noah-blockchain/noah-node-go-api"
	"github.com/noah-blockchain/noah-node-go-api/responses"
	"github.com/sirupsen/logrus"
)

const (
	updaterWorkerAddressTimeout     = 2 * time.Second
	updaterWorkerTransactionTimeout = 3 * time.Second
)

type Service struct {
	env                   *models.ExtenderEnvironment
	nodeApi               *noah_node_go_api.NoahNodeApi
	repository            *Repository
	addressRepository     *address.Repository
	logger                *logrus.Entry
	jobUpdateCoins        chan []*models.Transaction
	jobUpdateCoinsFromMap chan map[string]struct{}
	//updater               *sync.Map
}

func NewService(env *models.ExtenderEnvironment, nodeApi *noah_node_go_api.NoahNodeApi, repository *Repository,
	addressRepository *address.Repository, logger *logrus.Entry) *Service {
	return &Service{
		env:                   env,
		nodeApi:               nodeApi,
		repository:            repository,
		addressRepository:     addressRepository,
		logger:                logger,
		jobUpdateCoins:        make(chan []*models.Transaction, 1),
		jobUpdateCoinsFromMap: make(chan map[string]struct{}, 1),
		//updater:               new(sync.Map),
	}
}

type CreateCoinData struct {
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	InitialAmount  string `json:"initial_amount"`
	InitialReserve string `json:"initial_reserve"`
	Crr            string `json:"crr"`
}

func (s *Service) GetUpdateCoinsFromTxsJobChannel() chan []*models.Transaction {
	return s.jobUpdateCoins
}

func (s *Service) GetUpdateCoinsFromCoinsMapJobChannel() chan map[string]struct{} {
	return s.jobUpdateCoinsFromMap
}

func (s Service) ExtractCoinsFromTransactions(transactions []responses.Transaction) ([]*models.Coin, error) {
	var coins []*models.Coin
	for _, tx := range transactions {
		if tx.Type == models.TxTypeCreateCoin {
			coin, err := s.ExtractFromTx(tx)
			if err != nil {
				s.logger.Error(err)
				return nil, err
			}
			coins = append(coins, coin)
		}
	}
	return coins, nil
}

func (s *Service) ExtractFromTx(tx responses.Transaction) (*models.Coin, error) {
	if tx.Data == nil {
		s.logger.Warn("empty transaction data")
		return nil, errors.New("no data for creating a coin")
	}
	txData := tx.IData.(models.CreateCoinTxData)

	crr, err := strconv.ParseUint(txData.ConstantReserveRatio, 10, 64)
	if err != nil {
		s.logger.Error(err)
		return nil, err
	}

	coin := &models.Coin{
		Crr:            crr,
		Volume:         txData.InitialAmount,
		ReserveBalance: txData.InitialReserve,
		Name:           txData.Name,
		Symbol:         txData.Symbol,
		DeletedAt:      nil,
		Price:          getTokenPrice(txData.InitialAmount, txData.InitialReserve, crr),
		Delegated:      0,
	}

	//repeatTime := 10
	//for i := 0; i < repeatTime; i++ {
	//	fromId, err := s.addressRepository.FindId(helpers.RemovePrefixFromAddress(tx.From))
	//	if err != nil {
	//		s.logger.Error(err)
	//		time.Sleep(5 * time.Second)
	//		continue
	//	} else {
	//		coin.CreationAddressID = &fromId
	//	}
	//}
	//
	//for i := 0; i < repeatTime; i++ {
	//	fromTxId, err := s.repository.FindTransactionIdByHash(helpers.RemovePrefix(tx.Hash))
	//	if err != nil {
	//		s.logger.Print(err)
	//		time.Sleep(5 * time.Second)
	//		continue
	//	} else {
	//		coin.CreationTransactionID = &fromTxId
	//	}
	//}

	go s.updateCoinAddressInfo(coin.Symbol, helpers.RemovePrefixFromAddress(tx.From))
	go s.updateCoinTransactionInfo(coin.Symbol, helpers.RemovePrefix(tx.Hash))

	return coin, nil
}

func (s *Service) updateCoinAddressInfo(symbol string, address string) {
	ticker := time.NewTicker(updaterWorkerAddressTimeout)

	for { // nolint:gosimple
		select {
		case <-ticker.C:
			fromId, err := s.addressRepository.FindId(address)
			if err != nil {
				s.logger.Error(err)
			} else {
				s.logger.Error("updateCoinAddressInfo fromId: ", fromId)
				if err = s.repository.UpdateCoinOwner(symbol, fromId); err == nil {
					s.logger.Error("updateCoinAddressInfo Stop")
					ticker.Stop()
					break
				}
				s.logger.Error("updateCoinAddressInfo ERROR: ", err)
			}
		}
	}
}

func (s *Service) updateCoinTransactionInfo(symbol string, hash string) {
	ticker := time.NewTicker(updaterWorkerTransactionTimeout)

	for { // nolint:gosimple
		select {
		case <-ticker.C:
			fromTxId, err := s.repository.FindTransactionIdByHash(hash)
			if err != nil {
				s.logger.Error(err)
			} else {
				s.logger.Error("updateCoinTransactionInfo fromTxId: ", fromTxId)
				if err = s.repository.UpdateCoinTransaction(symbol, fromTxId); err == nil {
					s.logger.Error("updateCoinTransactionInfo Stop")
					ticker.Stop()
					break
				}
				s.logger.Error("updateCoinTransactionInfo ERROR: ", err)
			}
		}
	}
}

func (s *Service) CreateNewCoins(coins []*models.Coin) error {
	err := s.repository.SaveAllIfNotExist(coins)
	if err != nil {
		s.logger.Error(err)
	}
	return err
}

func (s *Service) UpdateCoinsInfoFromTxsWorker(jobs <-chan []*models.Transaction) {
	for transactions := range jobs {
		coinsMap := make(map[string]struct{})
		// Find coins in transaction for update
		for _, tx := range transactions {
			symbol, err := s.repository.FindSymbolById(tx.GasCoinID)
			if err != nil {
				s.logger.Error(err)
				continue
			}
			coinsMap[symbol] = struct{}{}
			switch tx.Type {
			case models.TxTypeSellCoin:
				coinsMap[tx.IData.(models.SellCoinTxData).CoinToBuy] = struct{}{}
				coinsMap[tx.IData.(models.SellCoinTxData).CoinToSell] = struct{}{}
			case models.TxTypeBuyCoin:
				coinsMap[tx.IData.(models.BuyCoinTxData).CoinToBuy] = struct{}{}
				coinsMap[tx.IData.(models.BuyCoinTxData).CoinToSell] = struct{}{}
			case models.TxTypeSellAllCoin:
				coinsMap[tx.IData.(models.SellAllCoinTxData).CoinToBuy] = struct{}{}
				coinsMap[tx.IData.(models.SellAllCoinTxData).CoinToSell] = struct{}{}
			}
		}
		s.GetUpdateCoinsFromCoinsMapJobChannel() <- coinsMap
	}
}

func (s Service) UpdateCoinsInfoFromCoinsMap(job <-chan map[string]struct{}) {
	for coinsMap := range job {
		delete(coinsMap, s.env.BaseCoin)
		if len(coinsMap) > 0 {
			coinsForUpdate := make([]string, len(coinsMap))
			i := 0
			for symbol := range coinsMap {
				coinsForUpdate[i] = symbol
				i++
			}
			err := s.UpdateCoinsInfo(coinsForUpdate)
			if err != nil {
				s.logger.Error(err)
			}
		}
	}
}

func (s *Service) UpdateCoinsInfo(symbols []string) error {
	var coins []*models.Coin
	for _, symbol := range symbols {
		if symbol == s.env.BaseCoin {
			continue
		}
		coin, err := s.GetCoinFromNode(symbol)
		if err != nil {
			s.logger.Error(err)
			continue
		}
		coins = append(coins, coin)
	}
	if len(coins) > 0 {
		return s.repository.SaveAllIfNotExist(coins)
	}
	return nil
}

func (s *Service) GetCoinFromNode(symbol string) (*models.Coin, error) {
	coinResp, err := s.nodeApi.GetCoinInfo(symbol)
	if err != nil {
		s.logger.Error(err)
		return nil, err
	}
	now := time.Now()
	coin := new(models.Coin)
	id, err := s.repository.FindIdBySymbol(symbol)
	if err != nil {
		s.logger.Error(err)
		return nil, err
	}
	coin.ID = id
	if coinResp.Error != nil {
		return nil, errors.New(coinResp.Error.Message)
	}
	crr, err := strconv.ParseUint(coinResp.Result.Crr, 10, 64)
	if err != nil {
		s.logger.Error(err)
		return nil, err
	}
	coin.Name = coinResp.Result.Name
	coin.Symbol = coinResp.Result.Symbol
	coin.Crr = crr
	coin.ReserveBalance = coinResp.Result.ReserveBalance
	coin.Volume = coinResp.Result.Volume
	coin.DeletedAt = nil
	coin.UpdatedAt = now
	coin.Delegated = 0
	coin.Price = getTokenPrice(coinResp.Result.Volume, coinResp.Result.ReserveBalance, crr)

	return coin, nil
}