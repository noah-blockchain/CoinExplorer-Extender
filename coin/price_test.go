package coin

import (
	"testing"
)

func TestCalculateTokenPrice(t *testing.T) {
	volume := "600"
	reserve := "10000"
	var crr uint64 = 100

	price := getTokenPrice(volume, reserve, crr)
	if price != "16666666666666666666" {
		t.Error("Price must be 16666666666666666666 but now ", price)
	}

	volume = "600"
	reserve = "10000"
	crr = 10

	price = getTokenPrice(volume, reserve, crr)
	if price != "167922238458378386515" {
		t.Error("Price must be 167922238458378386515 but now ", price)
	}

	volume = "1799000000000000000000"
	reserve = "29983333333333333333333"
	crr = 100

	price = getTokenPrice(volume, reserve, crr)
	if price != "16666666666666666666" {
		t.Error("Price must be 16666666666666666666 but now ", price)
	}
}
