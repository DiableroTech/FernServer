package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func newTestEncryptor(t *testing.T) *Encryptor {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	enc, err := New(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	return enc
}

func TestRoundTrip(t *testing.T) {
	enc := newTestEncryptor(t)
	plain := []byte(`[{"role":"user","content":"hello fern"}]`)

	sealed, err := enc.EncryptJSON(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("hello fern")) {
		t.Fatal("ciphertext contains plaintext")
	}

	got, err := enc.DecryptJSON(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("round trip mismatch: %s", got)
	}
}

func TestLegacyPlaintextPassthrough(t *testing.T) {
	enc := newTestEncryptor(t)
	legacy := []byte(`[{"role":"user","content":"old unencrypted row"}]`)
	got, err := enc.DecryptJSON(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, legacy) {
		t.Fatal("legacy plaintext should pass through unchanged")
	}
}

func TestWrongKeyFails(t *testing.T) {
	sealed, err := newTestEncryptor(t).EncryptJSON([]byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTestEncryptor(t).DecryptJSON(sealed); err == nil {
		t.Fatal("decrypting with a different key should fail")
	}
}

func TestBadKeyRejected(t *testing.T) {
	if _, err := New("dG9vc2hvcnQ="); err == nil {
		t.Fatal("short key should be rejected")
	}
	if _, err := New("not base64!!!"); err == nil {
		t.Fatal("invalid base64 should be rejected")
	}
}
