package utils

import (
	"math/big"
	"strings"

	"github.com/google/uuid"
)

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func base62Encode(b []byte) string {
	num := new(big.Int).SetBytes(b)
	if num.Cmp(big.NewInt(0)) == 0 {
		return string(base62Alphabet[0])
	}
	var sb strings.Builder
	for num.Cmp(big.NewInt(0)) > 0 {
		mod := new(big.Int)
		num.DivMod(num, big.NewInt(62), mod)
		sb.WriteByte(base62Alphabet[mod.Int64()])
	}
	// Reverse the string
	runes := []rune(sb.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func GenerateShortKey(length int) (string, error) {
	for {
		u := uuid.New()
		encoded := base62Encode(u[:])
		if len(encoded) >= length {
			return encoded[:length], nil
		}
		// If for some reason the encoding is too short, try again
	}
}
