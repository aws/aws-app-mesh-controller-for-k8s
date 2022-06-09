package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// RandomDNS1123Label generates a random DNS1123 compatible label with specified length
func RandomDNS1123Label(length int) string {
	seedLen := (length + 1) / 2
	seedBuf := make([]byte, seedLen)
	io.ReadFull(rand.Reader, seedBuf[:])

	labelBuf := make([]byte, seedLen*2)
	hex.Encode(labelBuf, seedBuf)
	return string(labelBuf[:length])
}

func RandomDNS1123LabelWithPrefix(prefix string) string {
	return fmt.Sprintf("vn-%s", RandomDNS1123Label(8))
}
