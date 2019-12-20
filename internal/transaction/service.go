package transaction

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/noah-blockchain/coinExplorer-tools/helpers"
	"github.com/noah-blockchain/coinExplorer-tools/models"
	node_models "github.com/noah-blockchain/noah-explorer-tools/models"
	"github.com/noah-blockchain/noah-extender/internal/address"
	"github.com/noah-blockchain/noah-extender/internal/coin"
	"github.com/noah-blockchain/noah-extender/internal/validator"
	"github.com/noah-blockchain/noah-go-node/core/check"
	"github.com/noah-blockchain/noah-node-go-api/responses"
	"github.com/sirupsen/logrus"
)

type Service struct {
	env                 *models.ExtenderEnvironment
	txRepository        *Repository
	addressRepository   *address.Repository
	validatorRepository *validator.Repository
	coinRepository      *coin.Repository
	coinService         *coin.Service
	jobSaveTxs          chan []*models.Transaction
	jobSaveTxsOutput    chan []*models.Transaction
	jobSaveValidatorTxs chan []*models.TransactionValidator
	jobSaveInvalidTxs   chan []*models.InvalidTransaction
	logger              *logrus.Entry
}

func NewService(env *models.ExtenderEnvironment, repository *Repository, addressRepository *address.Repository,
	validatorRepository *validator.Repository, coinRepository *coin.Repository, coinService *coin.Service, logger *logrus.Entry) *Service {
	return &Service{
		env:                 env,
		txRepository:        repository,
		coinRepository:      coinRepository,
		addressRepository:   addressRepository,
		coinService:         coinService,
		validatorRepository: validatorRepository,
		jobSaveTxs:          make(chan []*models.Transaction, env.WrkSaveTxsCount),
		jobSaveTxsOutput:    make(chan []*models.Transaction, env.WrkSaveTxsOutputCount),
		jobSaveValidatorTxs: make(chan []*models.TransactionValidator, env.WrkSaveValidatorTxsCount),
		jobSaveInvalidTxs:   make(chan []*models.InvalidTransaction, env.WrkSaveInvTxsCount),
		logger:              logger,
	}
}

func (s *Service) GetSaveTxJobChannel() chan []*models.Transaction {
	return s.jobSaveTxs
}
func (s *Service) GetSaveTxsOutputJobChannel() chan []*models.Transaction {
	return s.jobSaveTxsOutput
}
func (s *Service) GetSaveInvalidTxsJobChannel() chan []*models.InvalidTransaction {
	return s.jobSaveInvalidTxs
}
func (s *Service) GetSaveTxValidatorJobChannel() chan []*models.TransactionValidator {
	return s.jobSaveValidatorTxs
}

//Handle response and save block to DB
func (s *Service) HandleTransactionsFromBlockResponse(blockHeight uint64, blockCreatedAt time.Time,
	transactions []responses.Transaction) error {

	var txList []*models.Transaction
	var invalidTxList []*models.InvalidTransaction

	for _, tx := range transactions {
		if tx.Log == nil {
			transaction, err := s.handleValidTransaction(tx, blockHeight, blockCreatedAt)
			if err != nil {
				s.logger.Error(err)
				return err
			}
			txList = append(txList, transaction)
		} else {
			transaction, err := s.handleInvalidTransaction(tx, blockHeight, blockCreatedAt)
			if err != nil {
				s.logger.Error(err)
				return err
			}
			invalidTxList = append(invalidTxList, transaction)
		}
	}

	if len(txList) > 0 {
		s.GetSaveTxJobChannel() <- txList
		s.coinService.GetUpdateCoinsFromTxsJobChannel() <- txList
	}

	if len(invalidTxList) > 0 {
		s.GetSaveInvalidTxsJobChannel() <- invalidTxList
	}

	return nil
}

