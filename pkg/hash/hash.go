package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

func New(text, salt string) string {
	hasher := sha256.New()
	hasher.Write([]byte(text + salt))
	return hex.EncodeToString(hasher.Sum(nil))
}
