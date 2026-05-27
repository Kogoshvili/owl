package extractor

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	_ "image/gif"
	"io"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
	"owl/internal/store"
)

func extractMetadata(f *store.File) map[string]any {
	ext := strings.ToLower(f.Extension)
	meta := make(map[string]any)

	switch {
	case isPlainText(ext):
		extractTextMeta(f.Path, meta)
	case isImageExt(ext):
		extractImageMeta(f.Path, meta)
	case ext == ".svg":
		extractSVGMeta(f.Path, meta)
	case ext == ".pdf":
		extractPDFMeta(f.Path, meta)
	case ext == ".docx":
		extractOfficeCoreMeta(f.Path, meta)
	case ext == ".xlsx":
		extractOfficeCoreMeta(f.Path, meta)
	case ext == ".pptx":
		extractOfficeCoreMeta(f.Path, meta)
	}

	return meta
}

func extractTextMeta(path string, meta map[string]any) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	text := string(bytes.ToValidUTF8(data, []byte("")))
	meta["lines"] = strings.Count(text, "\n") + 1
	meta["characters"] = len(text)
	words := len(strings.Fields(text))
	meta["words"] = words
}

func isImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return true
	}
	return false
}

func extractImageMeta(path string, meta map[string]any) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return
	}
	meta["width"] = cfg.Width
	meta["height"] = cfg.Height
}

func extractSVGMeta(path string, meta map[string]any) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
		if se, ok := token.(xml.StartElement); ok && se.Name.Local == "svg" {
			for _, attr := range se.Attr {
				switch attr.Name.Local {
				case "width":
					meta["width"] = parseSVGDimension(attr.Value)
				case "height":
					meta["height"] = parseSVGDimension(attr.Value)
				case "viewBox":
					parts := strings.Fields(attr.Value)
					if len(parts) == 4 {
						meta["width"] = parseSVGDimension(parts[2])
						meta["height"] = parseSVGDimension(parts[3])
					}
				}
			}
			return
		}
	}
}

func parseSVGDimension(s string) any {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "px")
	s = strings.TrimSuffix(s, "pt")
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
		return int(f)
	}
	return s
}

func extractPDFMeta(path string, meta map[string]any) {
	f, r, err := openPDFFile(path)
	if err != nil {
		return
	}
	defer f.Close()

	meta["pages"] = r.NumPage()
}

func openPDFFile(path string) (*os.File, *pdf.Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}

	r, err := pdf.NewReader(f, stat.Size())
	if err != nil {
		f.Close()
		return nil, nil, err
	}

	return f, r, nil
}

func extractOfficeCoreMeta(path string, meta map[string]any) {
	rc, err := zip.OpenReader(path)
	if err != nil {
		return
	}
	defer rc.Close()

	for _, f := range rc.File {
		if f.Name == "docProps/core.xml" {
			parseCoreXML(f, meta)
			return
		}
	}
}

func parseCoreXML(f *zip.File, meta map[string]any) {
	r, err := f.Open()
	if err != nil {
		return
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var currentElement string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
		switch t := token.(type) {
		case xml.StartElement:
			currentElement = t.Name.Local
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text == "" {
				continue
			}
			switch currentElement {
			case "title":
				meta["title"] = text
			case "creator":
				meta["author"] = text
			case "subject":
				meta["subject"] = text
			case "description":
				meta["description"] = text
			case "keywords":
				meta["keywords"] = text
			case "lastModifiedBy":
				meta["lastModifiedBy"] = text
			case "revision":
				meta["revision"] = text
			}
		case xml.EndElement:
			currentElement = ""
		}
	}
}
