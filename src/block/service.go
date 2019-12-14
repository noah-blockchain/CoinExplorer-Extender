package block

import (
	"github.com/noah-blockchain/CoinExplorer-Extender/validator"
	"github.com/noah-blockchain/coinExplorer-tools/helpers"
	"github.com/noah-blockchain/coinExplorer-tools/models"
	"github.com/noah-blockchain/noah-node-go-api/responses"
	"math"
	"strconv"
	"time"
)

type Service struct {
	blockRepository     *Repository
	validatorRepository *validator.Repository
	blockCache          *models.Block //Contain previous block model
}

func NewBlockService(blockRepository *Repository, validatorRepository *validator.Repository) *Service {
	return &Service{
		blockRepository:     blockRepository,
		validatorRepository: validatorRepository,
	}
}

func (s *Service) SetBlockCache(b *models.Block) {
	s.blockCache = b
}

func (s *Service) GetBlockCache() (b *models.Block) {
	return s.blockCache
}

//Handle response and save block to DB
func (s *Service) HandleBlockResponse(response *responses.BlockResponse) error {
	height, err := strconv.ParseUint(response.Result.Height, 10, 64)
	helpers.HandleError(err)
	totalTx, err := strconv.ParseUint(response.Result.TotalTx, 10, 64)
	helpers.HandleError(err)
	numTx, err := strconv.ParseUint(response.Result.TxCount, 10, 32)
	helpers.HandleError(err)
	size, err := strconv.ParseUint(response.Result.Size, 10, 64)
	helpers.HandleError(err)

	var proposerId uint64
	if response.Result.Proposer != "" {
		proposerId, err = s.validatorRepository.FindIdByPk(helpers.RemovePrefix(response.Result.Proposer))
		helpers.HandleError(err)
	} else {
		proposerId = 1
	}

	blockTime := s.getBlockTime(response.Result.Time)
	if blockTime >= math.MaxInt64 {
		blockTime = math.MaxInt64 - 1
	}

	block := &models.Block{
		ID:                  height,
		TotalTxs:            totalTx,
		NumTxs:              uint32(numTx),
		Size:                size,
		BlockTime:           blockTime,
		CreatedAt:           response.Result.Time,
		BlockReward:         response.Result.BlockReward,
		ProposerValidatorID: proposerId,
		Hash:                response.Result.Hash,
	}
	s.SetBlockCache(block)

	return s.blockRepository.Save(block)
}

func (s *Service) getBlockTime(blockTime time.Time) uint64 {
	if s.blockCache == nil {
		return uint64(1 * time.Second) //ns, 1 second for the first block
	}
	result := blockTime.Sub(s.blockCache.CreatedAt)
	return uint64(result.Nanoseconds())
}
