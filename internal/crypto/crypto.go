package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

const prefix = "enc:v1:"

// Encryptor provides AES-256-GCM encryption at rest for journal transcripts.
type Encryptor struct {
	aead cipher.AEAD
}

func New(base64Key string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must decode to 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Encryptor{aead: aead}, nil
}

// EncryptJSON seals raw JSON and returns it wrapped as a JSON string token,
// so it can still live in a JSONB column.
func (e *Encryptor) EncryptJSON(raw []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	sealed := e.aead.Seal(nonce, nonce, raw, nil)
	return json.Marshal(prefix + base64.StdEncoding.EncodeToString(sealed))
}

// DecryptJSON reverses EncryptJSON. Non-encrypted input (legacy plaintext
// rows) is returned unchanged.
func (e *Encryptor) DecryptJSON(stored []byte) ([]byte, error) {
	var token string
	if err := json.Unmarshal(stored, &token); err != nil || !strings.HasPrefix(token, prefix) {
		return stored, nil
	}
	sealed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(token, prefix))
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}
	ns := e.aead.NonceSize()
	if len(sealed) < ns {
		return nil, fmt.Errorf("ciphertext too short")
	}
	raw, err := e.aead.Open(nil, sealed[:ns], sealed[ns:], nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt transcript: %w", err)
	}
	return raw, nil
}
