package coin

import (
	"github.com/noah-blockchain/noah-extender/internal/utils"
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
	volume := "1000495425641816924540763"
	price := "3506938666610866169"

	capitalization := utils.ConvertCapitalizationQNoahToNoah(GetCapitalization(volume, price))
	if capitalization != "3508676.093999999851159724" {
		t.Error("Capitalization must be 3508676.093999999851159724 but now ", capitalization)
	}

	volume = "1672243766121708484342"
	price = "1022577590444008568724"
	capitalization = utils.ConvertCapitalizationQNoahToNoah(GetCapitalization(volume, price))
	if capitalization != "1709999.000999999927460752" {
		t.Error("Capitalization must be 1709999.000999999927460752 but now ", capitalization)
	}

	volume = "1837730042574115525015"
	price = "209654952685560632237"
	capitalization = utils.ConvertCapitalizationQNoahToNoah(GetCapitalization(volume, price))
	if capitalization != "385289.205099999983655786" {
		t.Error("Capitalization must be 385289.205099999983655786 but now ", capitalization)
	}
}
