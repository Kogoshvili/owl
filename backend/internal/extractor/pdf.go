package extractor

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

func extractPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	plainText, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract pdf text: %w", err)
	}

	if _, err := io.Copy(&buf, plainText); err != nil {
		return "", fmt.Errorf("read pdf text: %w", err)
	}

	result := buf.String()
	if len(result) > maxTextLength {
		result = result[:maxTextLength]
	}
	return string(bytes.ToValidUTF8([]byte(result), []byte(""))), nil
}
