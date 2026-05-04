package http

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestExtractChatFileTextFromDocx(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>OneDrive 文档内容</w:t></w:r></w:p></w:body>
</w:document>`)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	text, err := extractChatFileText(".docx", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "OneDrive 文档内容") {
		t.Fatalf("expected docx text, got %q", text)
	}
}

func TestExtractChatFileTextFromPDFLiteralText(t *testing.T) {
	raw := []byte(`%PDF-1.4
1 0 obj
<< /Length 44 >>
stream
BT /F1 12 Tf 72 720 Td (PDF file content) Tj ET
endstream
endobj
%%EOF`)
	text, err := extractChatFileText(".pdf", raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "PDF file content") {
		t.Fatalf("expected pdf text, got %q", text)
	}
}
