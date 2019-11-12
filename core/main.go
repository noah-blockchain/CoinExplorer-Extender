package core

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/go-pg/pg"
	"github.com/noah-blockchain/CoinExplorer-Extender/address"
	"github.com/noah-blockchain/CoinExplorer-Extender/balance"
	"github.com/noah-blockchain/CoinExplorer-Extender/block"
	"github.com/noah-blockchain/CoinExplorer-Extender/coin"
	"github.com/noah-blockchain/CoinExplorer-Extender/events"
	"github.com/noah-blockchain/CoinExplorer-Extender/transaction"
	"github.com/noah-blockchain/CoinExplorer-Extender/validator"
	"github.com/noah-blockchain/coinExplorer-tools/helpers"
	"github.com/noah-blockchain/coinExplorer-tools/models"
	"github.com/noah-blockchain/noah-node-go-api"
	"github.com/noah-blockchain/noah-node-go-api/responses"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

const (
	ChasingModDiff = 2

	badgerFolder = "db/badger"

	fallbackCount   = 10
	fallbackTimeout = 15 * time.Second
)

type Extender struct {
	env                 *models.ExtenderEnvironment
	nodeApi             *noah_node_go_api.NoahNodeApi
	blockService        *block.Service
	addressService      *address.Service
	blockRepository     *block.Repository
	validatorService    *validator.Service
	validatorRepository *validator.Repository
	transactionService  *transaction.Service
	eventService        *events.Service
	balanceService      *balance.Service
	coinService         *coin.Service
	chasingMode         bool
	currentNodeHeight   uint64
	logger              *logrus.Entry
	dbCoinWorker        *badger.DB
}

type dbLogger struct {
	logger *logrus.Entry
}

func (d dbLogger) BeforeQuery(q *pg.QueryEvent) {}

func (d dbLogger) AfterQuery(q *pg.QueryEvent) {
	d.logger.Info(q.FormattedQuery())
}

func NewExtender(env *models.ExtenderEnvironment) *Extender {
	//Init Logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetReportCaller(true)

	if env.Debug {
		logger.SetFormatter(&logrus.TextFormatter{
			DisableColors: false,
			FullTimestamp: true,
		})
	} else {
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetLevel(logrus.WarnLevel)
	}

	contextLogger := logger.WithFields(logrus.Fields{
		"version": "2.1.0",
		"app":     "Coin Explorer Extender",
	})

	//Init DB
	db := pg.Connect(&pg.Options{
		Addr:            fmt.Sprintf("%s:%d", env.DbHost, env.DbPort),
		User:            env.DbUser,
		Password:        env.DbPassword,
		Database:        env.DbName,
		ApplicationName: env.AppName,
		MinIdleConns:    env.DbMinIdleConns,
		PoolSize:        env.DbPoolSize,
		MaxRetries:      10,
	})

	if env.Debug {
		db.AddQueryHook(dbLogger{logger: contextLogger})
	}
	//api
	nodeApi := noah_node_go_api.NewWithFallbackRetries(env.NodeApi, fallbackCount, fallbackTimeout)

	// Repositories
	blockRepository := block.NewRepository(db)
	validatorRepository := validator.NewRepository(db)
	transactionRepository := transaction.NewRepository(db)
	addressRepository := address.NewRepository(db)
	coinRepository := coin.NewRepository(db)
	eventsRepository := events.NewRepository(db)
	balanceRepository := balance.NewRepository(db)

	// Services
	balanceService := balance.NewService(env, balanceRepository, nodeApi, addressRepository, coinRepository, contextLogger)

	if err := os.MkdirAll(badgerFolder, 0774); err != nil {
		logger.Panicln(err)
	}
	dbCoinWorker, err := badger.Open(badger.DefaultOptions(badgerFolder))
	if err != nil {
		logger.Panicln(err)
	}
	coinService := coin.NewService(env, nodeApi, coinRepository, addressRepository, contextLogger, dbCoinWorker)
	return &Extender{
		env:                 env,
		nodeApi:             nodeApi,
		blockService:        block.NewBlockService(blockRepository, validatorRepository),
		eventService:        events.NewService(env, eventsRepository, validatorRepository, addressRepository, coinRepository, coinService, balanceRepository, contextLogger),
		blockRepository:     blockRepository,
		validatorService:    validator.NewService(env, nodeApi, validatorRepository, addressRepository, coinRepository, contextLogger),
		transactionService:  transaction.NewService(env, transactionRepository, addressRepository, validatorRepository, coinRepository, coinService, contextLogger),
		addressService:      address.NewService(env, addressRepository, balanceService.GetAddressesChannel(), contextLogger),
		validatorRepository: validatorRepository,
		balanceService:      balanceService,
		coinService:         coinService,
		chasingMode:         true,
		currentNodeHeight:   0,
		logger:              contextLogger,
		dbCoinWorker:        dbCoinWorker,
	}
}

