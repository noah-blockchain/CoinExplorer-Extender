package coin

import (
	"math/big"

	"github.com/noah-blockchain/noah-go-node/core/types"
	"github.com/noah-blockchain/noah-go-node/math"
	"github.com/pkg/errors"
)

const (
	precision = 100
)

func newFloat(x float64) *big.Float {
	return big.NewFloat(x).SetPrec(precision)
}

func convertStringToBigInt(value string) (*big.Int, error) {
	newValue := new(big.Int)
	newValue, ok := newValue.SetString(value, 10)
	if !ok {
		return nil, errors.New("Can't convert string to big.Int (" + value + ")")
	}
	return newValue, nil
}

func NoahToQNoah(noah *big.Float) *big.Int {
	p := big.NewInt(10)
	p.Exp(p, big.NewInt(18), nil)
	noah.Mul(noah, new(big.Float).SetInt(p))
	res, _ := noah.Int(nil)
	return res
}

//reserve * (math.pow(1 + 1 / volume, 100 / crr) - 1)
func calculatePurchaseAmount(supply *big.Int, reserve *big.Int, crr uint, wantReceive *big.Int) *big.Int {
	if wantReceive.Cmp(types.Big0) == 0 || supply == nil || reserve == nil || supply.Cmp(types.Big0) == 0 {
		return big.NewInt(0)
	}

	tSupply := newFloat(0).SetInt(supply)
	tReserve := newFloat(0).SetInt(reserve)
	tWantReceive := newFloat(0).SetInt(wantReceive)

	if crr == 100 {
		result := newFloat(float64(wantReceive.Uint64()))
		result.Mul(result, tReserve)
		result.Quo(result, tSupply)
		return NoahToQNoah(result)
	}

	res := newFloat(0).Add(tWantReceive, tSupply)   // reserve + supply
	res.Quo(res, tSupply)                           // (reserve + supply) / supply
	res = math.Pow(res, newFloat(100/float64(crr))) // ((reserve + supply) / supply)^(100/c)
	res.Sub(res, newFloat(1))                       // (((reserve + supply) / supply)^(100/c) - 1)
	res.Mul(res, tReserve)                          // reserve * (((reserve + supply) / supply)^(100/c) - 1)

	return NoahToQNoah(res)
}

func GetTokenPrice(volumeStr string, reserveStr string, crr uint64) string {
	volume, _ := convertStringToBigInt(volumeStr)
	reserve, _ := convertStringToBigInt(reserveStr)

	return calculatePurchaseAmount(volume, reserve, uint(crr), big.NewInt(1)).String()
}

func GetCapitalization(volumeStr string, priceStr string) string {
	tVolume, _ := newFloat(0).SetString(volumeStr)
	tPrice, _ := newFloat(0).SetString(priceStr)
	tVolume.Mul(tVolume, tPrice)

	return tVolume.String()
}
