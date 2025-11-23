package hwpv5

import (
	"compress/flate"
	"crypto/aes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/richardlehane/mscfb"
)

// Reader wraps an open HWP document.
type Reader struct {
	ra           io.ReaderAt
	Header       FileHeader
	sectionCount int
}

// OpenReader opens an HWP 5.0 file and returns a Reader.
func OpenReader(ra io.ReaderAt) (*Reader, error) {
	r := &Reader{ra: ra}

	headerStream, err := r.openStream("FileHeader")
	if err != nil {
		return nil, fmt.Errorf("failed to open FileHeader: %w", err)
	}
	r.Header, err = readFileHeader(headerStream)
	if err != nil {
		return nil, fmt.Errorf("failed to read FileHeader: %w", err)
	}

	if r.Header.Properties.Encrypted() {
		return nil, errors.New("password encrypted documents are not supported")
	}

	docInfoStream, err := r.openStream("DocInfo")
	if err != nil {
		return nil, fmt.Errorf("failed to open DocInfo: %w", err)
	}

	var currentReader io.Reader = docInfoStream
	if r.Header.Properties.Compressed() {
		currentReader = flate.NewReader(docInfoStream)
		defer currentReader.(io.Closer).Close()
	}

	scanner := NewRecScanner(currentReader)
	const HWPTAG_DOCUMENT_PROPERTIES = 0x10
	for {
		rec, err := scanner.ScanNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to scan DocInfo: %w", err)
		}

		if rec.Tag() == HWPTAG_DOCUMENT_PROPERTIES {
			if docProps, ok := rec.(RecUnknown); ok && len(docProps.Data) >= 2 {
				r.sectionCount = int(binary.LittleEndian.Uint16(docProps.Data[0:2]))
			}
			break
		}
	}

	if r.sectionCount == 0 {
		r.sectionCount = 1
	}

	return r, nil
}

// openStream opens a named stream from the OLE container.
func (r *Reader) openStream(name string) (io.Reader, error) {
	doc, err := mscfb.New(r.ra)
	if err != nil {
		return nil, err
	}

	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		fullPath := ""
		if len(entry.Path) > 0 {
			for _, p := range entry.Path {
				fullPath += p + "/"
			}
		}
		fullPath += entry.Name

		if fullPath == name {
			return doc, nil
		}
	}
	return nil, fmt.Errorf("stream %s not found", name)
}

// IsDistributionDoc returns true if this is a distribution document (uses ViewText).
func (r *Reader) IsDistributionDoc() bool {
	return r.Header.Properties.Raw&0x04 != 0
}

// SectionCount returns the number of sections in the document.
func (r *Reader) SectionCount() int {
	return r.sectionCount
}

// OpenSection opens a section stream by index.
// Returns a reader that handles decompression and decryption as needed.
func (r *Reader) OpenSection(index int) (io.ReadCloser, error) {
	var streamName string
	if r.IsDistributionDoc() {
		streamName = fmt.Sprintf("ViewText/Section%d", index)
	} else {
		streamName = fmt.Sprintf("BodyText/Section%d", index)
	}

	rawStream, err := r.openStream(streamName)
	if err != nil {
		return nil, err
	}

	var currentReader io.Reader = rawStream

	if r.IsDistributionDoc() {
		var hBuf [4]byte
		if _, err := io.ReadFull(currentReader, hBuf[:]); err != nil {
			return nil, fmt.Errorf("failed to read distribute doc header: %w", err)
		}
		tagVal := binary.LittleEndian.Uint32(hBuf[:])
		tagID := uint16(tagVal & 0x3FF)
		size := tagVal >> 20

		const HWPTAG_DISTRIBUTE_DOC_DATA = 0x1C
		if tagID == HWPTAG_DISTRIBUTE_DOC_DATA && size == 256 {
			distData := make([]byte, 256)
			if _, err := io.ReadFull(currentReader, distData); err != nil {
				return nil, fmt.Errorf("failed to read distribute doc data: %w", err)
			}

			key, err := deriveKey(distData)
			if err != nil {
				return nil, fmt.Errorf("failed to derive key: %w", err)
			}

			block, err := aes.NewCipher(key)
			if err != nil {
				return nil, fmt.Errorf("failed to create cipher: %w", err)
			}

			currentReader = &cryptoReader{r: currentReader, block: block}
		} else {
			return nil, fmt.Errorf("invalid distribution document stream (tag=0x%x, size=%d)", tagID, size)
		}
	}

	if r.Header.Properties.Compressed() {
		return flate.NewReader(currentReader), nil
	}

	return io.NopCloser(currentReader), nil
}