func (ext *Extender) coinWorker() {
	for {

		err := ext.dbCoinWorker.Update(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = false
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				fmt.Println("KEY ", string(k))
				s := strings.Split(string(k), "_") // (trx/address)_symbol_(hash/addr)
				if len(s) != 3 {
					_ = txn.Delete(k)
					continue
				}

				if s[0] == "address" {
					addrID, err := ext.addressService.FindId(s[2])
					if err != nil {
						ext.logger.Error(err)
						continue
					}

					if err = ext.coinService.UpdateCoinOwner(s[1], addrID); err != nil {
						ext.logger.Error(err)
						continue
					}
				} else if s[0] == "trx" {
					trxID, err := ext.transactionService.FindTransactionIdByHash(s[2])
					if err != nil {
						ext.logger.Error(err)
						continue
					}

					if err = ext.coinService.UpdateCoinTransaction(s[1], trxID); err != nil {
						ext.logger.Error(err)
						continue
					}
				}

				if err := txn.Delete(k); err != nil {
					ext.logger.Panicln(err)
				}
			}
			return nil
		})

		if err != nil {
			ext.logger.Error(err)
		}

		ext.logger.Println("Next attempt")
		time.Sleep(5 * time.Second)
	}
}

func (ext *Extender) Run() {
	//check connections to node
	_, err := ext.nodeApi.GetStatus()
	if err == nil {
		err = ext.blockRepository.DeleteLastBlockData()
	} else {
		ext.logger.Error(err)
	}
	helpers.HandleError(err)

	var height uint64

	// ----- Workers -----
	ext.runWorkers()

	lastExplorerBlock, _ := ext.blockRepository.GetLastFromDB()

	if lastExplorerBlock != nil {
		height = lastExplorerBlock.ID + 1
		ext.blockService.SetBlockCache(lastExplorerBlock)
	} else {
		height = 1
	}

	//height = 645764

	for {

		start := time.Now()
		ext.findOutChasingMode(height)
		//Pulling block data
		blockResponse, err := ext.nodeApi.GetBlock(height)
		helpers.HandleError(err)
		if blockResponse.Error != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		//Pulling events
		eventsResponse, err := ext.nodeApi.GetBlockEvents(height)
		if err != nil {
			ext.logger.Error(err)
		}
		helpers.HandleError(err)

		ext.handleAddressesFromResponses(blockResponse, eventsResponse)
		ext.handleBlockResponse(blockResponse)
		ext.handleCoinsFromTransactions(blockResponse.Result.Transactions)

		if height%uint64(ext.env.RewardAggregateEveryBlocksCount) == 0 {
			go ext.eventService.AggregateRewards(ext.env.RewardAggregateTimeInterval, height)
		}
		go ext.handleEventResponse(height, eventsResponse)

		height++

		elapsed := time.Since(start)
		ext.logger.Info("Processing time: ", elapsed)
	}
}

