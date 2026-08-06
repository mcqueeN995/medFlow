package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// buildTestPDF собирает минимальный валидный однострочный PDF с одной
// страницей текста, вычисляя реальные байтовые offset'ы для xref-таблицы -
// ledongthuc/pdf не восстанавливает файлы с некорректным xref, так что
// оффсеты "на глазок" здесь недопустимы.
func buildTestPDF(t *testing.T, text string) []byte {
	t.Helper()

	var buf bytes.Buffer
	offsets := make([]int, 0, 5)

	writeObj := func(body string) {
		offsets = append(offsets, buf.Len())
		buf.WriteString(body)
	}

	buf.WriteString("%PDF-1.4\n")
	writeObj("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	writeObj("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	writeObj("3 0 obj\n<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 4 0 R >> >> /MediaBox [0 0 300 300] /Contents 5 0 R >>\nendobj\n")
	writeObj("4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	content := fmt.Sprintf("BT /F1 18 Tf 10 100 Td (%s) Tj ET", text)
	writeObj(fmt.Sprintf("5 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(content), content))

	xrefStart := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(offsets)+1))
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(offsets)+1, xrefStart))

	return buf.Bytes()
}

func TestExtractPages_SinglePageWithText(t *testing.T) {
	data := buildTestPDF(t, "Hello World")

	pages, err := ExtractPages(data)
	if err != nil {
		t.Fatalf("ExtractPages() error = %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("len(pages) = %d, want 1", len(pages))
	}
	if pages[0].Number != 1 {
		t.Errorf("pages[0].Number = %d, want 1", pages[0].Number)
	}
	if !strings.Contains(pages[0].Text, "Hello World") {
		t.Errorf("pages[0].Text = %q, want to contain %q", pages[0].Text, "Hello World")
	}
}

func TestExtractPages_InvalidPDF(t *testing.T) {
	_, err := ExtractPages([]byte("not a pdf"))
	if err == nil {
		t.Fatal("ExtractPages() error = nil, want non-nil for invalid PDF")
	}
}
