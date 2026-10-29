package port

import "github.com/alexmargomez/docforge-api/internal/domain/entity"

// DocumentSignerPort (Outbound Port - Hexagonal Architecture)
// Abstrae el cálculo de hashes SHA-256 / HMAC-SHA256 y el registro de auditoría verificable.
type DocumentSignerPort interface {
	ComputeHashes(pdfBytes []byte) (sha256Hex string, hmacHex string)
	RegisterDocument(rec entity.DocumentAuditRecord)
	VerifyHash(sha256Hex string) (entity.DocumentAuditRecord, bool)
	CountRegistered() int
}