func (ext *Extender) runWorkers() {

	// Addresses
	for w := 1; w <= ext.env.WrkSaveAddressesCount; w++ {
		go ext.addressService.SaveAddressesWorker(ext.addressService.GetSaveAddressesJobChannel())
	}

	// Transactions
	for w := 1; w <= ext.env.WrkSaveTxsCount; w++ {
		go ext.transactionService.SaveTransactionsWorker(ext.transactionService.GetSaveTxJobChannel())
	}
	for w := 1; w <= ext.env.WrkSaveTxsOutputCount; w++ {
		go ext.transactionService.SaveTransactionsOutputWorker(ext.transactionService.GetSaveTxsOutputJobChannel())
	}
	for w := 1; w <= ext.env.WrkSaveInvTxsCount; w++ {
		go ext.transactionService.SaveInvalidTransactionsWorker(ext.transactionService.GetSaveInvalidTxsJobChannel())
	}
	go ext.transactionService.UpdateTxsIndexWorker()

	// Validators
	for w := 1; w <= ext.env.WrkSaveValidatorTxsCount; w++ {
		go ext.transactionService.SaveTxValidatorWorker(ext.transactionService.GetSaveTxValidatorJobChannel())
	}
	//обновляет награды валидаторов
	go ext.validatorService.UpdateValidatorsWorker(ext.validatorService.GetUpdateValidatorsJobChannel())

	//обновляет стейки валидаторов
	go ext.validatorService.UpdateStakesWorker(ext.validatorService.GetUpdateStakesJobChannel())

	// Events
	for w := 1; w <= ext.env.WrkSaveRewardsCount; w++ {
		go ext.eventService.SaveRewardsWorker(ext.eventService.GetSaveRewardsJobChannel())
	}
	for w := 1; w <= ext.env.WrkSaveSlashesCount; w++ {
		go ext.eventService.SaveSlashesWorker(ext.eventService.GetSaveSlashesJobChannel())
	}

	// Balances
	go ext.balanceService.Run()
	for w := 1; w <= ext.env.WrkGetBalancesFromNodeCount; w++ {
		go ext.balanceService.GetBalancesFromNodeWorker(ext.balanceService.GetBalancesFromNodeChannel(), ext.balanceService.GetUpdateBalancesJobChannel())
	}
	for w := 1; w <= ext.env.WrkUpdateBalanceCount; w++ {
		go ext.balanceService.UpdateBalancesWorker(ext.balanceService.GetUpdateBalancesJobChannel())
	}

	//Coins
	go ext.coinService.UpdateCoinsInfoFromTxsWorker(ext.coinService.GetUpdateCoinsFromTxsJobChannel())
	go ext.coinService.UpdateCoinsInfoFromCoinsMap(ext.coinService.GetUpdateCoinsFromCoinsMapJobChannel())
	go ext.coinWorker()
}

func (ext *Extender) handleAddressesFromResponses(blockResponse *responses.BlockResponse, eventsResponse *responses.EventsResponse) {
	err := ext.addressService.HandleResponses(blockResponse, eventsResponse)
	if err != nil {
		ext.logger.Error(err)
	}
	helpers.HandleError(err)
}

func (ext *Extender) handleBlockResponse(response *responses.BlockResponse) {
	// Save validators if not exist
	validators, err := ext.validatorService.HandleBlockResponse(response)
	if err != nil {
		ext.logger.Error(err)
	}
	helpers.HandleError(err)

	// Save block
	err = ext.blockService.HandleBlockResponse(response)
	if err != nil {
		ext.logger.Error(err)
	}
	helpers.HandleError(err)

	ext.linkBlockValidator(*response)

	//first block don't have validators
	if response.Result.TxCount != "0" && len(validators) > 0 {
		ext.handleTransactions(response, validators)
	}

	height, err := strconv.ParseUint(response.Result.Height, 10, 64)
	if err != nil {
		ext.logger.Error(err)
	}
	helpers.HandleError(err)

	// No need to update candidate and stakes at the same time
	// Candidate will be updated in the next iteration
	if height%12 == 0 {
		ext.validatorService.GetUpdateStakesJobChannel() <- height
	} else if height > 1 {
		ext.validatorService.GetUpdateValidatorsJobChannel() <- height
	}
}

func (ext *Extender) handleCoinsFromTransactions(transactions []responses.Transaction) {
	if len(transactions) > 0 {
		coins, err := ext.coinService.ExtractCoinsFromTransactions(transactions)
		if err != nil {
			ext.logger.Error(err)
			helpers.HandleError(err)
		}
		if len(coins) > 0 {
			err = ext.coinService.CreateNewCoins(coins)
			if err != nil {
				ext.logger.Error(err)
				helpers.HandleError(err)
			}
		}
	}
}

func (ext *Extender) handleTransactions(response *responses.BlockResponse, validators []*models.Validator) {
	height, err := strconv.ParseUint(response.Result.Height, 10, 64)
	if err != nil {
		ext.logger.Error(err)
	}
	helpers.HandleError(err)
	chunksCount := int(math.Ceil(float64(len(response.Result.Transactions)) / float64(ext.env.TxChunkSize)))
	for i := 0; i < chunksCount; i++ {
		start := ext.env.TxChunkSize * i
		end := start + ext.env.TxChunkSize
		if end > len(response.Result.Transactions) {
			end = len(response.Result.Transactions)
		}
		ext.saveTransactions(height, response.Result.Time, response.Result.Transactions[start:end])
	}
}

