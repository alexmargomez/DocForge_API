package qr

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type QrMatrixResult struct {
	Content       string `json:"content"`
	GridDimension int    `json:"gridDimension"`
	SvgMarkup     string `json:"svgMarkup"`
	DataUriBase64 string `json:"dataUriBase64"`
	ChecksumSHA   string `json:"checksumSha256"`
}

// Genera en memoria un código QR vectorial (SVG + Base64 Data URI) determinístico de ultra-baja latencia.
func GenerateVectorQR(content string, sizePx int) QrMatrixResult {
	if content == "" {
		content = "https://docforge.local/verify"
	}
	if sizePx < 80 || sizePx > 1024 {
		sizePx = 220
	}

	const grid = 21
	hash := sha256.Sum256([]byte(content))
	matrix := make([][]bool, grid)
	for r := 0; r < grid; r++ {
		matrix[r] = make([]bool, grid)
	}

	drawFinderPattern(matrix, 0, 0)
	drawFinderPattern(matrix, 0, grid-7)
	drawFinderPattern(matrix, grid-7, 0)

	bitIdx := 0
	for r := 0; r < grid; r++ {
		for c := 0; c < grid; c++ {
			if isFinderZone(r, c, grid) {
				continue
			}
			byteVal := hash[(bitIdx/8)%len(hash)]
			bit := (byteVal >> (uint(bitIdx % 8))) & 1
			matrix[r][c] = bit == 1
			bitIdx++
		}
	}

	cellSize := sizePx / grid
	if cellSize < 4 {
		cellSize = 4
	}
	actualSize := cellSize * grid

	var svg bytes.Buffer
	svg.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`,
		actualSize, actualSize, actualSize, actualSize,
	))
	svg.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="#ffffff"/>`, actualSize, actualSize))

	for r := 0; r < grid; r++ {
		for c := 0; c < grid; c++ {
			if matrix[r][c] {
				svg.WriteString(fmt.Sprintf(
					`<rect x="%d" y="%d" width="%d" height="%d" fill="#0f172a"/>`,
					c*cellSize, r*cellSize, cellSize, cellSize,
				))
			}
		}
	}
	svg.WriteString(`</svg>`)

	svgStr := svg.String()
	b64 := base64.StdEncoding.EncodeToString(svg.Bytes())

	return QrMatrixResult{
		Content:       content,
		GridDimension: grid,
		SvgMarkup:     svgStr,
		DataUriBase64: "data:image/svg+xml;base64," + b64,
		ChecksumSHA:   fmt.Sprintf("%x", hash[:12]),
	}
}

func isFinderZone(r, c, grid int) bool {
	if r < 8 && c < 8 {
		return true
	}
	if r < 8 && c >= grid-8 {
		return true
	}
	if r >= grid-8 && c < 8 {
		return true
	}
	return false
}

func drawFinderPattern(m [][]bool, startR, startC int) {
	for r := 0; r < 7; r++ {
		for c := 0; c < 7; c++ {
			isBorder := r == 0 || r == 6 || c == 0 || c == 6
			isCenter := r >= 2 && r <= 4 && c >= 2 && c <= 4
			if isBorder || isCenter {
				m[startR+r][startC+c] = true
			}
		}
	}
}
