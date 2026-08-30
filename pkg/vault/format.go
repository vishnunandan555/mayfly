package vault

import (
	"bytes"
	"encoding/binary"
	"errors"
)

var (
	MagicBytes = []byte("MFVAUL")
	ErrCorruptHeader = errors.New("vault: corrupt or invalid file header")
)

const (
	CurrentVersion = 1
	KDFPBKDF2SHA256 = 1
	DefaultIterations = 600000
	DefaultSaltLen = 16
	DefaultNonceLen = 12
)

// Header contains the authenticated header metadata for the vault file.
type Header struct {
	Magic      [6]byte
	Version    uint8
	KDFID      uint8
	Iterations uint32
	SaltLen    uint16
	NonceLen   uint8
	Salt       []byte
	Nonce      []byte
}

func (h *Header) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(h.Magic[:])
	buf.WriteByte(h.Version)
	buf.WriteByte(h.KDFID)

	var iters [4]byte
	binary.BigEndian.PutUint32(iters[:], h.Iterations)
	buf.Write(iters[:])

	var saltLen [2]byte
	binary.BigEndian.PutUint16(saltLen[:], h.SaltLen)
	buf.Write(saltLen[:])

	buf.WriteByte(h.NonceLen)
	buf.Write(h.Salt)
	buf.Write(h.Nonce)

	return buf.Bytes(), nil
}

func UnmarshalHeader(data []byte) (*Header, int, error) {
	if len(data) < 15 {
		return nil, 0, ErrCorruptHeader
	}
	if !bytes.Equal(data[:6], MagicBytes) {
		return nil, 0, ErrCorruptHeader
	}

	h := &Header{}
	copy(h.Magic[:], data[:6])
	h.Version = data[6]
	h.KDFID = data[7]
	h.Iterations = binary.BigEndian.Uint32(data[8:12])
	h.SaltLen = binary.BigEndian.Uint16(data[12:14])
	h.NonceLen = data[14]

	offset := 15
	if len(data) < offset+int(h.SaltLen)+int(h.NonceLen) {
		return nil, 0, ErrCorruptHeader
	}

	h.Salt = make([]byte, h.SaltLen)
	copy(h.Salt, data[offset:offset+int(h.SaltLen)])
	offset += int(h.SaltLen)

	h.Nonce = make([]byte, h.NonceLen)
	copy(h.Nonce, data[offset:offset+int(h.NonceLen)])
	offset += int(h.NonceLen)

	return h, offset, nil
}
