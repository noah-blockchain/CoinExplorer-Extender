package coin

import (
	"math/big"

	"github.com/noah-blockchain/CoinExplorer-Extender/internal/utils"
	"github.com/noah-blockchain/noah-go-node/core/types"
	"github.com/noah-blockchain/noah-go-node/math"
)

const (
	precision = 100
)

//reserve * (math.pow(1 + 1 / volume, 100 / crr) - 1)
func calculatePurchaseAmount(supply *big.Int, reserve *big.Int, crr uint, wantReceive *big.Int) *big.Int {
	if wantReceive.Cmp(types.Big0) == 0 || supply.Cmp(types.Big0) == 0 {
		return big.NewInt(0)
	}

	tSupply := utils.NewFloat(0, precision).SetInt(supply)
	tReserve := utils.NewFloat(0, precision).SetInt(reserve)
	tWantReceive := utils.NewFloat(0, precision).SetInt(wantReceive)

	if crr == 100 {
		result := utils.NewFloat(float64(wantReceive.Uint64()), precision)
		result.Mul(result, tReserve)
		result.Quo(result, tSupply)
		return utils.NoahToQNoah(result)
	}

	res := utils.NewFloat(0, precision).Add(tWantReceive, tSupply)   // reserve + supply
	res.Quo(res, tSupply)                                            // (reserve + supply) / supply
	res = math.Pow(res, utils.NewFloat(100/float64(crr), precision)) // ((reserve + supply) / supply)^(100/c)
	res.Sub(res, utils.NewFloat(1, precision))                       // (((reserve + supply) / supply)^(100/c) - 1)
	res.Mul(res, tReserve)                                           // reserve * (((reserve + supply) / supply)^(100/c) - 1)

	return utils.NoahToQNoah(res)
}

func GetTokenPrice(volumeStr string, reserveStr string, crr uint64) string {
	volume := utils.ConvertStringToBigInt(volumeStr)
	reserve := utils.ConvertStringToBigInt(reserveStr)

	return calculatePurchaseAmount(volume, reserve, uint(crr), big.NewInt(1)).String()
}

func GetCapitalization(volumeStr string, priceStr string) string {
	tVolume, _ := utils.NewFloat(0, precision).SetString(volumeStr)
	tPrice, _ := utils.NewFloat(0, precision).SetString(priceStr)
	tVolume.Mul(tVolume, tPrice)

	return tVolume.String()
}
