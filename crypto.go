package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/hammertrack/tracker/errors"
	"github.com/hammertrack/tracker/utils"
)

var ErrInvalidLength = errors.New("aead: invalid length")

func encrypt(plaintext, secret []byte) ([]byte, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Wrap(err)
	}

	b := aead.Seal(nonce, nonce, plaintext, nil)
	return b64enc(b), nil
}

func decrypt(enc string, secret []byte) ([]byte, error) {
	data, err := b64dec(enc)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	l := aead.NonceSize()
	if len(data) < l {
		return nil, errors.Wrap(ErrInvalidLength)
	}

	nonce, plaintext := data[:l], data[l:]
	b, err := aead.Open(nil, nonce, plaintext, nil)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return b, nil
}

func b64enc(src []byte) []byte {
	// TODO - maybe we could minimize memory allocations reusing buffers from
	// other cursors by using a pool of buffers for b64 encoding/decoding, but
	// this buffer pool should be used at a higher scope where we control the
	// lifespan of dst so we can put it back to the pool. Which requires
	// restructuring this, encrypt/decrypt and cursor
	dst := make([]byte, base64.URLEncoding.EncodedLen(len(src)))
	base64.URLEncoding.Encode(dst, src)
	return dst
}

func b64dec(data string) ([]byte, error) {
	dst := make([]byte, len(data))
	// Saves an extra allocation in this hotpath by no using DecodeString
	n, err := base64.URLEncoding.Decode(dst, utils.StrToByte(data))
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return dst[:n], nil
}
