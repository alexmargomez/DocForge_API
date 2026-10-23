package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/alexmargomez/docforge-api/internal/domain/entity"
)

type HashSignerService struct {
	secretKey []byte
	mu        sync.RWMutex
	registry  map[string]entity.DocumentAuditRecord
}

func NewHashSignerService(secret string) *HashSignerService {
	if secret == "" {
		secret = "docforge_default_hmac_secret_2026"
	}
	return &HashSignerService{
		secretKey: []byte(secret),
		registry:  make(map[string]entity.DocumentAuditRecord),
	}
}

func (s *HashSignerService) ComputeHashes(data []byte) (sha256Hex string, hmacHex string) {
	sum := sha256.Sum256(data)
	sha256Hex = hex.EncodeToString(sum[:])

	mac := hmac.New(sha256.New, s.secretKey)
	mac.Write(data)
	hmacHex = hex.EncodeToString(mac.Sum(nil))
	return sha256Hex, hmacHex
}

func (s *HashSignerService) RegisterDocument(rec entity.DocumentAuditRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.registry) > 5000 {
		s.registry = make(map[string]entity.DocumentAuditRecord)
	}
	s.registry[rec.DocumentHashSHA256] = rec
}

func (s *HashSignerService) VerifyHash(hash string) (entity.DocumentAuditRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.registry[hash]
	return rec, ok
}

func (s *HashSignerService) CountRegistered() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.registry)
}
