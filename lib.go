// Package hwp provides functionality to read and render HWP (Hangul Word Processor) documents.
//
// This package supports both binary HWP v5 format (.hwp) and XML-based HWPX format (.hwpx).
// It extracts text content and renders tables with ASCII borders to plain text output.
//
// # Example Usage
//
//	file, err := os.Open("document.hwp")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer file.Close()
//
//	// Auto-detect format and render to stdout
//	if err := hwp.Read(file, os.Stdout); err != nil {
//		log.Fatal(err)
//	}
//
// # Supported Formats
//
// HWP v5 (.hwp): Binary format with OLE Compound File container
//   - Paragraph and table extraction
//   - AES-128 ECB decryption for distribution documents
//   - UTF-16LE text decoding
//
// HWPX (.hwpx): XML-based format with ZIP container
//   - OWPML (Open Word-processor Markup Language) parsing
//   - Full table support with cell merging
//   - Section-based document structure
package hwp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hanpama/hwp/internal/hwpv5"
	"github.com/hanpama/hwp/internal/hwpx"
	"github.com/hanpama/hwp/internal/render"
)

// ReadHWP reads a binary HWP v5 format file and renders its content as plain text.
//
// The input must be an *os.File because the HWP v5 format requires random access
// to read the OLE Compound File structure.
//
// Text is extracted from paragraphs and tables are rendered with ASCII borders.
// Images are represented as [IMAGE] placeholders.
//
// Example:
//
//	file, _ := os.Open("document.hwp")
//	defer file.Close()
//	hwp.ReadHWP(file, os.Stdout)
func ReadHWP(in io.Reader, out io.Writer) error {
	file, ok := in.(*os.File)
	if !ok {
		return fmt.Errorf("input must be an *os.File for HWP format")
	}

	scanner, err := hwpv5.Open(file)
	if err != nil {
		return fmt.Errorf("failed to parse HWP file: %w", err)
	}

	if err := render.RenderText(scanner, out); err != nil {
		return fmt.Errorf("failed to render HWP: %w", err)
	}

	return nil
}

// ReadHWPX reads an XML-based HWPX format file and renders its content as plain text.
//
// HWPX files are ZIP containers with XML content following the OWPML specification.
// The input must implement io.ReaderAt for ZIP extraction, and size must be the file size.
//
// This function parses section XML files, extracts paragraphs and tables with proper
// cell merging support, and renders them to plain text.
//
// Example:
//
//	file, _ := os.Open("document.hwpx")
//	defer file.Close()
//	info, _ := file.Stat()
//	hwp.ReadHWPX(file, info.Size(), os.Stdout)
func ReadHWPX(in io.ReaderAt, size int64, out io.Writer) error {
	reader, err := hwpx.Open(in, size)
	if err != nil {
		return fmt.Errorf("failed to parse HWPX file: %w", err)
	}

	scanner, err := reader.NewContentScanner()
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	if err := render.RenderText(scanner, out); err != nil {
		return fmt.Errorf("failed to render HWPX: %w", err)
	}

	return nil
}

// Read automatically detects the file format and renders the document to plain text.
//
// Format detection is based on the file extension:
//   - .hwpx → calls ReadHWPX
//   - .hwp or other → calls ReadHWP
//
// This is the recommended function for general use as it handles both formats seamlessly.
//
// Example:
//
//	file, _ := os.Open("document.hwp")  // or document.hwpx
//	defer file.Close()
//	hwp.Read(file, os.Stdout)
func Read(file *os.File, out io.Writer) error {
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(file.Name()))

	if ext == ".hwpx" {
		return ReadHWPX(file, fileInfo.Size(), out)
	}

	return ReadHWP(file, out)
}
