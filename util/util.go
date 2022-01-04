package util

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func RoundFloatWithPrecision(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(math.Floor(num*output)) / output
}

func ChunkSlice(slice []int64, chunkSize int) [][]int64 {
	var chunks [][]int64
	for {
		if len(slice) == 0 {
			break
		}

		// necessary check to avoid slicing beyond
		// slice capacity
		if len(slice) < chunkSize {
			chunkSize = len(slice)
		}

		chunks = append(chunks, slice[0:chunkSize])
		slice = slice[chunkSize:]
	}

	return chunks
}

func GetCurrencyFromSymbol(symbol string) (string, error) {

	if symbol[len(symbol)-4:] == "USDT" {
		return "USDT", nil

	} else if symbol[len(symbol)-4:] == "BUSD" {
		return "BUSD", nil

	} else if symbol[len(symbol)-3:] == "BNB" {
		return "BNB", nil

	} else if symbol[len(symbol)-3:] == "BTC" {
		return "BTC", nil

	} else {
		return "", fmt.Errorf("asset not recognized in symbol %s", symbol)
	}
}

func ComparePositionSides(positionSideBinance string, positionSide string) bool {

	s1 := strings.ToUpper(positionSideBinance)
	s2 := strings.ToUpper(positionSide)
	return s1 == s2

}

func Float64ToString(x float64) string {
	return strconv.FormatFloat(x, 'f', -1, 64)
}
