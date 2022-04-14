package detect

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
)

func OpenJSONFile(fileName string) []byte {
	jsonFile, err := os.Open(fileName)

	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	return byteValue
}

func GetBalance(balance string, decimal int) float64 {

	fbalance := new(big.Float)
	fbalance.SetString(balance)
	ethValue, _ := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(decimal))).Float64()
	return ethValue
}
