package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	signaturePrefix = "sha256="
	signatureLength = 64
)

func ValidateSignature(payload []byte, signature, secret string) bool {
	if secret == "" {
		return false
	}

	if signature == "" {
		return false
	}

	if !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}

	receivedHash := strings.TrimPrefix(signature, signaturePrefix)
	if len(receivedHash) != signatureLength {
		return false
	}

	expectedHash := computeHMAC(payload, secret)

	return hmac.Equal([]byte(receivedHash), []byte(expectedHash))
}

func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func GenerateSignature(payload []byte, secret string) string {
	return signaturePrefix + computeHMAC(payload, secret)
}
