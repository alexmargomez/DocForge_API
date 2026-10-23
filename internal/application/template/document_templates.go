package template

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/alexmargomez/docforge-api/internal/domain/entity"
	"github.com/alexmargomez/docforge-api/internal/domain/factory"
	"github.com/alexmargomez/docforge-api/internal/infrastructure/pdf"
)

// Template Method Pattern —
// Define el flujo invariante de renderizado de cualquier documento transaccional:
// 1. Validate() -> 2. ResolveLayout() -> 3. BuildSections() -> 4. ComputeTotal() -> 5. WriteStream()
type DocumentTemplate interface {
	DocumentType() string
	ReferenceNumber() string
	IssuerName() string
	RecipientName() string
	Validate() error
	LayoutFormat() string
	PopulateBuilder(b *pdf.StreamDocumentBuilder) float64
}

func ExecuteTemplate(tmpl DocumentTemplate, buf *bytes.Buffer) (float64, error) {
	if err := tmpl.Validate(); err != nil {
		return 0, err
	}
	layoutFactory := factory.ResolveLayoutFactory(tmpl.LayoutFormat())
	geom := layoutFactory.CreateGeometry()

	builder := pdf.NewStreamDocumentBuilder(geom)
	total := tmpl.PopulateBuilder(builder)
	builder.WithSecurityFooter(fmt.Sprintf(
		"DocForge Stream Engine | %s | Ref: %s | %s",
		tmpl.DocumentType(),
		tmpl.ReferenceNumber(),
		time.Now().UTC().Format(time.RFC3339),
	))

	if err := builder.WriteToBuffer(buf); err != nil {
		return 0, err
	}
	return total, nil
}

// 1. Implementación Concreta: Factura / Cotización A4
type InvoiceA4Template struct {
	Payload entity.InvoiceA4Payload
}

func (t InvoiceA4Template) DocumentType() string    { return "INVOICE_A4" }
func (t InvoiceA4Template) ReferenceNumber() string { return t.Payload.InvoiceNumber }
func (t InvoiceA4Template) IssuerName() string      { return t.Payload.IssuerName }
func (t InvoiceA4Template) RecipientName() string   { return t.Payload.CustomerName }
func (t InvoiceA4Template) LayoutFormat() string    { return "A4_PORTRAIT" }

func (t InvoiceA4Template) Validate() error {
	if t.Payload.InvoiceNumber == "" || t.Payload.IssuerName == "" {
		return errors.New("invoiceNumber e issuerName son obligatorios")
	}
	if len(t.Payload.Items) == 0 {
		return errors.New("la factura debe incluir al menos un item")
	}
	return nil
}

func (t InvoiceA4Template) PopulateBuilder(b *pdf.StreamDocumentBuilder) float64 {
	b.WithHeader(
		fmt.Sprintf("FACTURA ELECTRONICA / COMERCIAL No. %s", t.Payload.InvoiceNumber),
		fmt.Sprintf("Emisor: %s (NIT: %s)", t.Payload.IssuerName, t.Payload.IssuerNit),
	)
	b.AddDivider('=')
	b.AddKeyValue("Cliente", t.Payload.CustomerName)
	b.AddKeyValue("Documento Cliente", t.Payload.CustomerDoc)
	b.AddKeyValue("Fecha Emision", t.Payload.IssueDate)
	b.AddDivider('-')
	b.AddLine(fmt.Sprintf("%-10s %-28s %6s %12s %12s", "SKU", "DESCRIPCION", "CANT", "V.UNIT", "TOTAL"))
	b.AddDivider('-')

	var subtotal, taxTotal float64
	for _, it := range t.Payload.Items {
		lineBase := it.Quantity * it.UnitPrice
		lineTax := lineBase * (it.TaxPercent / 100.0)
		subtotal += lineBase
		taxTotal += lineTax
		b.AddLine(fmt.Sprintf("%-10s %-28s %6.1f %12.0f %12.0f",
			truncate(it.SKU, 10), truncate(it.Description, 28), it.Quantity, it.UnitPrice, lineBase+lineTax))
	}

	grandTotal := subtotal + taxTotal
	b.AddDivider('=')
	b.AddKeyValue("Subtotal Antes IVA", fmt.Sprintf("$%.0f %s", subtotal, t.Payload.Currency))
	b.AddKeyValue("Impuestos / IVA", fmt.Sprintf("$%.0f %s", taxTotal, t.Payload.Currency))
	b.AddKeyValue("TOTAL A PAGAR", fmt.Sprintf("$%.0f %s", grandTotal, t.Payload.Currency))
	if t.Payload.Notes != "" {
		b.AddDivider('-')
		b.AddLine("Notas: " + t.Payload.Notes)
	}
	return grandTotal
}

