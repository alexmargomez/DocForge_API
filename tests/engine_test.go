package tests

import (
	"bytes"
	"sync"
	"testing"

	"github.com/alexmargomez/docforge-api/internal/application/pool"
	"github.com/alexmargomez/docforge-api/internal/application/template"
	"github.com/alexmargomez/docforge-api/internal/domain/entity"
	"github.com/alexmargomez/docforge-api/internal/domain/service"
	"github.com/alexmargomez/docforge-api/internal/infrastructure/qr"
)

func TestInvoiceA4StreamAndSignatureVerification(t *testing.T) {
	signer := service.NewHashSignerService("test_secret_key")
	workerPool := pool.NewRenderWorkerPool(4, signer)

	res := workerPool.Submit(template.InvoiceA4Template{
		Payload: entity.InvoiceA4Payload{
			InvoiceNumber: "FE-2026-0091",
			IssuerName:    "Alexmar Software SAS",
			IssuerNit:     "901456789-2",
			CustomerName:  "Clínica Regional",
			CustomerDoc:   "800123456-1",
			Currency:      "COP",
			IssueDate:     "2026-10-23",
			Items: []entity.InvoiceItem{
				{SKU: "LIC-01", Description: "Licencia Anual Motor FHIR", Quantity: 1, UnitPrice: 3500000, TaxPercent: 19},
			},
		},
	})

	if res.Err != nil {
		t.Fatalf("error inesperado generando PDF en stream: %v", res.Err)
	}
	if !bytes.HasPrefix(res.PdfBytes, []byte("%PDF-1.4")) {
		t.Fatalf("el binario generado no tiene cabecera PDF 1.4 válida")
	}
	if res.AuditRecord.TotalAmount != 4165000 {
		t.Fatalf("total incorrecto: esperado 4165000, obtenido %.0f", res.AuditRecord.TotalAmount)
	}

	verifiedRec, ok := signer.VerifyHash(res.AuditRecord.DocumentHashSHA256)
	if !ok || verifiedRec.ReferenceNumber != "FE-2026-0091" {
		t.Fatalf("falló la verificación criptográfica SHA-256 del documento emitido")
	}
}

func TestConcurrentWorkerPoolPosTickets(t *testing.T) {
	signer := service.NewHashSignerService("concurrent_secret")
	workerPool := pool.NewRenderWorkerPool(8, signer)

	const totalRequests = 100
	var wg sync.WaitGroup
	wg.Add(totalRequests)

	for i := 0; i < totalRequests; i++ {
		go func(idx int) {
			defer wg.Done()
			res := workerPool.Submit(template.PosThermalTicketTemplate{
				Payload: entity.PosTicketPayload{
					PaperWidthMm: 80,
					StoreName:    "FactuPOS Taller Central",
					StoreNit:     "900888777-1",
					TicketNumber: "POS-800",
					Cashier:      "Caja 1",
					PaymentMode:  "EFECTIVO",
					Items: []entity.InvoiceItem{
						{SKU: "ACE-10W40", Description: "Aceite Sintetico", Quantity: 2, UnitPrice: 45000, TaxPercent: 19},
					},
				},
			})
			if res.Err != nil || len(res.PdfBytes) < 150 {
				t.Errorf("fallo en goroutine #%d: %v", idx, res.Err)
			}
		}(i)
	}
	wg.Wait()
}

func TestVectorQRGeneration(t *testing.T) {
	res := qr.GenerateVectorQR("https://github.com/alexmargomez", 240)
	if res.GridDimension != 21 || len(res.SvgMarkup) < 200 {
		t.Fatalf("QR vectorial inválido")
	}
}
