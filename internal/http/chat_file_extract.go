package http

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	maxChatFileBytes          = 10 * 1024 * 1024
	maxExtractedFileTextRunes = 200000
)

func extractChatFileText(ext string, raw []byte) (string, error) {
	var (
		text string
		err  error
	)
	switch ext {
	case ".txt", ".md", ".json", ".csv", ".log":
		text = strings.ToValidUTF8(string(raw), "")
	case ".docx":
		text, err = extractDocxText(raw)
	case ".pdf":
		text, err = extractPDFText(raw)
	default:
		err = errors.New("unsupported chat file type")
	}
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(collapseDocumentWhitespace(text))
	if text == "" {
		return "", errors.New("uploaded file has no readable text content")
	}
	return limitRunes(text, maxExtractedFileTextRunes), nil
}

func extractDocxText(raw []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return "", errors.New("docx file could not be read")
	}
	var b strings.Builder
	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		if name != "word/document.xml" && !strings.HasPrefix(name, "word/header") && !strings.HasPrefix(name, "word/footer") {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", err
		}
		if err := appendDocxXMLText(&b, rc); err != nil {
			_ = rc.Close()
			return "", err
		}
		_ = rc.Close()
	}
	if strings.TrimSpace(b.String()) == "" {
		return "", errors.New("docx file has no readable text content")
	}
	return b.String(), nil
}

func appendDocxXMLText(b *strings.Builder, r io.Reader) error {
	decoder := xml.NewDecoder(r)
	inText := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return errors.New("docx text could not be parsed")
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inText = true
			case "tab":
				b.WriteString("\t")
			case "br", "cr", "p":
				b.WriteString("\n")
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
			}
		case xml.CharData:
			if inText {
				b.Write(t)
			}
		}
	}
}

func extractPDFText(raw []byte) (string, error) {
	chunks := pdfStreamChunks(raw)
	if len(chunks) == 0 {
		chunks = [][]byte{raw}
	}
	var b strings.Builder
	for _, chunk := range chunks {
		text := extractPDFStrings(chunk)
		if text == "" {
			continue
		}
		b.WriteString(text)
		b.WriteString("\n")
	}
	if strings.TrimSpace(b.String()) == "" {
		return "", errors.New("pdf file has no extractable text; scanned PDFs are not supported")
	}
	return b.String(), nil
}

func pdfStreamChunks(raw []byte) [][]byte {
	var chunks [][]byte
	cursor := 0
	for {
		streamIdx := bytes.Index(raw[cursor:], []byte("stream"))
		if streamIdx < 0 {
			break
		}
		streamIdx += cursor
		dataStart := streamIdx + len("stream")
		if dataStart < len(raw) && raw[dataStart] == '\r' {
			dataStart++
		}
		if dataStart < len(raw) && raw[dataStart] == '\n' {
			dataStart++
		}
		endIdx := bytes.Index(raw[dataStart:], []byte("endstream"))
		if endIdx < 0 {
			break
		}
		dataEnd := dataStart + endIdx
		dictStart := streamIdx - 2048
		if dictStart < 0 {
			dictStart = 0
		}
		dict := raw[dictStart:streamIdx]
		chunk := raw[dataStart:dataEnd]
		if bytes.Contains(dict, []byte("/FlateDecode")) || bytes.Contains(dict, []byte("/Fl")) {
			if decoded, err := inflatePDFStream(chunk); err == nil {
				chunk = decoded
			}
		}
		chunks = append(chunks, chunk)
		cursor = dataEnd + len("endstream")
	}
	return chunks
}

func inflatePDFStream(raw []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(bytes.TrimSpace(raw)))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			log.Printf("error closing reader: %v", err)
		}
	}()
	return io.ReadAll(io.LimitReader(reader, maxChatFileBytes))
}

var pdfSpacePattern = regexp.MustCompile(`\s+`)

func extractPDFStrings(raw []byte) string {
	var parts []string
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '(':
			text, next := parsePDFLiteral(raw, i+1)
			i = next
			text = decodePDFTextBytes([]byte(text))
			if looksReadable(text) {
				parts = append(parts, text)
			}
		case '<':
			if i+1 < len(raw) && raw[i+1] == '<' {
				continue
			}
			end := bytes.IndexByte(raw[i+1:], '>')
			if end < 0 {
				continue
			}
			encoded := raw[i+1 : i+1+end]
			i = i + 1 + end
			decoded, err := hex.DecodeString(cleanPDFHex(encoded))
			if err != nil {
				continue
			}
			text := decodePDFTextBytes(decoded)
			if looksReadable(text) {
				parts = append(parts, text)
			}
		}
	}
	return strings.TrimSpace(pdfSpacePattern.ReplaceAllString(strings.Join(parts, " "), " "))
}

func parsePDFLiteral(raw []byte, start int) (string, int) {
	var out []byte
	depth := 1
	for i := start; i < len(raw); i++ {
		ch := raw[i]
		if ch == '\\' && i+1 < len(raw) {
			next := raw[i+1]
			switch next {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case '\\', '(', ')':
				out = append(out, next)
			case '\r', '\n':
			default:
				out = append(out, next)
			}
			i++
			continue
		}
		if ch == '(' {
			depth++
		}
		if ch == ')' {
			depth--
			if depth == 0 {
				return string(out), i
			}
		}
		out = append(out, ch)
	}
	return string(out), len(raw) - 1
}

func cleanPDFHex(raw []byte) string {
	var b strings.Builder
	for _, ch := range raw {
		if unicode.IsSpace(rune(ch)) {
			continue
		}
		b.WriteByte(ch)
	}
	out := b.String()
	if len(out)%2 == 1 {
		out += "0"
	}
	return out
}

func decodePDFTextBytes(raw []byte) string {
	if len(raw) >= 2 && raw[0] == 0xfe && raw[1] == 0xff {
		units := make([]uint16, 0, (len(raw)-2)/2)
		for i := 2; i+1 < len(raw); i += 2 {
			units = append(units, uint16(raw[i])<<8|uint16(raw[i+1]))
		}
		return string(utf16.Decode(units))
	}
	return strings.ToValidUTF8(string(raw), "")
}

func looksReadable(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	printable := 0
	total := 0
	for _, r := range text {
		total++
		if r == utf8.RuneError {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			printable++
		}
	}
	return total > 0 && printable*100/total >= 70
}

func collapseDocumentWhitespace(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func limitRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}
