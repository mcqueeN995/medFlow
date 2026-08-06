// Package chunker режет постраничный текст PDF (см. internal/pkg/pdf) на
// перекрывающиеся куски для эмбеддинга и векторного поиска в RAG-конвейере
// ИИ-карточек.
package chunker

import (
	"strings"

	"github.com/medflow/backend/internal/pkg/pdf"
)

const (
	DefaultSize    = 1500
	DefaultOverlap = 200
)

type Chunk struct {
	PageNumber int
	Content    string
}

// Split режет каждую страницу независимо (чанк никогда не пересекает границу
// страницы - иначе page_number чанка перестаёт быть однозначным), кусками по
// size рун с overlap рун перекрытия между соседними кусками одной страницы.
// Страница короче size целиком становится одним чанком.
func Split(pages []pdf.Page, size, overlap int) []Chunk {
	if size <= 0 {
		size = DefaultSize
	}
	if overlap < 0 || overlap >= size {
		overlap = DefaultOverlap
	}

	var out []Chunk
	for _, page := range pages {
		runes := []rune(page.Text)
		if len(runes) == 0 {
			continue
		}

		step := size - overlap
		for start := 0; start < len(runes); start += step {
			end := min(start+size, len(runes))
			content := strings.TrimSpace(string(runes[start:end]))
			if content != "" {
				out = append(out, Chunk{PageNumber: page.Number, Content: content})
			}
			if end == len(runes) {
				break
			}
		}
	}
	return out
}
