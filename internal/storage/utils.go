package storage

import (
	"crypto/rand"
	"math/big"
)

func generateInviteCode() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // без похожих I, 1, O, 0
	code := make([]byte, 6)
	for i := range code {
		// Выбираем случайный индекс из charset
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[num.Int64()]
	}
	return string(code), nil
}
