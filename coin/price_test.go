package coin

import (
	"testing"
)

func TestCalculateTokenPrice(t *testing.T) {
	volume := "600"
	reserve := "10000"
	var crr uint64 = 100

	price := GetTokenPrice(volume, reserve, crr)
	if price != "16666666666666666666" {
		t.Error("Price must be 16666666666666666666 but now ", price)
	}

	volume = "600"
	reserve = "10000"
	crr = 10

	price = GetTokenPrice(volume, reserve, crr)
	if price != "167922238458378386515" {
		t.Error("Price must be 167922238458378386515 but now ", price)
	}

	volume = "1799000000000000000000"
	reserve = "29983333333333333333333"
	crr = 100

	price = GetTokenPrice(volume, reserve, crr)
	if price != "16666666666666666666" {
		t.Error("Price must be 16666666666666666666 but now ", price)
	}
}

func TestCalculateTokenCapitalization(t *testing.T) {
	volume := "600"
	reserve := "10000"
	var crr uint64 = 100

	price := GetTokenPrice(volume, reserve, crr)

	capitalization := GetCapitalization(volume, price)
	if capitalization != "10000000000000000000000" {
		t.Error("Capitalization must be 10000000000000000000000 but now ", capitalization)
	}

	volume = "600"
	reserve = "10000"
	crr = 10

	price = GetTokenPrice(volume, reserve, crr)

	capitalization = GetCapitalization(volume, price)
	if capitalization != "100753343075027028803584" {
		t.Error("Capitalization must be 100753343075027028803584 but now ", capitalization)
	}

	volume = "666000000000000000000"
	capitalization = GetCapitalization(volume, "1")
	if capitalization != "666000000000000000000" {
		t.Error("Capitalization must be 666000000000000000000 but now ", capitalization)
	}
}