func (ext *Extender) handleEventResponse(blockHeight uint64, response *responses.EventsResponse) {
	if len(response.Result.Events) > 0 {
		//Save events
		err := ext.eventService.HandleEventResponse(blockHeight, response)
		if err != nil {
			ext.logger.Error(err)
		}
		helpers.HandleError(err)
	}
}

func (ext *Extender) linkBlockValidator(response responses.BlockResponse) {
	if response.Result.Height == "1" {
		return
	}
	var links []*models.BlockValidator
	height, err := strconv.ParseUint(response.Result.Height, 10, 64)
	if err != nil {
		ext.logger.Error(err)
	}
	helpers.HandleError(err)
	for _, v := range response.Result.Validators {
		vId, err := ext.validatorRepository.FindIdByPk(helpers.RemovePrefix(v.PubKey))
		if err != nil {
			ext.logger.Error(err)
		}
		helpers.HandleError(err)
		link := models.BlockValidator{
			ValidatorID: vId,
			BlockID:     height,
			Signed:      *v.Signed,
		}
		links = append(links, &link)
	}
	err = ext.blockRepository.LinkWithValidators(links)
	if err != nil {
		ext.logger.Error(err)
	}
	helpers.HandleError(err)
}

func (ext *Extender) saveTransactions(blockHeight uint64, blockCreatedAt time.Time, transactions []responses.Transaction) {
	// Save transactions
	err := ext.transactionService.HandleTransactionsFromBlockResponse(blockHeight, blockCreatedAt, transactions)
	if err != nil {
		ext.logger.Error(err)
	}
	helpers.HandleError(err)
}

func (ext *Extender) getNodeLastBlockId() (uint64, error) {
	statusResponse, err := ext.nodeApi.GetStatus()
	if err != nil {
		ext.logger.Error(err)
		return 0, err
	}
	return strconv.ParseUint(statusResponse.Result.LatestBlockHeight, 10, 64)
}

func (ext *Extender) findOutChasingMode(height uint64) {
	var err error
	if ext.currentNodeHeight == 0 {
		ext.currentNodeHeight, err = ext.getNodeLastBlockId()
		if err != nil {
			ext.logger.Error(err)
		}
		helpers.HandleError(err)
	}
	isChasingMode := ext.currentNodeHeight-height > ChasingModDiff
	if ext.chasingMode && !isChasingMode {
		ext.currentNodeHeight, err = ext.getNodeLastBlockId()
		if err != nil {
			ext.logger.Error(err)
		}
		helpers.HandleError(err)
		ext.chasingMode = ext.currentNodeHeight-height > ChasingModDiff
	}
}

func (egu *Extender) Do() error {
	egu.logger.Info("Getting genesis data...")
	url := egu.env.NodeApi + "/genesis"
	_, data, err := fasthttp.Get(nil, url)
	helpers.HandleError(err)
	genesisResponse := new(GenesisResponse)
	err = json.Unmarshal(data, genesisResponse)
	helpers.HandleError(err)
	egu.logger.Info("Genesis data has been downloading")

	genesis := &genesisResponse.Result.Genesis

	egu.logger.Info("Extracting addresses...")
	addresses, err := egu.extractAddresses(genesis)
	helpers.HandleError(err)
	msg := fmt.Sprintf("%d addresses have been extracting", len(addresses))
	egu.logger.Info(msg)

	egu.logger.Info("Extracting coins...")
	coins, err := egu.extractCoins(genesis)
	helpers.HandleError(err)
	msg = fmt.Sprintf("%d coins has been extracting", len(coins))
	egu.logger.Info(msg)

	wg := new(sync.WaitGroup)
	wg.Add(2)
	go egu.saveAddresses(addresses, wg)
	go egu.saveCoins(coins, wg)
	wg.Wait()

	egu.logger.Info("Extracting validators...")
	validators, err := egu.extractCandidates(genesis)
	helpers.HandleError(err)
	msg = fmt.Sprintf("%d validators have been extracting", len(validators))
	egu.logger.Info(msg)
	err = egu.saveCandidates(validators)
	helpers.HandleError(err)
	egu.logger.Info("Validators has been uploading")

	egu.logger.Info("Extracting balances...")
	balances, err := egu.extractBalances(genesis)
	helpers.HandleError(err)
	msg = fmt.Sprintf("%d balances has been extracting", len(balances))
	egu.logger.Info(msg)
	err = egu.saveBalances(balances)
	helpers.HandleError(err)
	egu.logger.Info("Balances has been uploading")

	egu.logger.Info("Extracting stakes...")
	stakes, err := egu.extractStakes(genesis)
	helpers.HandleError(err)
	msg = fmt.Sprintf("%d stakes have been extracting", len(stakes))
	egu.logger.Info(msg)
	err = egu.saveStakes(stakes)
	helpers.HandleError(err)
	egu.logger.Info("Stakes has been uploading")

	egu.logger.Info("Upload complete")
	return err
}

