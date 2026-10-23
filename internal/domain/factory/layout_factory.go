package factory

import "fmt"

// Abstract Factory Pattern — Define las dimensiones geométricas y tipográficas
// según el soporte físico del documento (A4 Estándar, Rollo Térmico 80mm o 58mm).
type PageGeometrySpec struct {
	FormatCode     string
	WidthPt        int
	HeightPt       int
	MarginLeftPt   int
	MarginTopPt    int
	TitleFontSize  int
	BodyFontSize   int
	LineHeightPt   int
	DividerColumns int
}

type PageLayoutFactory interface {
	CreateGeometry() PageGeometrySpec
}

type A4PortraitLayoutFactory struct{}

func (f A4PortraitLayoutFactory) CreateGeometry() PageGeometrySpec {
	return PageGeometrySpec{
		FormatCode:     "A4_PORTRAIT",
		WidthPt:        595,
		HeightPt:       842,
		MarginLeftPt:   42,
		MarginTopPt:    790,
		TitleFontSize:  14,
		BodyFontSize:   10,
		LineHeightPt:   15,
		DividerColumns: 74,
	}
}

type Thermal80mmLayoutFactory struct{}

func (f Thermal80mmLayoutFactory) CreateGeometry() PageGeometrySpec {
	return PageGeometrySpec{
		FormatCode:     "THERMAL_80MM",
		WidthPt:        226,
		HeightPt:       650,
		MarginLeftPt:   12,
		MarginTopPt:    625,
		TitleFontSize:  10,
		BodyFontSize:   8,
		LineHeightPt:   12,
		DividerColumns: 38,
	}
}

type Thermal58mmLayoutFactory struct{}

func (f Thermal58mmLayoutFactory) CreateGeometry() PageGeometrySpec {
	return PageGeometrySpec{
		FormatCode:     "THERMAL_58MM",
		WidthPt:        164,
		HeightPt:       600,
		MarginLeftPt:   8,
		MarginTopPt:    575,
		TitleFontSize:  9,
		BodyFontSize:   7,
		LineHeightPt:   11,
		DividerColumns: 28,
	}
}

func ResolveLayoutFactory(format string) PageLayoutFactory {
	switch format {
	case "THERMAL_58MM", "58":
		return Thermal58mmLayoutFactory{}
	case "THERMAL_80MM", "80":
		return Thermal80mmLayoutFactory{}
	default:
		return A4PortraitLayoutFactory{}
	}
}

func MakeDivider(cols int, char rune) string {
	b := make([]rune, cols)
	for i := range b {
		b[i] = char
	}
	return fmt.Sprintf("%s", string(b))
}
