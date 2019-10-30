package coin

import (
	"log"
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

//reserve * (math.pow(1 + 1 / volume, 100 / crr) - 1)
func calculatePurchaseAmount(supply *big.Int, reserve *big.Int, crr uint, wantReceive *big.Int) *big.Int {
	if wantReceive.Cmp(types.Big0) == 0 {
		return big.NewInt(0)
	}

	if crr == 100 {
		result := big.NewInt(0).Mul(wantReceive, reserve)
		return result.Div(result, supply)
	}

	tSupply := newFloat(0).SetInt(supply)
	tReserve := newFloat(0).SetInt(reserve)
	tWantReceive := newFloat(0).SetInt(wantReceive)

	res := newFloat(0).Add(tWantReceive, tSupply)   // reserve + supply
	res.Quo(res, tSupply)                           // (reserve + supply) / supply
	res = math.Pow(res, newFloat(100/float64(crr))) // ((reserve + supply) / supply)^(100/c)
	res.Sub(res, newFloat(1))                       // (((reserve + supply) / supply)^(100/c) - 1)
	res.Mul(res, tReserve)                          // reserve * (((reserve + supply) / supply)^(100/c) - 1)

	result, _ := res.Int(nil)
	return result
}

func getTokenPrice(volumeStr string, reserveStr string, crr uint64) string {
	volume, err := convertStringToBigInt(volumeStr)
	if err != nil {
		log.Println(err)
		return "0"
	}

	reserve, err := convertStringToBigInt(reserveStr)
	if err != nil {
		log.Println(err)
		return "0"
	}

	return calculatePurchaseAmount(volume, reserve, uint(crr), big.NewInt(1)).String()
}
