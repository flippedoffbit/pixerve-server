package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
)

func GenerateRNS() (string, error) {
	const rnsLength = 12
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	byteArray := make([]byte, rnsLength)
	_, err := rand.Read(byteArray)
	if err != nil {
		return "", err
	}

	var rnsBuilder strings.Builder
	for _, b := range byteArray {
		rnsBuilder.WriteByte(charset[int(b)%len(charset)])
	}

	rns := rnsBuilder.String()
	if len(rns) != rnsLength {
		return "", errors.New("failed to generate RNS of correct length")
	}

	return rns, nil
}

func GenerateRandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
