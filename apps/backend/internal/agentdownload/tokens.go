package agentdownload

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const tokenValidDuration = 5 * time.Minute

type TokenStore struct {
	mu     sync.Mutex
	tokens map[string]time.Time
}

func NewTokenStore() *TokenStore {
	return &TokenStore{tokens: make(map[string]time.Time)}
}

func (s *TokenStore) Create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	expiry := time.Now().Add(tokenValidDuration)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = expiry
	return token, nil
}

func (s *TokenStore) Consume(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.tokens[token]
	if !ok || time.Now().After(expiry) {
		return false
	}
	delete(s.tokens, token)
	return true
}
