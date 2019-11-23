package entropy

import (
	"math"
	"strings"
)

func GetWordShannonEntropy(data string) (entropy float64) {
	words := strings.Fields(data)
	var maxEntropy float64
	maxEntropy = 0
	for _, word := range words {
		_entropy := GetShannonEntropy(word)
		if _entropy > maxEntropy {
			maxEntropy = _entropy
		}
	}
	return maxEntropy
}

func GetShannonEntropy(data string) (entropy float64) {
	if data == "" {
		return 0
	}

	charCounts := make(map[rune]int)
	for _, char := range data {
		charCounts[char]++
	}

	invLength := 1.0 / float64(len(data))
	for _, count := range charCounts {
		freq := float64(count) * invLength
		entropy -= freq * math.Log2(freq)
	}

	return entropy
}