func (egu *Extender) extractAddresses(genesis *Genesis) ([]string, error) {
	addressesMap := make(map[string]struct{})
	for _, val := range genesis.AppState.Validators {
		addressesMap[helpers.RemovePrefixFromAddress(val.RewardAddress)] = struct{}{}
	}
	for _, candidate := range genesis.AppState.Candidates {
		addressesMap[helpers.RemovePrefixFromAddress(candidate.RewardAddress)] = struct{}{}
		addressesMap[helpers.RemovePrefixFromAddress(candidate.OwnerAddress)] = struct{}{}
		for _, stake := range candidate.Stakes {
			addressesMap[helpers.RemovePrefixFromAddress(stake.Owner)] = struct{}{}
		}
	}
	for _, account := range genesis.AppState.Accounts {
		addressesMap[helpers.RemovePrefixFromAddress(account.Address)] = struct{}{}
	}
	var addresses = make([]string, len(addressesMap))
	i := 0
	for adr := range addressesMap {
		addresses[i] = adr
		i++
	}
	return addresses, nil
}

func (egu *Extender) extractCoins(genesis *Genesis) ([]*models.Coin, error) {
	var coins = make([]*models.Coin, len(genesis.AppState.Coins))
	i := 0
	for _, c := range genesis.AppState.Coins {
		crr, err := strconv.ParseUint(c.Crr, 10, 64)
		if err != nil {
			egu.logger.Error(err)
		}
		coins[i] = &models.Coin{
			Name:           c.Name,
			Symbol:         c.Symbol,
			Crr:            crr,
			Volume:         c.Volume,
			ReserveBalance: c.ReserveBalance,
			UpdatedAt:      time.Now(),
		}
		i++
	}
	return coins, nil
}

func (egu *Extender) extractStakes(genesis *Genesis) ([]*models.Stake, error) {
	var stakes []*models.Stake
	for _, candidate := range genesis.AppState.Candidates {
		for _, stake := range candidate.Stakes {
			coinId, err := egu.coinRepository.FindIdBySymbol(stake.Coin)
			if err != nil {
				egu.logger.Error(err)
			}
			ownerId, err := egu.addressRepository.FindId(helpers.RemovePrefixFromAddress(stake.Owner))
			if err != nil {
				egu.logger.Error(err)
			}
			validatorId, err := egu.validatorRepository.FindIdByPk(helpers.RemovePrefix(candidate.PubKey))
			if err != nil {
				egu.logger.Error(err)
			}
			stakes = append(stakes, &models.Stake{
				CoinID:         coinId,
				OwnerAddressID: ownerId,
				ValidatorID:    validatorId,
				Value:          stake.Value,
				NoahValue:      stake.NoahValue,
			})
		}
	}
	return stakes, nil
}

func (egu *Extender) extractBalances(genesis *Genesis) ([]*models.Balance, error) {
	var balances []*models.Balance
	for _, account := range genesis.AppState.Accounts {
		addressId, err := egu.addressRepository.FindId(helpers.RemovePrefixFromAddress(account.Address))
		if err != nil {
			egu.logger.Error(err)
		}
		for _, bls := range account.Balance {
			coinId, err := egu.coinRepository.FindIdBySymbol(bls.Coin)
			if err != nil {
				egu.logger.Error(err)
			}
			balances = append(balances, &models.Balance{
				CoinID:    coinId,
				AddressID: addressId,
				Value:     bls.Value,
			})
		}
	}
	return balances, nil
}

