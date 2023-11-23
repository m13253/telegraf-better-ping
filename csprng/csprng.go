package csprng

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/chacha20poly1305"
)

type CSPRNG struct {
	mtx       sync.Mutex
	cipher    *chacha20.Cipher
	remaining int64
}

const (
	chacha20BlockSize  int64 = 64
	chacha20RekeyCount int64 = 1 << 32
)

func (rng *CSPRNG) Read(buf []byte) (n int, err error) {
	for i := range buf {
		buf[i] = 0
	}
	rng.mtx.Lock()
	for len(buf) != 0 {
		if rng.remaining == 0 {
			var init [chacha20.KeySize + chacha20.NonceSize]byte
			_, err = rand.Read(init[:])
			if err != nil {
				break
			}
			rng.cipher, err = chacha20.NewUnauthenticatedCipher(init[:chacha20.KeySize], init[chacha20.KeySize:])
			if err != nil {
				break
			}
			rng.remaining = chacha20BlockSize * chacha20RekeyCount
		}
		step := min(rng.remaining, int64(len(buf)))
		rng.remaining -= step
		rng.cipher.XORKeyStream(buf[:step], buf[:step])
		buf = buf[step:]
		n += int(step)
	}
	rng.mtx.Unlock()
	if err != nil {
		err = fmt.Errorf("failed to generate random numbers: %w", err)
	}
	return
}

func (rng *CSPRNG) DeriveCipher() (c cipher.AEAD, err error) {
	var key [chacha20poly1305.KeySize]byte
	_, err = rng.Read(key[:])
	if err == nil {
		c, err = chacha20poly1305.New(key[:])
	}
	if err != nil {
		err = fmt.Errorf("failed to initialize encryption engine: %w", err)
	}
	return
}

func (rng *CSPRNG) UInt16() (val uint16, err error) {
	var buf [2]byte
	_, err = rng.Read(buf[:])
	val = binary.NativeEndian.Uint16(buf[:])
	return
}
