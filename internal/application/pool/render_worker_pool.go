package pool

import (
	"bytes"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexmargomez/docforge-api/internal/application/template"
	"github.com/alexmargomez/docforge-api/internal/domain/entity"
	"github.com/alexmargomez/docforge-api/internal/domain/port"
)

var ErrPoolBackpressure = errors.New("cola de renderizado saturada o timeout excedido (backpressure activo): reintente en unos milisegundos")

type RenderResult struct {
	PdfBytes    []byte
	AuditRecord entity.DocumentAuditRecord
	Err         error
}

type renderJob struct {
	tmpl     template.DocumentTemplate
	respCh   chan RenderResult
	canceled *atomic.Bool
}

// Worker Pool + Object Pool Pattern (sync.Pool) + Backpressure —
// Acota el paralelismo máximo de renderizado, recicla buffers (*bytes.Buffer) y protege
// contra fugas de goroutines, timers y registros fantasma cancelando jobs expirados con atomic.Bool.
type RenderWorkerPool struct {
	workersCount  int
	jobsCh        chan renderJob
	bufferPool    sync.Pool
	signer        port.DocumentSignerPort
	totalRendered uint64
	totalBytes    uint64
}

func NewRenderWorkerPool(workers int, signer port.DocumentSignerPort) *RenderWorkerPool {
	if workers < 1 || workers > 64 {
		workers = 8
	}
	p := &RenderWorkerPool{
		workersCount: workers,
		jobsCh:       make(chan renderJob, workers*16),
		signer:       signer,
		bufferPool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 8192))
			},
		},
	}
	for i := 1; i <= workers; i++ {
		go p.workerLoop(i)
	}
	return p
}

func (p *RenderWorkerPool) Submit(tmpl template.DocumentTemplate) RenderResult {
	ch := make(chan RenderResult, 1)
	canceled := &atomic.Bool{}
	job := renderJob{
		tmpl:     tmpl,
		respCh:   ch,
		canceled: canceled,
	}

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	select {
	case p.jobsCh <- job:
		select {
		case res := <-ch:
			return res
		case <-timer.C:
			canceled.Store(true)
			return RenderResult{Err: ErrPoolBackpressure}
		}
	case <-timer.C:
		canceled.Store(true)
		return RenderResult{Err: ErrPoolBackpressure}
	}
}

func (p *RenderWorkerPool) workerLoop(workerID int) {
	for job := range p.jobsCh {
		if job.canceled != nil && job.canceled.Load() {
			continue
		}
		start := time.Now()
		buf := p.bufferPool.Get().(*bytes.Buffer)
		buf.Reset()

		totalAmount, err := template.ExecuteTemplate(job.tmpl, buf)
		if err != nil {
			p.bufferPool.Put(buf)
			job.respCh <- RenderResult{Err: err}
			continue
		}

		outCopy := make([]byte, buf.Len())
		copy(outCopy, buf.Bytes())
		p.bufferPool.Put(buf)

		if job.canceled != nil && job.canceled.Load() {
			continue
		}

		shaHex, hmacHex := p.signer.ComputeHashes(outCopy)
		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0

		audit := entity.DocumentAuditRecord{
			DocumentHashSHA256: shaHex,
			HmacSignature:      hmacHex,
			DocumentType:       job.tmpl.DocumentType(),
			ReferenceNumber:    job.tmpl.ReferenceNumber(),
			Issuer:             job.tmpl.IssuerName(),
			Recipient:          job.tmpl.RecipientName(),
			TotalAmount:        totalAmount,
			ByteSize:           len(outCopy),
			GeneratedAt:        time.Now().UTC(),
			RenderLatencyMs:    latencyMs,
			WorkerID:           workerID,
		}
		p.signer.RegisterDocument(audit)

		atomic.AddUint64(&p.totalRendered, 1)
		atomic.AddUint64(&p.totalBytes, uint64(len(outCopy)))
		job.respCh <- RenderResult{
			PdfBytes:    outCopy,
			AuditRecord: audit,
		}
	}
}

func (p *RenderWorkerPool) Metrics() map[string]any {
	return map[string]any{
		"workersCount":           p.workersCount,
		"totalDocumentsRendered": atomic.LoadUint64(&p.totalRendered),
		"totalStreamBytes":       atomic.LoadUint64(&p.totalBytes),
		"verifiedRegistrySize":   p.signer.CountRegistered(),
	}
}
