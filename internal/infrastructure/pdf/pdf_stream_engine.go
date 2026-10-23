package pdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alexmargomez/docforge-api/internal/domain/factory"
)

// Fluent Builder Pattern + Zero-Disk Stream Writer —
// Construye un documento PDF 1.4 válido directamente sobre un *bytes.Buffer reciclado en memoria.
type StreamDocumentBuilder struct {
	geom       factory.PageGeometrySpec
	title      string
	subtitle   string
	lines      []string
	footerSeal string
}

func NewStreamDocumentBuilder(geom factory.PageGeometrySpec) *StreamDocumentBuilder {
	return &StreamDocumentBuilder{
		geom:  geom,
		lines: make([]string, 0, 48),
	}
}

func (b *StreamDocumentBuilder) WithHeader(title, subtitle string) *StreamDocumentBuilder {
	b.title = sanitizePdfText(title)
	b.subtitle = sanitizePdfText(subtitle)
	return b
}

func (b *StreamDocumentBuilder) AddDivider(char rune) *StreamDocumentBuilder {
	b.lines = append(b.lines, factory.MakeDivider(b.geom.DividerColumns, char))
	return b
}

func (b *StreamDocumentBuilder) AddLine(text string) *StreamDocumentBuilder {
	b.lines = append(b.lines, sanitizePdfText(text))
	return b
}

func (b *StreamDocumentBuilder) AddKeyValue(label, value string) *StreamDocumentBuilder {
	b.lines = append(b.lines, sanitizePdfText(fmt.Sprintf("%-22s : %s", label, value)))
	return b
}

func (b *StreamDocumentBuilder) WithSecurityFooter(seal string) *StreamDocumentBuilder {
	b.footerSeal = sanitizePdfText(seal)
	return b
}

func (b *StreamDocumentBuilder) WriteToBuffer(buf *bytes.Buffer) error {
	buf.Reset()

	var contentStream bytes.Buffer
	contentStream.WriteString("BT\n")
	contentStream.WriteString(fmt.Sprintf("/F1 %d Tf\n", b.geom.TitleFontSize))
	contentStream.WriteString(fmt.Sprintf("%d %d Td\n", b.geom.MarginLeftPt, b.geom.MarginTopPt))
	contentStream.WriteString(fmt.Sprintf("(%s) Tj\n", b.title))

	contentStream.WriteString(fmt.Sprintf("0 -%d Td\n", b.geom.LineHeightPt+2))
	contentStream.WriteString(fmt.Sprintf("/F1 %d Tf\n", b.geom.BodyFontSize))
	if b.subtitle != "" {
		contentStream.WriteString(fmt.Sprintf("(%s) Tj\n", b.subtitle))
		contentStream.WriteString(fmt.Sprintf("0 -%d Td\n", b.geom.LineHeightPt))
	}

	for _, line := range b.lines {
		contentStream.WriteString(fmt.Sprintf("(%s) Tj\n", line))
		contentStream.WriteString(fmt.Sprintf("0 -%d Td\n", b.geom.LineHeightPt))
	}

	if b.footerSeal != "" {
		contentStream.WriteString(fmt.Sprintf("0 -%d Td\n", b.geom.LineHeightPt))
		contentStream.WriteString(fmt.Sprintf("(%s) Tj\n", b.footerSeal))
	}
	contentStream.WriteString("ET\n")

	streamBytes := contentStream.Bytes()

	offsets := make([]int, 6)
	buf.WriteString("%PDF-1.4\n")

	offsets[1] = buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	offsets[2] = buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	offsets[3] = buf.Len()
	buf.WriteString(fmt.Sprintf(
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>\nendobj\n",
		b.geom.WidthPt, b.geom.HeightPt,
	))

	offsets[4] = buf.Len()
	buf.WriteString("4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>\nendobj\n")

	offsets[5] = buf.Len()
	buf.WriteString(fmt.Sprintf("5 0 obj\n<< /Length %d >>\nstream\n", len(streamBytes)))
	buf.Write(streamBytes)
	buf.WriteString("endstream\nendobj\n")

	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset))
	return nil
}

func sanitizePdfText(in string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"(", "\\(",
		")", "\\)",
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U",
		"ñ", "n", "Ñ", "N",
	)
	return replacer.Replace(in)
}