// 2. Implementación Concreta: Tirilla Térmica POS (80mm / 58mm)
type PosThermalTicketTemplate struct {
	Payload entity.PosTicketPayload
}

func (t PosThermalTicketTemplate) DocumentType() string    { return "POS_THERMAL_TICKET" }
func (t PosThermalTicketTemplate) ReferenceNumber() string { return t.Payload.TicketNumber }
func (t PosThermalTicketTemplate) IssuerName() string      { return t.Payload.StoreName }
func (t PosThermalTicketTemplate) RecipientName() string   { return "CONSUMIDOR FINAL POS" }
func (t PosThermalTicketTemplate) LayoutFormat() string {
	if t.Payload.PaperWidthMm == 58 {
		return "THERMAL_58MM"
	}
	return "THERMAL_80MM"
}

func (t PosThermalTicketTemplate) Validate() error {
	if t.Payload.TicketNumber == "" || t.Payload.StoreName == "" {
		return errors.New("ticketNumber y storeName son obligatorios")
	}
	if len(t.Payload.Items) == 0 {
		return errors.New("el ticket POS debe incluir al menos un producto")
	}
	return nil
}

func (t PosThermalTicketTemplate) PopulateBuilder(b *pdf.StreamDocumentBuilder) float64 {
	b.WithHeader(t.Payload.StoreName, fmt.Sprintf("NIT: %s | Ticket: %s", t.Payload.StoreNit, t.Payload.TicketNumber))
	b.AddDivider('-')
	b.AddLine(fmt.Sprintf("Cajero: %s | Pago: %s", t.Payload.Cashier, t.Payload.PaymentMode))
	b.AddDivider('-')

	var grandTotal float64
	for _, it := range t.Payload.Items {
		lineTotal := it.Quantity * it.UnitPrice * (1 + it.TaxPercent/100.0)
		grandTotal += lineTotal
		b.AddLine(fmt.Sprintf("%.0fx %-16s $%8.0f", it.Quantity, truncate(it.Description, 16), lineTotal))
	}

	b.AddDivider('=')
	b.AddLine(fmt.Sprintf("TOTAL TIRILLA POS: $%.0f COP", grandTotal))
	return grandTotal
}

// 3. Implementación Concreta: Comprobante / Colilla de Nómina
type PayrollStubTemplate struct {
	Payload entity.PayrollStubPayload
}

func (t PayrollStubTemplate) DocumentType() string    { return "PAYROLL_STUB_A4" }
func (t PayrollStubTemplate) ReferenceNumber() string { return t.Payload.Period + "-" + t.Payload.EmployeeDoc }
func (t PayrollStubTemplate) IssuerName() string      { return t.Payload.CompanyName }
func (t PayrollStubTemplate) RecipientName() string   { return t.Payload.EmployeeName }
func (t PayrollStubTemplate) LayoutFormat() string    { return "A4_PORTRAIT" }

func (t PayrollStubTemplate) Validate() error {
	if t.Payload.EmployeeName == "" || t.Payload.BaseSalaryCOP <= 0 {
		return errors.New("employeeName y baseSalaryCOP (>0) son obligatorios")
	}
	return nil
}

