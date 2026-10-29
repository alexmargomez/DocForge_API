package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/alexmargomez/docforge-api/internal/domain/entity"
)

const DefaultMaxAuditCapacity = 5000

// HashSignerService implementa firma HMAC-SHA256 y un registro de auditoría en memoria
// con Ring Buffer FIFO O(1) que desaloja únicamente el documento más antiguo al alcanzar
// la capacidad máxima, sin purgar jamás la totalidad del historial.
type HashSignerService struct {
	secretKey   []byte
	mu          sync.RWMutex
	maxCapacity int
	registry    map[string]entity.DocumentAuditRecord
	ring        []string
	head        int
}

func NewHashSignerService(secret string) *HashSignerService {
	return NewHashSignerServiceWithCapacity(secret, DefaultMaxAuditCapacity)
}

func NewHashSignerServiceWithCapacity(secret string, maxCapacity int) *HashSignerService {
	if secret == "" {
		secret = "docforge_default_hmac_secret_2026"
	}
	if maxCapacity < 1 {
		maxCapacity = DefaultMaxAuditCapacity
	}
	return &HashSignerService{
		secretKey:   []byte(secret),
		maxCapacity: maxCapacity,
		registry:    make(map[string]entity.DocumentAuditRecord, maxCapacity),
		ring:        make([]string, 0, maxCapacity),
		head:        0,
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

	hash := rec.DocumentHashSHA256
	if _, exists := s.registry[hash]; exists {
		s.registry[hash] = rec
		return
	}

	if len(s.ring) < s.maxCapacity {
		s.ring = append(s.ring, hash)
	} else {
		oldestHash := s.ring[s.head]
		delete(s.registry, oldestHash)
		s.ring[s.head] = hash
		s.head = (s.head + 1) % s.maxCapacity
	}
	s.registry[hash] = rec
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
