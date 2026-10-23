package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexmargomez/docforge-api/internal/application/pool"
	"github.com/alexmargomez/docforge-api/internal/application/template"
	"github.com/alexmargomez/docforge-api/internal/domain/entity"
	"github.com/alexmargomez/docforge-api/internal/domain/service"
	"github.com/alexmargomez/docforge-api/internal/infrastructure/qr"
)

type ServerHandler struct {
	workerPool *pool.RenderWorkerPool
	signer     *service.HashSignerService
}

func NewServerHandler(workerPool *pool.RenderWorkerPool, signer *service.HashSignerService) *ServerHandler {
	return &ServerHandler{
		workerPool: workerPool,
		signer:     signer,
	}
}

func (h *ServerHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", h.handleDocsUI)
	mux.HandleFunc("/docs", h.handleDocsUI)
	mux.HandleFunc("/api/v1/openapi.json", h.handleOpenAPISpec)
	mux.HandleFunc("/api/v1/health", h.handleHealth)
	mux.HandleFunc("/api/v1/documents/invoice-a4", h.handleInvoiceA4)
	mux.HandleFunc("/api/v1/documents/pos-ticket", h.handlePosTicket)
	mux.HandleFunc("/api/v1/documents/payroll-stub", h.handlePayrollStub)
	mux.HandleFunc("/api/v1/documents/medical-order", h.handleMedicalOrder)
	mux.HandleFunc("/api/v1/documents/qr", h.handleQrGenerate)
	mux.HandleFunc("/api/v1/documents/verify/", h.handleVerifyHash)
}

func (h *ServerHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "UP",
		"service":      "DocForge Stream API (Go 1.22)",
		"architecture": "Pipeline & Filters + Hexagonal + Worker Pool (sync.Pool)",
		"metrics":      h.workerPool.Metrics(),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *ServerHandler) handleInvoiceA4(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Use POST"})
		return
	}
	var payload entity.InvoiceA4Payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: " + err.Error()})
		return
	}
	h.dispatchTemplateResponse(w, r, template.InvoiceA4Template{Payload: payload}, "invoice-"+payload.InvoiceNumber+".pdf")
}

func (h *ServerHandler) handlePosTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Use POST"})
		return
	}
	var payload entity.PosTicketPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: " + err.Error()})
		return
	}
	h.dispatchTemplateResponse(w, r, template.PosThermalTicketTemplate{Payload: payload}, "pos-ticket-"+payload.TicketNumber+".pdf")
}

func (h *ServerHandler) handlePayrollStub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Use POST"})
		return
	}
	var payload entity.PayrollStubPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: " + err.Error()})
		return
	}
	h.dispatchTemplateResponse(w, r, template.PayrollStubTemplate{Payload: payload}, "payroll-"+payload.EmployeeDoc+".pdf")
}

func (h *ServerHandler) handleMedicalOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Use POST"})
		return
	}
	var payload entity.MedicalOrderPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: " + err.Error()})
		return
	}
	h.dispatchTemplateResponse(w, r, template.MedicalOrderTemplate{Payload: payload}, "rx-"+payload.OrderNumber+".pdf")
}

func (h *ServerHandler) handleQrGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Use POST"})
		return
	}
	var payload entity.QrRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido: " + err.Error()})
		return
	}
	res := qr.GenerateVectorQR(payload.Content, payload.SizePx)
	writeJSON(w, http.StatusOK, res)
}

func (h *ServerHandler) handleVerifyHash(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimPrefix(r.URL.Path, "/api/v1/documents/verify/")
	hash = strings.TrimSpace(hash)
	if hash == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Debe especificar el hash SHA-256 a verificar"})
		return
	}
	rec, ok := h.signer.VerifyHash(hash)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"verified": false,
			"hash":     hash,
			"message":  "No existe ningún documento emitido con ese hash SHA-256 en el registro activo.",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"verified": true,
		"record":   rec,
	})
}