func (s *Service) SaveTransactionsWorker(jobs <-chan []*models.Transaction) {
	for transactions := range jobs {
		err := s.txRepository.SaveAll(transactions)
		if err != nil {
			s.logger.Error(err)
		}
		helpers.HandleError(err)

		links, err := s.getLinksTxValidator(transactions)
		helpers.HandleError(err)
		if len(links) > 0 {
			chunksCount := int(math.Ceil(float64(len(links)) / float64(s.env.TxChunkSize)))
			for i := 0; i < chunksCount; i++ {
				start := s.env.TxChunkSize * i
				end := start + s.env.TxChunkSize
				if end > len(links) {
					end = len(links)
				}
				s.GetSaveTxValidatorJobChannel() <- links[start:end]
			}
		}

		s.GetSaveTxsOutputJobChannel() <- transactions
	}
}
func (s *Service) SaveTransactionsOutputWorker(jobs <-chan []*models.Transaction) {
	for transactions := range jobs {
		err := s.SaveAllTxOutputs(transactions)
		if err != nil {
			s.logger.Error(err)
		}
		helpers.HandleError(err)
	}
}
func (s *Service) SaveInvalidTransactionsWorker(jobs <-chan []*models.InvalidTransaction) {
	for transactions := range jobs {
		err := s.txRepository.SaveAllInvalid(transactions)
		if err != nil {
			s.logger.Error(err)
		}
		helpers.HandleError(err)
	}
}

func (s *Service) SaveTxValidatorWorker(jobs <-chan []*models.TransactionValidator) {
	for links := range jobs {
		err := s.txRepository.LinkWithValidators(links)
		if err != nil {
			s.logger.Error(err)
		}
		helpers.HandleError(err)
	}
}

func (s *Service) UpdateTxsIndexWorker() {
	for {
		err := s.txRepository.IndexLastNTxAddress(s.env.WrkUpdateTxsIndexNumBlocks)
		if err != nil {
			s.logger.Error(err)
		}
		time.Sleep(time.Duration(s.env.WrkUpdateTxsIndexTime) * time.Second)
	}
}

func (s *Service) SaveAllTxOutputs(txList []*models.Transaction) error {
	var (
		list    []*models.TransactionOutput
		idsList []uint64
	)

	for _, tx := range txList {
		if tx.ID == 0 {
			return errors.New("no transaction id")
		}

		idsList = append(idsList, tx.ID)

		if tx.Type != node_models.TxTypeSend && tx.Type != node_models.TxTypeMultiSend && tx.Type != node_models.TxTypeRedeemCheck {
			continue
		}

		if tx.Type == node_models.TxTypeSend {
			if tx.IData.(node_models.SendTxData).To == "" {
				return errors.New("empty receiver of transaction")
			}

			toId, err := s.addressRepository.FindId(helpers.RemovePrefixFromAddress(tx.IData.(node_models.SendTxData).To))
			helpers.HandleError(err)
			coinID, err := s.coinRepository.FindIdBySymbol(tx.IData.(node_models.SendTxData).Coin)
			helpers.HandleError(err)
			list = append(list, &models.TransactionOutput{
				TransactionID: tx.ID,
				ToAddressID:   toId,
				CoinID:        coinID,
				Value:         tx.IData.(node_models.SendTxData).Value,
			})
		}
		if tx.Type == node_models.TxTypeMultiSend {
			for _, receiver := range tx.IData.(node_models.MultiSendTxData).List {
				toId, err := s.addressRepository.FindId(helpers.RemovePrefixFromAddress(receiver.To))
				helpers.HandleError(err)
				coinID, err := s.coinRepository.FindIdBySymbol(receiver.Coin)
				helpers.HandleError(err)
				list = append(list, &models.TransactionOutput{
					TransactionID: tx.ID,
					ToAddressID:   toId,
					CoinID:        coinID,
					Value:         receiver.Value,
				})
			}
		}
		if tx.Type == node_models.TxTypeRedeemCheck {
			decoded, err := base64.StdEncoding.DecodeString(tx.IData.(node_models.RedeemCheckTxData).RawCheck)
			if err != nil {
				s.logger.WithFields(logrus.Fields{
					"Tx": tx.Hash,
				}).Error(err)
				continue
			}
			data, err := check.DecodeFromBytes(decoded)
			if err != nil {
				s.logger.WithFields(logrus.Fields{
					"Tx": tx.Hash,
				}).Error(err)
				continue
			}
			sender, err := data.Sender()
			if err != nil {
				s.logger.WithFields(logrus.Fields{
					"Tx": tx.Hash,
				}).Error(err)
				continue
			}

			// We are put a creator of a check into "to" field
			// because "from" field use for a person who created a transaction
			toId, err := s.addressRepository.FindId(helpers.RemovePrefixFromAddress(sender.String()))
			helpers.HandleError(err)
			coinID, err := s.coinRepository.FindIdBySymbol(data.Coin.String())
			helpers.HandleError(err)

			list = append(list, &models.TransactionOutput{
				TransactionID: tx.ID,
				ToAddressID:   toId,
				CoinID:        coinID,
				Value:         data.Value.String(),
			})
		}
	}

	if len(list) > 0 {
		err := s.txRepository.SaveAllTxOutputs(list)
		helpers.HandleError(err)
	}
	if len(idsList) > 0 {
		err := s.txRepository.IndexTxAddress(idsList)
		helpers.HandleError(err)
	}

	return nil
}

