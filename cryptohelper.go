package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/charmbracelet/log"
)

type CryptoHelperError struct{}

func (m *CryptoHelperError) Error() string {
	return "ciphertext too short"
}

func Encrypt(key []byte, message string) (string, error) {
	plain := []byte(message)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error("could not create AES cipher", "cryptoErr", err)
		return "", err
	}

	CipherText := make([]byte, aes.BlockSize+len(plain))
	iv := CipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		log.Error("error reading random bytes into iv", "io", err)
		return "", err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(CipherText[aes.BlockSize:], plain)

	return base64.RawStdEncoding.EncodeToString(CipherText), nil
}

func Decrypt(key []byte, message string) (string, error) {
	encoded, err := base64.RawStdEncoding.DecodeString(message)
	if err != nil {
		log.Error("decoding base64 failed", "error", err)
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Error("could not create AES cipher", "cryptoErr", err)
		return "", err
	}

	if len(encoded) < aes.BlockSize {
		log.Error("ciphertext is too short")
		return "", &CryptoHelperError{}
	}

	iv := encoded[:aes.BlockSize]
	encoded = encoded[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	stream.XORKeyStream(encoded, encoded)
	return string(encoded), nil
}
