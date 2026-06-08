package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type SecretStore struct {
	path    string
	secrets map[string]string
	key     []byte
}

func NewSecretStore(path string) (*SecretStore, error) {
	keyPath := filepath.Join(filepath.Dir(path), ".secret_key")
	key, err := loadOrCreateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("secret key: %w", err)
	}

	s := &SecretStore{
		path:    path,
		secrets: make(map[string]string),
		key:     key,
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := s.decrypt(data); err != nil {
			return s, nil
		}
	}

	return s, nil
}

func (s *SecretStore) Get(key string) (string, bool) {
	v, ok := s.secrets[key]
	return v, ok
}

func (s *SecretStore) Set(key, value string) {
	s.secrets[key] = value
}

func (s *SecretStore) Delete(key string) {
	delete(s.secrets, key)
}

func (s *SecretStore) Save() error {
	data, err := s.encrypt()
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *SecretStore) encrypt() ([]byte, error) {
	plain, err := json.Marshal(s.secrets)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aesGCM.Seal(nonce, nonce, plain, nil), nil
}

func (s *SecretStore) decrypt(data []byte) error {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plain, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, &s.secrets)
}

func loadOrCreateKey(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil {
		return hex.DecodeString(string(data))
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	encoded := hex.EncodeToString(key)
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return nil, err
	}
	return key, nil
}
