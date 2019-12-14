package env

import (
	"flag"
	"os"

	"github.com/noah-blockchain/coinExplorer-tools/models"
)

func New() *models.ExtenderEnvironment {
	appName := flag.String("app_name", "Coin Extender", "App name")
	baseCoin := flag.String("base_coin", "NOAH", "Base coin symbol")
	coinsUpdateTime := flag.Int("coins_upd_time", 3600, "Coins update time in minutes")
	txChunkSize := flag.Int("tx_chunk_size", 100, "Transactions chunk size")
	eventsChunkSize := flag.Int("event_chunk_size", 100, "Events chunk size")
	stakeChunkSize := flag.Int("stake_chunk_size", 100, "Stake chunk size")
	wrkSaveTxsCount := flag.Int("wrk_save_txs_count", 3, "Count of workers that save transactions")
	wrkSaveTxsOutputCount := flag.Int("wrk_save_txs_output_count", 3, "Count of workers that save transactions output")
	wrkSaveInvalidTxsCount := flag.Int("wrk_save_invtxs_count", 3, "Count of workers that save invalid transactions")
	wrkSaveRewardsCount := flag.Int("wrk_save_rewards_count", 3, "Count of workers that save rewards")
	wrkSaveSlashesCount := flag.Int("wrk_save_slashes_count", 3, "Count of workers that save slashes")
	wrkSaveAddressesCount := flag.Int("wrk_save_addresses_count", 3, "Count of workers that save addresses")
	wrkSaveValidatorTxsCount := flag.Int("wrk_save_val_tx_count", 3, "Count of workers that save transaction-validator link")
	addrChunkSize := flag.Int("addr_chunk_size", 10, "Count of workers that save transaction-validator link")
	wrkUpdateBalanceCount := flag.Int("wrk_upd_balances_count", 1, "Count of workers that update balance")
	wrkGetBalancesFromNodeCount := flag.Int("wrk_node_balance_count", 1, "Count of workers that get balance from node")
	wrkUpdateTxsIndexNumBlocks := flag.Int("wrk_update_txs_index_num_blocks", 120, "Count of blocks that should be reindex")
	wrkUpdateTxsIndexTime := flag.Int("wrk_update_txs_index_time", 60, "Time in seconds which worker sleep before the next iteration")
	rewardAggregateEveryBlocksCount := flag.Int("reward_aggregate_every_blocks_count", 60, "Every X block will be launched reward aggregation")
	rewardAggregateTimeInterval := flag.String("reward_aggregate_time_interval", "hour", "Rewards aggregation time interval('hour' or 'day')")
	flag.Parse()

	envData := new(models.ExtenderEnvironment)

	envData.DbUser = os.Getenv("DB_USER")
	envData.DbName = os.Getenv("DB_NAME")
	envData.DbPassword = os.Getenv("DB_PASSWORD")
	envData.DbHost = os.Getenv("DB_HOST")
	envData.DbPort = getEnvAsInt("DB_PORT", 5432)
	envData.NodeApi = os.Getenv("NOAH_API_NODE")
	envData.ApiHost = os.Getenv("COIN_EXTENDER_API_HOST")
	envData.ApiPort = getEnvAsInt("COIN_EXTENDER_API_PORT", 10000)
	envData.Debug = getEnvAsBool("DEBUG", true)
	envData.WsHost = os.Getenv("WS_HOST")
	envData.WsPort = getEnvAsInt("WS_PORT", 8000)
	envData.WsKey = os.Getenv("WS_API_KEY")
	envData.NatsAddr = os.Getenv("NATS_ADDR")
	envData.NatsClusterID = os.Getenv("NATS_CLUSTER_ID")

	envData.AppName = *appName
	envData.DbMinIdleConns = 10
	envData.DbPoolSize = 20
	envData.TxChunkSize = *txChunkSize
	envData.EventsChunkSize = *eventsChunkSize
	envData.StakeChunkSize = *stakeChunkSize
	envData.WrkSaveTxsCount = *wrkSaveTxsCount
	envData.WrkSaveTxsOutputCount = *wrkSaveTxsOutputCount
	envData.WrkSaveInvTxsCount = *wrkSaveInvalidTxsCount
	envData.WrkSaveRewardsCount = *wrkSaveRewardsCount
	envData.WrkSaveSlashesCount = *wrkSaveSlashesCount
	envData.WrkSaveAddressesCount = *wrkSaveAddressesCount
	envData.WrkSaveValidatorTxsCount = *wrkSaveValidatorTxsCount
	envData.AddrChunkSize = *addrChunkSize
	envData.WrkUpdateBalanceCount = *wrkUpdateBalanceCount
	envData.WrkGetBalancesFromNodeCount = *wrkGetBalancesFromNodeCount
	envData.BaseCoin = *baseCoin
	envData.CoinsUpdateTime = *coinsUpdateTime
	envData.WrkUpdateTxsIndexNumBlocks = *wrkUpdateTxsIndexNumBlocks
	envData.WrkUpdateTxsIndexTime = *wrkUpdateTxsIndexTime
	envData.RewardAggregateEveryBlocksCount = *rewardAggregateEveryBlocksCount
	envData.RewardAggregateTimeInterval = *rewardAggregateTimeInterval

	return envData
}
