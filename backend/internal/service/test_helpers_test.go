package service

import (
	"github.com/medflow/backend/internal/pkg/password"
)

func hashPasswordForTest(pwd string) string {
	hash, err := password.Hash(pwd)
	if err != nil {
		panic(err)
	}
	return hash
}
