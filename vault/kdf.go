package vault

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

// deriveKey implements PBKDF2 with HMAC-SHA256 as described by RFC 8018.
// It is kept local because MayFly cannot depend on golang.org/x/crypto.
func deriveKey(password, salt []byte, iterations, length int) []byte {
	key := make([]byte, length)
	if length == 0 {
		return key
	}

	blocks := (length + sha256.Size - 1) / sha256.Size
	var blockNumber [4]byte
	var digest [sha256.Size]byte
	var previous [sha256.Size]byte
	var accumulator [sha256.Size]byte
	for block := 1; block <= blocks; block++ {
		binary.BigEndian.PutUint32(blockNumber[:], uint32(block))
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write(blockNumber[:])
		copy(digest[:], mac.Sum(nil))
		copy(previous[:], digest[:])
		copy(accumulator[:], digest[:])

		for round := 1; round < iterations; round++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(previous[:])
			copy(digest[:], mac.Sum(nil))
			for index := range digest {
				accumulator[index] ^= digest[index]
			}
			copy(previous[:], digest[:])
		}

		start := (block - 1) * sha256.Size
		copy(key[start:], accumulator[:])
	}
	return key
}
