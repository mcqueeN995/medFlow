// Package pdf извлекает постраничный текст из PDF для RAG-конвейера
// ИИ-карточек (см. internal/pkg/chunker, internal/service/card_service.go).
// Обёртка над github.com/ledongthuc/pdf - чистый Go, без cgo, что важно для
// статической сборки backend-образа (CGO_ENABLED=0 в Dockerfile).
package pdf

import (
	"bytes"
	"errors"
	"strings"

	upstream "github.com/ledongthuc/pdf"
)

// ErrNoExtractableText возвращается, если PDF открылся, но ни на одной
// странице не нашлось текста (например, скан без OCR-слоя) - конвейеру
// карточек нечего эмбеддить/подавать в LLM.
var ErrNoExtractableText = errors.New("pdf: no extractable text")

type Page struct {
	Number int
	Text   string
}

// ExtractPages читает весь PDF в память (документы этого модуля - главы
// учебников/конспекты, не многогигабайтные файлы) и возвращает текст
// постранично; страницы без текста (пустые/только изображения) пропускаются.
func ExtractPages(data []byte) ([]Page, error) {
	r, err := upstream.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var pages []Page
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		pages = append(pages, Page{Number: i, Text: text})
	}

	if len(pages) == 0 {
		return nil, ErrNoExtractableText
	}
	return pages, nil
}
