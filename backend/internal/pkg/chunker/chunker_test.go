package chunker

import (
	"strings"
	"testing"

	"github.com/medflow/backend/internal/pkg/pdf"
)

func TestChunk_ShortPageBecomesOneChunk(t *testing.T) {
	pages := []pdf.Page{{Number: 1, Text: "короткий текст страницы"}}

	chunks := Split(pages, 1500, 200)
	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}
	if chunks[0].PageNumber != 1 {
		t.Errorf("PageNumber = %d, want 1", chunks[0].PageNumber)
	}
	if chunks[0].Content != "короткий текст страницы" {
		t.Errorf("Content = %q, unexpected", chunks[0].Content)
	}
}

func TestChunk_LongPageSplitsWithOverlap(t *testing.T) {
	text := strings.Repeat("а", 1000) + strings.Repeat("б", 1000)
	pages := []pdf.Page{{Number: 3, Text: text}}

	chunks := Split(pages, 1000, 100)
	if len(chunks) < 2 {
		t.Fatalf("len(chunks) = %d, want >= 2 for a 2000-rune page with size=1000", len(chunks))
	}
	for _, c := range chunks {
		if c.PageNumber != 3 {
			t.Errorf("chunk PageNumber = %d, want 3 (chunks must not cross page boundaries)", c.PageNumber)
		}
		if utf8Len(c.Content) > 1000 {
			t.Errorf("chunk length = %d, want <= 1000", utf8Len(c.Content))
		}
	}
	// overlap: конец первого чанка должен пересекаться с началом второго
	if !strings.HasSuffix(chunks[0].Content, chunks[1].Content[:50]) {
		t.Errorf("expected overlap between chunk 0 tail and chunk 1 head")
	}
}

func TestChunk_NeverCrossesPageBoundary(t *testing.T) {
	pages := []pdf.Page{
		{Number: 1, Text: strings.Repeat("x", 50)},
		{Number: 2, Text: strings.Repeat("y", 50)},
	}
	chunks := Split(pages, 1500, 200)
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2 (one per page)", len(chunks))
	}
	if chunks[0].PageNumber != 1 || chunks[1].PageNumber != 2 {
		t.Errorf("page numbers = %d,%d, want 1,2", chunks[0].PageNumber, chunks[1].PageNumber)
	}
}

func TestChunk_EmptyPagesSkipped(t *testing.T) {
	pages := []pdf.Page{{Number: 1, Text: "   "}, {Number: 2, Text: "реальный текст"}}
	chunks := Split(pages, 1500, 200)
	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1 (blank page skipped)", len(chunks))
	}
	if chunks[0].PageNumber != 2 {
		t.Errorf("PageNumber = %d, want 2", chunks[0].PageNumber)
	}
}

func utf8Len(s string) int { return len([]rune(s)) }
