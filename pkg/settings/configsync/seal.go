package configsync

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"

	"golang.org/x/crypto/argon2"
)

const (
	codeDigits = 6
	keyTime    = 2
	keyMemory  = 32 * 1024
	keyThreads = 2
	keyBytes   = 32
)

var errOpen = errors.New("the settings could not be opened")

func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", codeDigits, n.Int64()), nil
}

type key []byte

func derive(code, id string) key {
	return argon2.IDKey(
		[]byte(code),
		[]byte("countertop sync "+id),
		keyTime,
		keyMemory,
		keyThreads,
		keyBytes,
	)
}

func (k key) proof(id string) []byte {
	mac := hmac.New(sha256.New, k)
	mac.Write([]byte("proof " + id))
	return mac.Sum(nil)
}

func (k key) proves(id string, proof []byte) bool { return hmac.Equal(k.proof(id), proof) }

func (k key) seal(id string, plain []byte) (nonce, sealed []byte, err error) {
	aead, err := k.aead()
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return nonce, aead.Seal(nil, nonce, plain, []byte(id)), nil
}

func (k key) open(id string, nonce, sealed []byte) ([]byte, error) {
	aead, err := k.aead()
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, errOpen
	}
	plain, err := aead.Open(nil, nonce, sealed, []byte(id))
	if err != nil {
		return nil, errOpen
	}
	return plain, nil
}

func (k key) aead() (cipher.AEAD, error) {
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
