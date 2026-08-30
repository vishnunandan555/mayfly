package vault

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

var ErrInvalidKDFParams = errors.New("vault: invalid KDF parameters")

// DeriveKey derives a key of keyLen bytes from password and salt using
// PBKDF2-HMAC-SHA256 (RFC 8018) with standard Go crypto primitives.
func DeriveKey(password, salt []byte, iterations int, keyLen int) ([]byte, error) {
	if iterations <= 0 || keyLen <= 0 || len(password) == 0 || len(salt) == 0 {
		return nil, ErrInvalidKDFParams
	}

	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var (
		derived = make([]byte, 0, numBlocks*hashLen)
		u       = make([]byte, hashLen)
		t       = make([]byte, hashLen)
		blockIdx [4]byte
	)

	for block := 1; block <= numBlocks; block++ {
		binary.BigEndian.PutUint32(blockIdx[:], uint32(block))

		// U_1 = PRF(password, salt || INT_32_BE(block))
		prf.Reset()
		prf.Write(salt)
		prf.Write(blockIdx[:])
		u = prf.Sum(u[:0])
		copy(t, u)

		// U_2 ... U_iterations
		for iter := 1; iter < iterations; iter++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for k := 0; k < hashLen; k++ {
				t[k] ^= u[k]
			}
		}

		derived = append(derived, t...)
	}

	return derived[:keyLen], nil
}
