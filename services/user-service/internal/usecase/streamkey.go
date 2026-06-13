package usecase

import (
	"fmt"

	"github.com/sahiy/sahiy-stream/pkg/crypto"
)

func generateStreamKey() (plain, prefix, lookup string, err error) {
	token, err := crypto.GenerateToken(24)
	if err != nil {
		return "", "", "", err
	}
	plain = fmt.Sprintf("sk_live_%s", token)
	if len(plain) > 16 {
		prefix = plain[:16]
	} else {
		prefix = plain
	}
	lookup = crypto.SHA256Hex(plain)
	return plain, prefix, lookup, nil
}
