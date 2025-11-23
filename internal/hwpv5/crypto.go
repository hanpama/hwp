package hwpv5

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"io"
)

// cryptoReader implements AES-128 ECB on-the-fly decryption
type cryptoReader struct {
	r     io.Reader
	block cipher.Block
	buf   []byte
	ptr   int
}

func (cr *cryptoReader) Read(p []byte) (n int, err error) {
	if cr.ptr < len(cr.buf) {
		n = copy(p, cr.buf[cr.ptr:])
		cr.ptr += n
		return n, nil
	}

	// ECB requires full 16-byte blocks
	blk := make([]byte, 16)
	readBytes, err := io.ReadFull(cr.r, blk)
	if err != nil {
		if err == io.EOF && readBytes == 0 {
			return 0, io.EOF
		}
		// Distribution documents must have block-aligned encrypted streams
		if err == io.ErrUnexpectedEOF {
			return 0, errors.New("encrypted stream not aligned to block size")
		}
		return 0, err
	}

	cr.block.Decrypt(blk, blk)
	cr.buf = blk
	cr.ptr = 0

	return cr.Read(p)
}

// deriveKey extracts the AES-128 key from the distribution header using HWP's custom algorithm:
// 1. Extract seed from first 4 bytes
// 2. Generate 256-byte random array using MSVC rand() with seed
// 3. XOR the random array with distData
// 4. Extract 16-byte key at offset (seed & 0x0F) + 4
func deriveKey(distData []byte) ([]byte, error) {
	if len(distData) != 256 {
		return nil, errors.New("invalid distribution data size")
	}

	seed := binary.LittleEndian.Uint32(distData[0:4])

	randParams := &msvcRand{state: seed}
	randomArray := make([]byte, 256)

	for i := 0; i < 256; {
		val := randParams.rand()
		cnt := randParams.rand()

		v := byte(val & 0xFF)
		c := int((cnt & 0x0F) + 1)

		for j := 0; j < c && i < 256; j++ {
			randomArray[i] = v
			i++
		}
	}

	xorData := make([]byte, 256)
	for i := 0; i < 256; i++ {
		xorData[i] = distData[i] ^ randomArray[i]
	}

	offset := int((seed & 0x0F) + 4)
	if offset+16 > 256 {
		return nil, errors.New("invalid key offset")
	}

	key := make([]byte, 16)
	copy(key, xorData[offset:offset+16])

	return key, nil
}

// msvcRand implements MS Visual C++ rand()
// Formula: next = previous * 214013 + 2531011
type msvcRand struct {
	state uint32
}

func (r *msvcRand) rand() uint32 {
	r.state = r.state*214013 + 2531011
	return (r.state >> 16) & 0x7FFF
}