func (t PayrollStubTemplate) PopulateBuilder(b *pdf.StreamDocumentBuilder) float64 {
	b.WithHeader(
		"COMPROBANTE OFICIAL DE PAGO DE NOMINA",
		fmt.Sprintf("Empresa: %s (NIT: %s) | Periodo: %s", t.Payload.CompanyName, t.Payload.CompanyNit, t.Payload.Period),
	)
	b.AddDivider('=')
	b.AddKeyValue("Trabajador", t.Payload.EmployeeName)
	b.AddKeyValue("Documento", t.Payload.EmployeeDoc)
	b.AddKeyValue("Cargo", t.Payload.Position)
	b.AddDivider('-')
	b.AddKeyValue("(+) Salario Basico", fmt.Sprintf("$%.0f COP", t.Payload.BaseSalaryCOP))
	b.AddKeyValue("(+) Aux. Transporte", fmt.Sprintf("$%.0f COP", t.Payload.TransportAuxCOP))
	b.AddKeyValue("(+) Horas Extra/Recargos", fmt.Sprintf("$%.0f COP", t.Payload.OvertimeCOP))
	b.AddKeyValue("(-) Salud (4%)", fmt.Sprintf("-$%.0f COP", t.Payload.HealthDeductCOP))
	b.AddKeyValue("(-) Pension (4%)", fmt.Sprintf("-$%.0f COP", t.Payload.PensionDeductCOP))
	b.AddDivider('=')

	net := (t.Payload.BaseSalaryCOP + t.Payload.TransportAuxCOP + t.Payload.OvertimeCOP) -
		(t.Payload.HealthDeductCOP + t.Payload.PensionDeductCOP)
	b.AddKeyValue("NETO PAGADO AL EMPLEADO", fmt.Sprintf("$%.0f COP", net))
	return net
}

// 4. Implementación Concreta: Fórmula / Orden Médica Estandarizada
type MedicalOrderTemplate struct {
	Payload entity.MedicalOrderPayload
}

func (t MedicalOrderTemplate) DocumentType() string    { return "MEDICAL_PRESCRIPTION_A4" }
func (t MedicalOrderTemplate) ReferenceNumber() string { return t.Payload.OrderNumber }
func (t MedicalOrderTemplate) IssuerName() string      { return t.Payload.InstitutionIPS }
func (t MedicalOrderTemplate) RecipientName() string   { return t.Payload.PatientName }
func (t MedicalOrderTemplate) LayoutFormat() string    { return "A4_PORTRAIT" }

func (t MedicalOrderTemplate) Validate() error {
	if t.Payload.OrderNumber == "" || t.Payload.PatientName == "" || len(t.Payload.Medications) == 0 {
		return errors.New("orderNumber, patientName y medications son obligatorios")
	}
	return nil
}

func (t MedicalOrderTemplate) PopulateBuilder(b *pdf.StreamDocumentBuilder) float64 {
	b.WithHeader(
		fmt.Sprintf("FORMULA MEDICA / ORDEN AMBULATORIA No. %s", t.Payload.OrderNumber),
		fmt.Sprintf("IPS: %s | Dr(a): %s (RM: %s)", t.Payload.InstitutionIPS, t.Payload.DoctorName, t.Payload.MedicalLicense),
	)
	b.AddDivider('=')
	b.AddKeyValue("Paciente", t.Payload.PatientName)
	b.AddKeyValue("Identificacion", t.Payload.PatientDoc)
	b.AddKeyValue("Diagnostico CIE-10", t.Payload.DiagnosisCIE10)
	b.AddDivider('-')
	for idx, m := range t.Payload.Medications {
		b.AddLine(fmt.Sprintf("%d. [%s] %s - %s (%s por %s) Total: %d und",
			idx+1, m.Code, m.Generic, m.Dosage, m.Frequency, m.Duration, m.TotalUnits))
	}
	b.AddDivider('=')
	return float64(len(t.Payload.Medications))
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
