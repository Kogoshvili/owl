package extractor

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

func extractDOCX(path string) (string, error) {
	rc, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer rc.Close()

	var buf strings.Builder
	for _, f := range rc.File {
		if f.Name == "word/document.xml" {
			text, err := extractXMLText(f, "t")
			if err != nil {
				return "", err
			}
			buf.WriteString(text)
		}
	}

	result := buf.String()
	if len(result) > maxTextLength {
		result = result[:maxTextLength]
	}
	return result, nil
}

func extractXLSX(path string) (string, error) {
	rc, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open xlsx: %w", err)
	}
	defer rc.Close()

	sharedStrings, err := parseSharedStrings(rc.File)
	if err != nil {
		return "", fmt.Errorf("parse shared strings: %w", err)
	}

	var sheetFiles []string
	for _, f := range rc.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			sheetFiles = append(sheetFiles, f.Name)
		}
	}
	sort.Strings(sheetFiles)

	var buf strings.Builder
	for _, name := range sheetFiles {
		for _, f := range rc.File {
			if f.Name == name {
				cells, err := extractXLSXCells(f, sharedStrings)
				if err != nil {
					continue
				}
				for _, cell := range cells {
					buf.WriteString(cell)
					buf.WriteByte(' ')
				}
			}
		}
	}

	result := buf.String()
	if len(result) > maxTextLength {
		result = result[:maxTextLength]
	}
	return result, nil
}

func parseSharedStrings(files []*zip.File) ([]string, error) {
	for _, f := range files {
		if f.Name == "xl/sharedStrings.xml" {
			r, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer r.Close()
			return parseSST(r)
		}
	}
	return nil, nil
}

func parseSST(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var strings_ []string
	var inT bool
	var current string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inT = true
				current = ""
			}
		case xml.CharData:
			if inT {
				current += string(t)
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inT = false
				strings_ = append(strings_, current)
			}
		}
	}
	return strings_, nil
}

func extractXLSXCells(f *zip.File, sharedStrings []string) ([]string, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var cells []string
	var inV bool
	var currentType string
	var currentV string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "c":
				currentType = ""
				currentV = ""
				for _, attr := range t.Attr {
					if attr.Name.Local == "t" {
						currentType = attr.Value
					}
				}
			case "v":
				inV = true
				currentV = ""
			}
		case xml.CharData:
			if inV {
				currentV += string(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v":
				inV = false
			case "c":
				if currentV != "" {
					if currentType == "s" && sharedStrings != nil {
						var idx int
						if _, err := fmt.Sscanf(currentV, "%d", &idx); err == nil && idx < len(sharedStrings) {
							cells = append(cells, sharedStrings[idx])
						}
					} else {
						cells = append(cells, currentV)
					}
				}
			}
		}
	}
	return cells, nil
}

func extractPPTX(path string) (string, error) {
	rc, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open pptx: %w", err)
	}
	defer rc.Close()

	var slideFiles []string
	for _, f := range rc.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideFiles = append(slideFiles, f.Name)
		}
	}
	sort.Strings(slideFiles)

	var buf strings.Builder
	for _, name := range slideFiles {
		for _, f := range rc.File {
			if f.Name == name {
				text, err := extractXMLText(f, "t")
				if err != nil {
					continue
				}
				buf.WriteString(text)
				buf.WriteByte(' ')
			}
		}
	}

	result := buf.String()
	if len(result) > maxTextLength {
		result = result[:maxTextLength]
	}
	return result, nil
}

func extractXMLText(f *zip.File, tagName string) (string, error) {
	r, err := f.Open()
	if err != nil {
		return "", err
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var buf strings.Builder
	var inTag bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == tagName {
				inTag = true
			}
		case xml.CharData:
			if inTag {
				buf.WriteString(string(t))
				buf.WriteByte(' ')
			}
		case xml.EndElement:
			if t.Name.Local == tagName {
				inTag = false
			}
		}
	}

	return buf.String(), nil
}
