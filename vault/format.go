package vault

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrInvalidFormat      = errors.New("vault: invalid vault format")
	ErrUnsupportedVersion = errors.New("vault: unsupported vault version")
)

var formatMagic = [6]byte{'M', 'F', 'V', 'A', 'U', 'L'}

const (
	formatVersion   byte = 1
	kdfPBKDF2SHA256 byte = 1
	headerFixedSize      = 6 + 1 + 1 + 4 + 2 + 1
	maxSaltSize          = 64
	maxNonceSize         = 32
)

// vaultHeader is serialized before the ciphertext. The complete serialized
// header, including salt and nonce, is authenticated as GCM associated data.
type vaultHeader struct {
	version    byte
	kdf        byte
	iterations uint32
	salt       []byte
	nonce      []byte
}

func (h vaultHeader) marshal() []byte {
	buffer := make([]byte, headerFixedSize+len(h.salt)+len(h.nonce))
	copy(buffer, formatMagic[:])
	buffer[6] = h.version
	buffer[7] = h.kdf
	binary.BigEndian.PutUint32(buffer[8:12], h.iterations)
	binary.BigEndian.PutUint16(buffer[12:14], uint16(len(h.salt)))
	buffer[14] = byte(len(h.nonce))
	copy(buffer[headerFixedSize:], h.salt)
	copy(buffer[headerFixedSize+len(h.salt):], h.nonce)
	return buffer
}

func parseHeader(data []byte) (vaultHeader, int, error) {
	if len(data) < headerFixedSize {
		return vaultHeader{}, 0, ErrInvalidFormat
	}
	if string(data[:len(formatMagic)]) != string(formatMagic[:]) {
		return vaultHeader{}, 0, ErrInvalidFormat
	}
	if data[6] != formatVersion {
		return vaultHeader{}, 0, fmt.Errorf("%w: %d", ErrUnsupportedVersion, data[6])
	}
	if data[7] != kdfPBKDF2SHA256 {
		return vaultHeader{}, 0, ErrInvalidFormat
	}
	saltSize := int(binary.BigEndian.Uint16(data[12:14]))
	nonceSize := int(data[14])
	if saltSize < 16 || saltSize > maxSaltSize || nonceSize < 8 || nonceSize > maxNonceSize {
		return vaultHeader{}, 0, ErrInvalidFormat
	}
	headerSize := headerFixedSize + saltSize + nonceSize
	if headerSize > len(data) {
		return vaultHeader{}, 0, ErrInvalidFormat
	}
	header := vaultHeader{
		version:    data[6],
		kdf:        data[7],
		iterations: binary.BigEndian.Uint32(data[8:12]),
		salt:       append([]byte(nil), data[headerFixedSize:headerFixedSize+saltSize]...),
		nonce:      append([]byte(nil), data[headerFixedSize+saltSize:headerSize]...),
	}
	return header, headerSize, nil
}