func (egu Extender) extractCandidates(genesis *Genesis) ([]*models.Validator, error) {
	var validators []*models.Validator
	for _, candidate := range genesis.AppState.Candidates {
		ownerAddress, err := egu.addressRepository.FindId(helpers.RemovePrefixFromAddress(candidate.OwnerAddress))
		if err != nil {
			egu.logger.Error(err)
		}
		rewardAddress, err := egu.addressRepository.FindId(helpers.RemovePrefixFromAddress(candidate.RewardAddress))
		if err != nil {
			egu.logger.Error(err)
		}
		status := uint8(candidate.Status)
		commission, err := strconv.ParseUint(candidate.Commission, 10, 64)
		stake := candidate.TotalNoahStake
		validators = append(append(validators, &models.Validator{
			OwnerAddressID:  &ownerAddress,
			RewardAddressID: &rewardAddress,
			PublicKey:       helpers.RemovePrefix(candidate.PubKey),
			Status:          &status,
			Commission:      &commission,
			TotalStake:      &stake,
		}))
	}

	return validators, nil
}

func (egu *Extender) saveAddresses(addresses []string, wg *sync.WaitGroup) {
	if len(addresses) > 0 {
		wgAddresses := new(sync.WaitGroup)
		chunksCount := int(math.Ceil(float64(len(addresses)) / float64(egu.env.AddrChunkSize)))
		for i := 0; i < chunksCount; i++ {
			start := egu.env.AddrChunkSize * i
			end := start + egu.env.AddrChunkSize
			if end > len(addresses) {
				end = len(addresses)
			}
			wgAddresses.Add(1)
			go func() {
				err := egu.addressRepository.SaveAllIfNotExist(addresses[start:end])
				helpers.HandleError(err)
				wgAddresses.Done()
			}()
		}
		wgAddresses.Wait()
	}
	wg.Done()
	egu.logger.Info("Addresses has been uploading")
}

func (egu *Extender) saveCoins(coins []*models.Coin, wg *sync.WaitGroup) {
	if len(coins) > 0 {
		wgCoins := new(sync.WaitGroup)
		chunksCount := int(math.Ceil(float64(len(coins)) / float64(egu.env.TxChunkSize)))
		for i := 0; i < chunksCount; i++ {
			start := egu.env.TxChunkSize * i
			end := start + egu.env.TxChunkSize
			if end > len(coins) {
				end = len(coins)
			}
			wgCoins.Add(1)
			go func() {
				err := egu.coinRepository.SaveAllIfNotExist(coins[start:end])
				helpers.HandleError(err)
				wgCoins.Done()
			}()
		}
		wgCoins.Wait()
	}
	wg.Done()
	egu.logger.Info("Coins has been uploading")
}

func (egu *Extender) saveCandidates(validators []*models.Validator) error {
	if len(validators) > 0 {
		wgCandidates := new(sync.WaitGroup)
		chunksCount := int(math.Ceil(float64(len(validators)) / float64(egu.env.TxChunkSize)))
		for i := 0; i < chunksCount; i++ {
			start := egu.env.TxChunkSize * i
			end := start + egu.env.TxChunkSize
			if end > len(validators) {
				end = len(validators)
			}
			wgCandidates.Add(1)
			go func() {
				err := egu.validatorRepository.SaveAllIfNotExist(validators[start:end])
				helpers.HandleError(err)
				wgCandidates.Done()
			}()
		}
		wgCandidates.Wait()
	}
	return nil
}

func (egu *Extender) saveBalances(balances []*models.Balance) error {
	if len(balances) > 0 {
		wgBalances := new(sync.WaitGroup)
		chunksCount := int(math.Ceil(float64(len(balances)) / float64(egu.env.TxChunkSize)))
		for i := 0; i < chunksCount; i++ {
			start := egu.env.TxChunkSize * i
			end := start + egu.env.TxChunkSize
			if end > len(balances) {
				end = len(balances)
			}
			wgBalances.Add(1)
			go func() {
				err := egu.balanceRepository.SaveAll(balances[start:end])
				if err != nil {
					egu.logger.Error(err)
				}
				wgBalances.Done()
			}()
			wgBalances.Wait()
		}
	}
	return nil
}

func (egu *Extender) saveStakes(stakes []*models.Stake) error {
	if len(stakes) > 0 {
		wgStakes := new(sync.WaitGroup)
		chunksCount := int(math.Ceil(float64(len(stakes)) / float64(egu.env.TxChunkSize)))
		for i := 0; i < chunksCount; i++ {
			start := egu.env.TxChunkSize * i
			end := start + egu.env.TxChunkSize
			if end > len(stakes) {
				end = len(stakes)
			}
			wgStakes.Add(1)
			go func() {
				err := egu.validatorRepository.SaveAllStakes(stakes[start:end])
				if err != nil {
					egu.logger.Error(err)
				}
				wgStakes.Done()
			}()
			wgStakes.Wait()
		}
	}
	return nil
}
