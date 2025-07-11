package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

var encryptionKey []byte

func init() {
	hexKey := os.Getenv("ENCRYPTION_KEY_HEX")
	if hexKey == "" {
		// Em um ambiente de produção real, isso deveria causar um erro fatal.
		// Para desenvolvimento, podemos usar uma chave padrão, mas isso é INSEGURO.
		// log.Fatalf("ENCRYPTION_KEY_HEX environment variable not set.")
		// Usando uma chave padrão APENAS para fins de desenvolvimento local se não definida.
		// ESTA CHAVE NÃO DEVE SER USADA EM PRODUÇÃO. GERE UMA NOVA.
		fmt.Println("AVISO DE SEGURANÇA: ENCRYPTION_KEY_HEX não definida, usando chave de desenvolvimento padrão. NÃO USE EM PRODUÇÃO.")
		hexKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f" // Chave de 32 bytes (AES-256)
	}
	var err error
	encryptionKey, err = hex.DecodeString(hexKey)
	if err != nil {
		panic(fmt.Sprintf("Erro ao decodificar ENCRYPTION_KEY_HEX: %v. A chave deve ser uma string hexadecimal de 32 bytes (64 caracteres).", err))
	}
	if len(encryptionKey) != 32 { // AES-256 requer chave de 32 bytes
		panic("ENCRYPTION_KEY_HEX deve ter 32 bytes (64 caracteres hexadecimais) para AES-256.")
	}
}

// Encrypt encrypts data using AES-GCM.
func Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts data using AES-GCM.
func Decrypt(ciphertextHex string) (string, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext muito curto")
	}

	nonce, encryptedMessage := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encryptedMessage, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