func (h *ServerHandler) dispatchTemplateResponse(
	w http.ResponseWriter,
	r *http.Request,
	tmpl template.DocumentTemplate,
	filename string,
) {
	res := h.workerPool.Submit(tmpl)
	if res.Err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": res.Err.Error()})
		return
	}

	w.Header().Set("X-Document-SHA256", res.AuditRecord.DocumentHashSHA256)
	w.Header().Set("X-Document-HMAC", res.AuditRecord.HmacSignature)
	w.Header().Set("X-Render-Latency-Ms", fmt.Sprintf("%.3f", res.AuditRecord.RenderLatencyMs))
	w.Header().Set("X-Pool-Worker-Id", fmt.Sprintf("%d", res.AuditRecord.WorkerID))

	if r.URL.Query().Get("format") == "json" {
		writeJSON(w, http.StatusOK, map[string]any{
			"audit":       res.AuditRecord,
			"pdfBase64":   base64.StdEncoding.EncodeToString(res.PdfBytes),
			"downloadUrl": "data:application/pdf;base64," + base64.StdEncoding.EncodeToString(res.PdfBytes),
		})
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res.PdfBytes)
}

func (h *ServerHandler) handleDocsUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/docs" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html>
  <head>
    <title>DocForge API — Go 1.22 Stream PDF & QR Engine (OpenAPI 3.1)</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script id="api-reference" data-url="/api/v1/openapi.json" data-configuration='{"theme":"kepler","darkMode":true}'></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`))
}

func (h *ServerHandler) handleOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	spec := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "DocForge API — Motor de Generación de Documentos PDF, Tirillas POS (80mm) y QR en Stream (Go 1.22)",
			"version":     "1.0.0",
			"description": "Microservicio 100% Backend en Go (Golang 1.22) que renderiza facturas A4, tirillas térmicas POS (80mm/58mm), colillas de nómina y órdenes médicas directamente en streams de memoria (*bytes.Buffer + sync.Pool) con firma criptográfica SHA-256 / HMAC-SHA256 y cero escrituras en disco.",
		},
		"paths": map[string]any{
			"/api/v1/health": map[string]any{
				"get": map[string]any{
					"summary": "Estado de salud y métricas del Worker Pool en Go",
					"tags":    []string{"1. Telemetría & Salud"},
				},
			},
			"/api/v1/documents/invoice-a4": map[string]any{
				"post": map[string]any{
					"summary": "Generar Factura / Cotización Comercial en PDF A4 en Stream (?format=json para Base64 + Auditoría)",
					"tags":    []string{"2. Renderizado PDF en Stream"},
				},
			},
			"/api/v1/documents/pos-ticket": map[string]any{
				"post": map[string]any{
					"summary": "Generar Tirilla Térmica POS (80mm o 58mm) en Stream",
					"tags":    []string{"2. Renderizado PDF en Stream"},
				},
			},
			"/api/v1/documents/payroll-stub": map[string]any{
				"post": map[string]any{
					"summary": "Generar Comprobante / Colilla de Pago de Nómina en PDF A4",
					"tags":    []string{"2. Renderizado PDF en Stream"},
				},
			},
			"/api/v1/documents/medical-order": map[string]any{
				"post": map[string]any{
					"summary": "Generar Fórmula / Orden Médica Estandarizada en PDF A4",
					"tags":    []string{"2. Renderizado PDF en Stream"},
				},
			},
			"/api/v1/documents/qr": map[string]any{
				"post": map[string]any{
					"summary": "Generar Código QR Vectorial (SVG + Base64 Data URI) en memoria",
					"tags":    []string{"3. Motor QR & Verificación Criptográfica"},
				},
			},
			"/api/v1/documents/verify/{hash}": map[string]any{
				"get": map[string]any{
					"summary": "Verificar autenticidad e integridad de un documento emitido por su Hash SHA-256",
					"tags":    []string{"3. Motor QR & Verificación Criptográfica"},
				},
			},
		},
	}
	writeJSON(w, http.StatusOK, spec)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