func (s *Service) handleValidTransaction(tx responses.Transaction, blockHeight uint64, blockCreatedAt time.Time) (*models.Transaction, error) {
	fromId, err := s.addressRepository.FindId(helpers.RemovePrefixFromAddress(tx.From))
	if err != nil {
		return nil, err
	}
	nonce, err := strconv.ParseUint(tx.Nonce, 10, 64)
	if err != nil {
		return nil, err
	}
	gas, err := strconv.ParseUint(tx.Gas, 10, 64)
	if err != nil {
		return nil, err
	}
	gasCoin, err := s.coinRepository.FindIdBySymbol(tx.GasCoin)
	if err != nil {
		return nil, err
	}
	payload, err := base64.StdEncoding.DecodeString(tx.Payload)
	if err != nil {
		return nil, err
	}
	rawTxData := make([]byte, hex.DecodedLen(len(tx.RawTx)))
	rawTx, err := hex.Decode(rawTxData, []byte(tx.RawTx))
	if err != nil {
		return nil, err
	}
	transaction := &models.Transaction{
		FromAddressID: fromId,
		BlockID:       blockHeight,
		Nonce:         nonce,
		GasPrice:      uint64(tx.GasPrice),
		Gas:           gas,
		GasCoinID:     gasCoin,
		CreatedAt:     blockCreatedAt,
		Type:          tx.Type,
		Hash:          helpers.RemovePrefix(tx.Hash),
		ServiceData:   tx.ServiceData,
		Data:          tx.Data,
		IData:         tx.IData,
		Tags:          *tx.Tags,
		Payload:       payload,
		RawTx:         rawTxData[:rawTx],
	}

	return transaction, nil
}

func (s *Service) handleInvalidTransaction(tx responses.Transaction, blockHeight uint64, blockCreatedAt time.Time) (*models.InvalidTransaction, error) {
	fromId, err := s.addressRepository.FindId(helpers.RemovePrefixFromAddress(tx.From))
	if err != nil {
		return nil, err
	}
	invalidTxData, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}
	return &models.InvalidTransaction{
		FromAddressID: fromId,
		BlockID:       blockHeight,
		CreatedAt:     blockCreatedAt,
		Type:          tx.Type,
		Hash:          helpers.RemovePrefix(tx.Hash),
		TxData:        string(invalidTxData),
	}, nil
}

func (s *Service) getLinksTxValidator(transactions []*models.Transaction) ([]*models.TransactionValidator, error) {
	var links []*models.TransactionValidator

	for _, tx := range transactions {
		if tx.ID == 0 {
			return nil, errors.New("no transaction id")
		}
		var validatorPk string
		switch tx.Type {
		case node_models.TxTypeDeclareCandidacy:
			validatorPk = tx.IData.(node_models.DeclareCandidacyTxData).PubKey
		case node_models.TxTypeDelegate:
			validatorPk = tx.IData.(node_models.DelegateTxData).PubKey
		case node_models.TxTypeUnbound:
			validatorPk = tx.IData.(node_models.UnbondTxData).PubKey
		case node_models.TxTypeSetCandidateOnline:
			validatorPk = tx.IData.(node_models.SetCandidateTxData).PubKey
		case node_models.TxTypeSetCandidateOffline:
			validatorPk = tx.IData.(node_models.SetCandidateTxData).PubKey
		case node_models.TxTypeEditCandidate:
			validatorPk = tx.IData.(node_models.EditCandidateTxData).PubKey
		}

		if validatorPk != "" {
			validatorId, err := s.validatorRepository.FindIdByPkOrCreate(helpers.RemovePrefix(validatorPk))
			if err != nil {
				return nil, err
			}
			links = append(links, &models.TransactionValidator{
				TransactionID: tx.ID,
				ValidatorID:   validatorId,
			})
		}
	}

	return links, nil
}

func (s *Service) FindTransactionByHash(hash string) (*models.Transaction, error) {
	trx, err := s.txRepository.FindTransactionByHash(hash)
	if err != nil {
		return nil, err
	}
	return trx, nil
}
