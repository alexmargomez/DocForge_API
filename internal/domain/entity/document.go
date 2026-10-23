package entity

import "time"

type InvoiceItem struct {
	SKU         string  `json:"sku"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	TaxPercent  float64 `json:"taxPercent"`
}

type InvoiceA4Payload struct {
	InvoiceNumber string        `json:"invoiceNumber"`
	IssuerName    string        `json:"issuerName"`
	IssuerNit     string        `json:"issuerNit"`
	CustomerName  string        `json:"customerName"`
	CustomerDoc   string        `json:"customerDoc"`
	Currency      string        `json:"currency"`
	IssueDate     string        `json:"issueDate"`
	Notes         string        `json:"notes"`
	Items         []InvoiceItem `json:"items"`
}

type PosTicketPayload struct {
	PaperWidthMm int           `json:"paperWidthMm"` // 80 or 58
	StoreName    string        `json:"storeName"`
	StoreNit     string        `json:"storeNit"`
	TicketNumber string        `json:"ticketNumber"`
	Cashier      string        `json:"cashier"`
	PaymentMode  string        `json:"paymentMode"`
	Items        []InvoiceItem `json:"items"`
}

type PayrollStubPayload struct {
	Period           string  `json:"period"`
	CompanyName      string  `json:"companyName"`
	CompanyNit       string  `json:"companyNit"`
	EmployeeName     string  `json:"employeeName"`
	EmployeeDoc      string  `json:"employeeDoc"`
	Position         string  `json:"position"`
	BaseSalaryCOP    float64 `json:"baseSalaryCOP"`
	TransportAuxCOP  float64 `json:"transportAuxCOP"`
	OvertimeCOP      float64 `json:"overtimeCOP"`
	HealthDeductCOP  float64 `json:"healthDeductCOP"`
	PensionDeductCOP float64 `json:"pensionDeductCOP"`
}

type MedicalMedicationItem struct {
	Code       string `json:"code"`
	Generic    string `json:"generic"`
	Dosage     string `json:"dosage"`
	Frequency  string `json:"frequency"`
	Duration   string `json:"duration"`
	TotalUnits int    `json:"totalUnits"`
}

type MedicalOrderPayload struct {
	OrderNumber    string                  `json:"orderNumber"`
	InstitutionIPS string                  `json:"institutionIPS"`
	DoctorName     string                  `json:"doctorName"`
	MedicalLicense string                  `json:"medicalLicense"`
	PatientName    string                  `json:"patientName"`
	PatientDoc     string                  `json:"patientDoc"`
	DiagnosisCIE10 string                  `json:"diagnosisCIE10"`
	Medications    []MedicalMedicationItem `json:"medications"`
}

type QrRequestPayload struct {
	Content string `json:"content"`
	SizePx  int    `json:"sizePx"`
}

type DocumentAuditRecord struct {
	DocumentHashSHA256 string    `json:"documentHashSha256"`
	HmacSignature      string    `json:"hmacSignature"`
	DocumentType       string    `json:"documentType"`
	ReferenceNumber    string    `json:"referenceNumber"`
	Issuer             string    `json:"issuer"`
	Recipient          string    `json:"recipient"`
	TotalAmount        float64   `json:"totalAmount"`
	ByteSize           int       `json:"byteSize"`
	GeneratedAt        time.Time `json:"generatedAt"`
	RenderLatencyMs    float64   `json:"renderLatencyMs"`
	WorkerID           int       `json:"workerId"`
}
