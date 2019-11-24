package utils

import (
	"math/big"
)

func NewFloat(x float64, precision uint) *big.Float {
	return big.NewFloat(x).SetPrec(precision)
}

func ConvertStringToBigInt(value string) *big.Int {
	newValue := new(big.Int)
	newValue, ok := newValue.SetString(value, 10)
	if !ok {
		return big.NewInt(0)
	}
	return newValue
}

func NoahToQNoah(noah *big.Float) *big.Int {
	p := big.NewInt(10)
	p.Exp(p, big.NewInt(18), nil)
	noah.Mul(noah, new(big.Float).SetInt(p))
	res, _ := noah.Int(nil)
	return res
}

func ConvertCapitalizationQNoahToNoah(value string) string {
	if value == "" {
		return "0"
	}
	var twiceQNoahToNoah = big.NewFloat(1000000000000000000000000000000000000)
	floatValue, _ := new(big.Float).SetPrec(500).SetString(value)
	return new(big.Float).SetPrec(500).Quo(floatValue, twiceQNoahToNoah).Text('f', 18)
}

func Min(x, y uint64) uint64 {
	if x > y {
		return y
	}
	return x
}
